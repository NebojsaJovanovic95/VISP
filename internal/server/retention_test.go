package server

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/NebojsaJovanovic95/VISP/internal/config"
	"github.com/NebojsaJovanovic95/VISP/internal/database"
	"github.com/NebojsaJovanovic95/VISP/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRunImageRetention(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	connURL := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.Database.User, cfg.Database.Password),
		Host:   net.JoinHostPort(cfg.Database.Host, cfg.Database.Port),
		Path:   cfg.Database.Name,
	}

	query := url.Values{}
	query.Set("sslmode", cfg.Database.SSLMode)
	connURL.RawQuery = query.Encode()

	testPool, err := pgxpool.New(ctx, connURL.String())
	if err != nil {
		t.Fatalf("create test database connection: %v", err)
	}
	defer testPool.Close()

	if err := testPool.Ping(ctx); err != nil {
		t.Fatalf("ping test database: %v", err)
	}

	var cameraID string

	err = testPool.QueryRow(ctx, `
		INSERT INTO cameras (
			camera_name,
			rtsp_url,
			location,
			status,
			acquisition_interval
		)
		VALUES ($1, $2, $3, $4, INTERVAL '5 seconds')
		RETURNING camera_id::text
	`,
		"Image Retention Test",
		"rtsp://localhost/retention-test",
		"Test Location",
		"active",
	).Scan(&cameraID)
	if err != nil {
		t.Fatalf("insert test camera: %v", err)
	}

	defer func() {
		_, err := testPool.Exec(ctx,
			`DELETE FROM images WHERE camera_id = $1`,
			cameraID,
		)
		if err != nil {
			t.Errorf("delete test images: %v", err)
		}

		_, err = testPool.Exec(ctx,
			`DELETE FROM cameras WHERE camera_id = $1`,
			cameraID,
		)
		if err != nil {
			t.Errorf("delete test camera: %v", err)
		}
	}()

	// Prepare three images: expired, expired with a deletion failure,
	// and recent.
	type testImage struct {
		id       string
		key      string
		captured time.Time
	}

	now := time.Now().UTC()

	images := []testImage{
		{
			key:      "frames/retention-success.png",
			captured: now.Add(-48 * time.Hour),
		},
		{
			key:      "frames/retention-failure.png",
			captured: now.Add(-48 * time.Hour),
		},
		{
			key:      "frames/retention-recent.png",
			captured: now.Add(-1 * time.Hour),
		},
	}

	for i := range images {
		err := testPool.QueryRow(ctx, `
			INSERT INTO images (
				image_reference,
				camera_id,
				camera_snapshot,
				captured_at
			)
			VALUES ($1, $2, '{}'::jsonb, $3)
			RETURNING image_id::text
		`,
			images[i].key,
			cameraID,
			images[i].captured,
		).Scan(&images[i].id)
		if err != nil {
			t.Fatalf("insert test image %q: %v", images[i].key, err)
		}
	}

	// Simulate S3 object storage.
	objects := map[string]bool{
		images[0].key: true,
		images[1].key: true,
		images[2].key: true,
	}

	failedDeleteKey := images[1].key

	s3Server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimPrefix(r.URL.Path, "/visp/")

			switch r.Method {
			case http.MethodHead:
				w.WriteHeader(http.StatusOK)

			case http.MethodDelete:
				if key == failedDeleteKey {
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				delete(objects, key)
				w.WriteHeader(http.StatusNoContent)

			default:
				t.Errorf("unexpected HTTP method: %s", r.Method)
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		},
	))
	defer s3Server.Close()

	objectStorage, err := storage.New(ctx, config.ObjectStorageConfig{
		Endpoint:  s3Server.URL,
		Region:    "us-east-1",
		Bucket:    "visp",
		AccessKey: "test-access-key",
		SecretKey: "test-secret-key",
	})
	if err != nil {
		t.Fatalf("initialize storage: %v", err)
	}

	server := New(db, objectStorage, 24*time.Hour, time.Hour)

	if err := server.RunImageRetention(ctx); err != nil {
		t.Fatalf("run image retention: %v", err)
	}

	tests := []struct {
		name             string
		image            testImage
		wantObjectExists bool
		wantAvailable    bool
	}{
		{
			name:             "expired image is deleted",
			image:            images[0],
			wantObjectExists: false,
			wantAvailable:    false,
		},
		{
			name:             "deletion failure preserves availability",
			image:            images[1],
			wantObjectExists: true,
			wantAvailable:    true,
		},
		{
			name:             "recent image is retained",
			image:            images[2],
			wantObjectExists: true,
			wantAvailable:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := objects[tt.image.key]; got != tt.wantObjectExists {
				t.Errorf("expected object existence %t, got %t",
					tt.wantObjectExists, got)
			}

			var (
				gotReference string
				gotAvailable bool
			)

			err := testPool.QueryRow(ctx, `
				SELECT image_reference, file_available
				FROM images
				WHERE image_id = $1
			`, tt.image.id).Scan(&gotReference, &gotAvailable)
			if err != nil {
				t.Fatalf("query image metadata: %v", err)
			}

			if gotReference != tt.image.key {
				t.Errorf("expected image reference %q, got %q",
					tt.image.key, gotReference)
			}

			if gotAvailable != tt.wantAvailable {
				t.Errorf("expected file_available %t, got %t",
					tt.wantAvailable, gotAvailable)
			}
		})
	}

	t.Run("scheduler runs periodically and stops on cancellation", func(t *testing.T) {
		// Create an image that is initially too recent for retention.
		key := "frames/retention-scheduler.png"

		var imageID string

		err := testPool.QueryRow(ctx, `
			INSERT INTO images (
				image_reference,
				camera_id,
				camera_snapshot,
				captured_at
			)
			VALUES ($1, $2, '{}'::jsonb, $3)
			RETURNING image_id::text
		`,
			key,
			cameraID,
			time.Now().UTC(),
		).Scan(&imageID)
		if err != nil {
			t.Fatalf("insert scheduler test image: %v", err)
		}

		objects[key] = true

		schedulerCtx, stopScheduler := context.WithCancel(ctx)
		done := make(chan struct{})

		// Use a short interval so the test doesn't have to wait an hour.
		scheduler := New(db, objectStorage, 24*time.Hour, 100*time.Millisecond)

		go func() {
			defer close(done)
			scheduler.RunImageRetentionScheduler(schedulerCtx)
		}()

		// Allow the scheduler's initial retention run to complete.
		time.Sleep(25 * time.Millisecond)

		var available bool

		err = testPool.QueryRow(ctx, `
			SELECT file_available
			FROM images
			WHERE image_id = $1
		`, imageID).Scan(&available)
		if err != nil {
			stopScheduler()
			<-done
			t.Fatalf("check initial image availability: %v", err)
		}

		if !available {
			stopScheduler()
			<-done
			t.Fatal("scheduler deleted an image before it expired")
		}

		// Make the image old enough for retention.
		_, err = testPool.Exec(ctx, `
			UPDATE images
			SET captured_at = $1
			WHERE image_id = $2
		`, time.Now().UTC().Add(-48*time.Hour), imageID)
		if err != nil {
			stopScheduler()
			<-done
			t.Fatalf("expire scheduler test image: %v", err)
		}

		// Wait for the next scheduled retention run.
		deadline := time.Now().Add(2 * time.Second)

		for time.Now().Before(deadline) {
			err = testPool.QueryRow(ctx, `
				SELECT file_available
				FROM images
				WHERE image_id = $1
			`, imageID).Scan(&available)
			if err != nil {
				stopScheduler()
				<-done
				t.Fatalf("check scheduled retention result: %v", err)
			}

			if !available {
				break
			}

			time.Sleep(10 * time.Millisecond)
		}

		stopScheduler()
		<-done

		if available {
			t.Fatal("scheduler did not delete the expired image")
		}

		if objects[key] {
			t.Error("expected expired image file to be deleted from storage")
		}
	})
}
