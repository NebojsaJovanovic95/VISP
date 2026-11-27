package database

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/NebojsaJovanovic95/VISP/internal/config"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

func TestNew(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}
	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()
}

func withTransaction(t *testing.T, db *Database, test func(context.Context, pgx.Tx)) {
	t.Helper()

	ctx := context.Background()

	tx, err := db.pool.Begin(ctx)
	if err != nil {
		t.Fatalf("begin transaction: %v", err)
	}

	defer func() {
		if err := tx.Rollback(ctx); err != nil {
			t.Errorf("rollback transaction: %v", err)
		}
	}()

	test(ctx, tx)
}

func createTestInferenceResult(t *testing.T, ctx context.Context, tx pgx.Tx) string {
	t.Helper()

	var cameraID, modelID, subscriptionID string
	var cameraSubscriptionID, imageID, taskID, resultID string

	err := tx.QueryRow(ctx, `
        INSERT INTO cameras (camera_name, rtsp_url, status, acquisition_interval)
        VALUES ('Test Camera', 'rtsp://localhost/test', 'active', INTERVAL '5 seconds')
        RETURNING camera_id
    `).Scan(&cameraID)
	if err != nil {
		t.Fatalf("insert camera: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO models (model_name, model_reference, model_type, version, status)
        VALUES ('Test Model', 'test-model', 'object-detection', '1.0', 'active')
        RETURNING model_id
    `).Scan(&modelID)
	if err != nil {
		t.Fatalf("insert model: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO subscriptions (
            subscription_name, model_id, inference_schedule, saving_policy, status
        )
        VALUES ('Test Subscription', $1, '{}', '{}', 'active')
        RETURNING subscription_id
    `, modelID).Scan(&subscriptionID)
	if err != nil {
		t.Fatalf("insert subscription: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO camera_subscriptions (camera_id, subscription_id, status)
        VALUES ($1, $2, 'active')
        RETURNING camera_subscription_id
    `, cameraID, subscriptionID).Scan(&cameraSubscriptionID)
	if err != nil {
		t.Fatalf("insert camera subscription: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO images (image_reference, camera_id, camera_snapshot, captured_at)
        VALUES ('test-image', $1, '{}', NOW())
        RETURNING image_id
    `, cameraID).Scan(&imageID)
	if err != nil {
		t.Fatalf("insert image: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO inference_tasks (
            image_id, camera_subscription_id, model_id,
            subscription_snapshot, model_snapshot, status
        )
        VALUES ($1, $2, $3, '{}', '{}', 'completed')
        RETURNING task_id
    `, imageID, cameraSubscriptionID, modelID).Scan(&taskID)
	if err != nil {
		t.Fatalf("insert inference task: %v", err)
	}

	err = tx.QueryRow(ctx, `
        INSERT INTO inference_results (
            task_id, status, completed_at, detection_count
        )
        VALUES ($1, 'completed', NOW(), 1)
        RETURNING result_id
    `, taskID).Scan(&resultID)
	if err != nil {
		t.Fatalf("insert inference result: %v", err)
	}

	return resultID
}

func TestImageRequiresExistingCamera(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	withTransaction(t, db, func(ctx context.Context, tx pgx.Tx) {
		_, err := tx.Exec(ctx, `
            INSERT INTO images (
                image_reference,
                camera_id,
                camera_snapshot,
                captured_at
            )
            VALUES ($1, $2, $3, NOW())
        `,
			"test-image",
			"00000000-0000-0000-0000-000000000001",
			"{}",
		)

		if err == nil {
			t.Fatal("expected foreign key violation, got nil")
		}

		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) {
			t.Fatalf("expected PostgreSQL error, got: %v", err)
		}

		if pgErr.Code != "23503" {
			t.Errorf("expected foreign key violation (23503), got %s", pgErr.Code)
		}
	})
}

func TestCameraSubscriptionMustBeUnique(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	withTransaction(t, db, func(ctx context.Context, tx pgx.Tx) {
		cameraID := "00000000-0000-0000-0000-000000000001"
		modelID := "00000000-0000-0000-0000-000000000002"
		subscriptionID := "00000000-0000-0000-0000-000000000003"

		_, err := tx.Exec(ctx, `
            INSERT INTO cameras (
                camera_id, camera_name, rtsp_url, status, acquisition_interval
            )
            VALUES ($1, $2, $3, $4, INTERVAL '5 seconds')
        `, cameraID, "Test Camera", "rtsp://localhost/test", "active")
		if err != nil {
			t.Fatalf("insert test camera: %v", err)
		}

		_, err = tx.Exec(ctx, `
            INSERT INTO models (
                model_id, model_name, model_reference, model_type, version, status
            )
            VALUES ($1, $2, $3, $4, $5, $6)
        `, modelID, "Test Model", "test-model", "object-detection", "1.0", "active")
		if err != nil {
			t.Fatalf("insert test model: %v", err)
		}

		_, err = tx.Exec(ctx, `
            INSERT INTO subscriptions (
                subscription_id, subscription_name, model_id,
                inference_schedule, saving_policy, status
            )
            VALUES ($1, $2, $3, $4, $5, $6)
        `, subscriptionID, "Test Subscription", modelID, `{}`, `{}`, "active")
		if err != nil {
			t.Fatalf("insert test subscription: %v", err)
		}

		for i := range 2 {
			_, err = tx.Exec(ctx, `
                INSERT INTO camera_subscriptions (
                    camera_id, subscription_id, status
                )
                VALUES ($1, $2, $3)
            `, cameraID, subscriptionID, "active")

			if i == 0 && err != nil {
				t.Fatalf("insert first camera subscription: %v", err)
			}

			if i == 1 {
				if err == nil {
					t.Fatal("expected unique constraint violation, got nil")
				}

				var pgErr *pgconn.PgError
				if !errors.As(err, &pgErr) {
					t.Fatalf("expected PostgreSQL error, got: %v", err)
				}

				if pgErr.Code != "23505" {
					t.Errorf("expected unique violation (23505), got %s", pgErr.Code)
				}
			}
		}
	})
}

func TestCameraAcquisitionIntervalMustBePositive(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	withTransaction(t, db, func(ctx context.Context, tx pgx.Tx) {
		_, err := tx.Exec(ctx, `
            INSERT INTO cameras (
                camera_name,
                rtsp_url,
                status,
                acquisition_interval
            )
            VALUES ($1, $2, $3, INTERVAL '0 seconds')
        `, "Test Camera", "rtsp://localhost/test", "active")

		if err == nil {
			t.Fatal("expected check constraint violation, got nil")
		}

		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) {
			t.Fatalf("expected PostgreSQL error, got: %v", err)
		}

		if pgErr.Code != "23514" {
			t.Errorf("expected check violation (23514), got %s", pgErr.Code)
		}
	})
}

func TestDetectionConfidenceMustBeValid(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	withTransaction(t, db, func(ctx context.Context, tx pgx.Tx) {
		resultID := createTestInferenceResult(t, ctx, tx)

		_, err := tx.Exec(ctx, `
            INSERT INTO detections (
                result_id, class_id, class_name, confidence,
                x_min, y_min, width, height
            )
            VALUES ($1, 0, 'person', 1.5, 0.1, 0.1, 0.2, 0.2)
        `, resultID)

		if err == nil {
			t.Fatal("expected check constraint violation, got nil")
		}

		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) {
			t.Fatalf("expected PostgreSQL error, got: %v", err)
		}

		if pgErr.Code != "23514" {
			t.Errorf("expected check violation (23514), got %s", pgErr.Code)
		}
	})
}

func TestDetectionBoundingBoxMustBeWithinImage(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	withTransaction(t, db, func(ctx context.Context, tx pgx.Tx) {
		resultID := createTestInferenceResult(t, ctx, tx)

		_, err := tx.Exec(ctx, `
            INSERT INTO detections (
                result_id, class_id, class_name, confidence,
                x_min, y_min, width, height
            )
            VALUES ($1, 0, 'person', 0.95, 0.8, 0.1, 0.3, 0.2)
        `, resultID)

		if err == nil {
			t.Fatal("expected check constraint violation, got nil")
		}

		var pgErr *pgconn.PgError
		if !errors.As(err, &pgErr) {
			t.Fatalf("expected PostgreSQL error, got: %v", err)
		}

		if pgErr.Code != "23514" {
			t.Errorf("expected check violation (23514), got %s", pgErr.Code)
		}
	})
}

func TestListCameras(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	var cameraID string

	err = db.pool.QueryRow(ctx, `
		INSERT INTO cameras (
			camera_name,
			rtsp_url,
			location,
			status,
			acquisition_interval
		)
		VALUES ($1, $2, $3, $4, INTERVAL '7 seconds')
		RETURNING camera_id::text
	`,
		"ListCameras Test",
		"rtsp://localhost/test",
		"Test Location",
		"active",
	).Scan(&cameraID)
	if err != nil {
		t.Fatalf("insert test camera: %v", err)
	}

	defer func() {
		_, err := db.pool.Exec(ctx,
			`DELETE FROM cameras WHERE camera_id = $1`,
			cameraID,
		)
		if err != nil {
			t.Errorf("delete test camera: %v", err)
		}
	}()

	cameras, err := db.ListCameras(ctx)
	if err != nil {
		t.Fatalf("list cameras: %v", err)
	}

	var found *CameraConfig

	for i := range cameras {
		if cameras[i].CameraID == cameraID {
			found = &cameras[i]
			break
		}
	}

	if found == nil {
		t.Fatal("expected test camera in camera list")
	}

	if found.CameraName != "ListCameras Test" {
		t.Errorf("expected camera name %q, got %q",
			"ListCameras Test", found.CameraName)
	}

	if found.RTSPURL != "rtsp://localhost/test" {
		t.Errorf("unexpected RTSP URL: %q", found.RTSPURL)
	}

	if found.Location == nil || *found.Location != "Test Location" {
		t.Errorf("unexpected location: %v", found.Location)
	}

	if found.Status != "active" {
		t.Errorf("expected status %q, got %q", "active", found.Status)
	}

	if found.AcquisitionIntervalSeconds != 7 {
		t.Errorf("expected acquisition interval 7 seconds, got %f",
			found.AcquisitionIntervalSeconds)
	}
}

func TestCameraExists(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	var cameraID string
	err = db.pool.QueryRow(ctx, `
		INSERT INTO cameras (
			camera_name,
			rtsp_url,
			status,
			acquisition_interval
		)
		VALUES ($1, $2, $3, INTERVAL '5 seconds')
		RETURNING camera_id::text
	`, "CameraExists Test", "rtsp://localhost/test", "active").Scan(&cameraID)
	if err != nil {
		t.Fatalf("insert test camera: %v", err)
	}

	defer func() {
		if _, err := db.pool.Exec(ctx, `DELETE FROM cameras WHERE camera_id = $1`, cameraID); err != nil {
			t.Errorf("delete test camera: %v", err)
		}
	}()

	exists, err := db.CameraExists(ctx, cameraID)
	if err != nil {
		t.Fatalf("check existing camera: %v", err)
	}
	if !exists {
		t.Error("expected camera to exist")
	}

	exists, err = db.CameraExists(ctx, "00000000-0000-0000-0000-000000000001")
	if err != nil {
		t.Fatalf("check nonexistent camera: %v", err)
	}
	if exists {
		t.Error("expected camera not to exist")
	}
}

func TestCreateImage(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	var cameraID string

	err = db.pool.QueryRow(ctx, `
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
		"CreateImage Test",
		"rtsp://localhost/test",
		"Test Location",
		"active",
	).Scan(&cameraID)
	if err != nil {
		t.Fatalf("insert test camera: %v", err)
	}

	imageID := uuid.NewString()
	imageReference := "frames/" + imageID + ".png"
	capturedAt := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)

	defer func() {
		_, err := db.pool.Exec(ctx,
			`DELETE FROM images WHERE image_id = $1`,
			imageID,
		)
		if err != nil {
			t.Errorf("delete test image: %v", err)
		}

		_, err = db.pool.Exec(ctx,
			`DELETE FROM cameras WHERE camera_id = $1`,
			cameraID,
		)
		if err != nil {
			t.Errorf("delete test camera: %v", err)
		}
	}()

	err = db.CreateImage(ctx, imageID, imageReference, cameraID, capturedAt)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}

	var (
		gotReference     string
		gotCameraID      string
		gotSnapshot      []byte
		gotCaptured      time.Time
		gotFileAvailable bool
	)

	err = db.pool.QueryRow(ctx, `
		SELECT
			image_reference,
			camera_id::text,
			camera_snapshot,
			captured_at,
			file_available
		FROM images
		WHERE image_id = $1
	`, imageID).Scan(
		&gotReference,
		&gotCameraID,
		&gotSnapshot,
		&gotCaptured,
		&gotFileAvailable,
	)
	if err != nil {
		t.Fatalf("query created image: %v", err)
	}

	if gotReference != imageReference {
		t.Errorf("expected image reference %q, got %q", imageReference, gotReference)
	}

	if gotCameraID != cameraID {
		t.Errorf("expected camera ID %q, got %q", cameraID, gotCameraID)
	}

	if !gotCaptured.Equal(capturedAt) {
		t.Errorf("expected captured_at %v, got %v", capturedAt, gotCaptured)
	}

	if !gotFileAvailable {
		t.Error("expected file_available to default to true")
	}

	var snapshot map[string]any
	if err := json.Unmarshal(gotSnapshot, &snapshot); err != nil {
		t.Fatalf("decode camera snapshot: %v", err)
	}

	expectedSnapshot := map[string]string{
		"camera_name": "CreateImage Test",
		"rtsp_url":    "rtsp://localhost/test",
		"location":    "Test Location",
		"status":      "active",
	}

	for key, expected := range expectedSnapshot {
		if snapshot[key] != expected {
			t.Errorf("expected snapshot %s=%q, got %v", key, expected, snapshot[key])
		}
	}
}

func TestCreateImageRequiresExistingCamera(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	imageID := uuid.NewString()
	cameraID := uuid.NewString()

	err = db.CreateImage(
		ctx,
		imageID,
		"frames/"+imageID+".png",
		cameraID,
		time.Now(),
	)
	if err == nil {
		t.Fatal("expected an error for a nonexistent camera")
	}
}

func TestGetImage(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	var cameraID string

	err = db.pool.QueryRow(ctx, `
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
		"GetImage Test",
		"rtsp://localhost/test",
		"Test Location",
		"active",
	).Scan(&cameraID)
	if err != nil {
		t.Fatalf("insert test camera: %v", err)
	}

	imageID := uuid.NewString()
	imageReference := "frames/" + imageID + ".png"
	capturedAt := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)

	defer func() {
		_, err := db.pool.Exec(ctx,
			`DELETE FROM images WHERE image_id = $1`,
			imageID,
		)
		if err != nil {
			t.Errorf("delete test image: %v", err)
		}

		_, err = db.pool.Exec(ctx,
			`DELETE FROM cameras WHERE camera_id = $1`,
			cameraID,
		)
		if err != nil {
			t.Errorf("delete test camera: %v", err)
		}
	}()

	err = db.CreateImage(ctx, imageID, imageReference, cameraID, capturedAt)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}

	image, err := db.GetImage(ctx, imageID)
	if err != nil {
		t.Fatalf("get image: %v", err)
	}

	if image.ImageID != imageID {
		t.Errorf("expected image ID %q, got %q", imageID, image.ImageID)
	}

	if image.ImageReference != imageReference {
		t.Errorf(
			"expected image reference %q, got %q",
			imageReference,
			image.ImageReference,
		)
	}

	if image.CameraID != cameraID {
		t.Errorf("expected camera ID %q, got %q", cameraID, image.CameraID)
	}

	if !image.FileAvailable {
		t.Error("expected image file to be available after creation")
	}

	if !image.CapturedAt.Equal(capturedAt) {
		t.Errorf("expected captured_at %v, got %v", capturedAt, image.CapturedAt)
	}

	var snapshot map[string]any
	if err := json.Unmarshal(image.CameraSnapshot, &snapshot); err != nil {
		t.Fatalf("decode camera snapshot: %v", err)
	}

	if snapshot["camera_name"] != "GetImage Test" {
		t.Errorf("unexpected camera snapshot: %v", snapshot)
	}
}

func TestGetImageNotFound(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	imageID := uuid.NewString()

	_, err = db.GetImage(ctx, imageID)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected pgx.ErrNoRows, got %v", err)
	}
}

func TestMarkImageUnavailable(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	var cameraID string

	err = db.pool.QueryRow(ctx, `
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
		"MarkImageUnavailable Test",
		"rtsp://localhost/test",
		"Test Location",
		"active",
	).Scan(&cameraID)
	if err != nil {
		t.Fatalf("insert test camera: %v", err)
	}

	imageID := uuid.NewString()
	imageReference := "frames/" + imageID + ".png"
	capturedAt := time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

	defer func() {
		_, err := db.pool.Exec(ctx,
			`DELETE FROM images WHERE image_id = $1`,
			imageID,
		)
		if err != nil {
			t.Errorf("delete test image: %v", err)
		}

		_, err = db.pool.Exec(ctx,
			`DELETE FROM cameras WHERE camera_id = $1`,
			cameraID,
		)
		if err != nil {
			t.Errorf("delete test camera: %v", err)
		}
	}()

	err = db.CreateImage(ctx, imageID, imageReference, cameraID, capturedAt)
	if err != nil {
		t.Fatalf("create image: %v", err)
	}

	err = db.MarkImageUnavailable(ctx, imageID)
	if err != nil {
		t.Fatalf("mark image unavailable: %v", err)
	}

	image, err := db.GetImage(ctx, imageID)
	if err != nil {
		t.Fatalf("get image: %v", err)
	}

	if image.FileAvailable {
		t.Error("expected image file to be unavailable")
	}

	if image.ImageID != imageID {
		t.Errorf("expected image ID %q, got %q", imageID, image.ImageID)
	}

	if image.ImageReference != imageReference {
		t.Errorf(
			"expected image reference %q, got %q",
			imageReference,
			image.ImageReference,
		)
	}

	if image.CameraID != cameraID {
		t.Errorf("expected camera ID %q, got %q", cameraID, image.CameraID)
	}

	if !image.CapturedAt.Equal(capturedAt) {
		t.Errorf("expected captured_at %v, got %v", capturedAt, image.CapturedAt)
	}
}

func TestListImagesEligibleForRetention(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	var cameraID string

	err = db.pool.QueryRow(ctx, `
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
		"ListImagesEligibleForRetention Test",
		"rtsp://localhost/test",
		"Test Location",
		"active",
	).Scan(&cameraID)
	if err != nil {
		t.Fatalf("insert test camera: %v", err)
	}

	cutoff := time.Now().UTC().Add(-24 * time.Hour)

	oldImageID := uuid.NewString()
	recentImageID := uuid.NewString()
	unavailableImageID := uuid.NewString()

	defer func() {
		_, err := db.pool.Exec(ctx, `
			DELETE FROM images
			WHERE image_id = ANY($1::uuid[])
		`, []string{oldImageID, recentImageID, unavailableImageID})
		if err != nil {
			t.Errorf("delete test images: %v", err)
		}

		_, err = db.pool.Exec(ctx,
			`DELETE FROM cameras WHERE camera_id = $1`,
			cameraID,
		)
		if err != nil {
			t.Errorf("delete test camera: %v", err)
		}
	}()

	tests := []struct {
		imageID    string
		capturedAt time.Time
	}{
		{oldImageID, cutoff.Add(-time.Hour)},
		{recentImageID, cutoff.Add(time.Hour)},
		{unavailableImageID, cutoff.Add(-2 * time.Hour)},
	}

	for _, test := range tests {
		err := db.CreateImage(
			ctx,
			test.imageID,
			"frames/"+test.imageID+".png",
			cameraID,
			test.capturedAt,
		)
		if err != nil {
			t.Fatalf("create test image %q: %v", test.imageID, err)
		}
	}

	err = db.MarkImageUnavailable(ctx, unavailableImageID)
	if err != nil {
		t.Fatalf("mark image unavailable: %v", err)
	}

	images, err := db.ListImagesEligibleForRetention(ctx, cutoff)
	if err != nil {
		t.Fatalf("list images eligible for retention: %v", err)
	}

	if len(images) != 1 {
		t.Fatalf("expected 1 eligible image, got %d", len(images))
	}

	image := images[0]

	if image.ImageID != oldImageID {
		t.Errorf("expected image ID %q, got %q", oldImageID, image.ImageID)
	}

	if !image.FileAvailable {
		t.Error("expected eligible image file to be available")
	}

	if !image.CapturedAt.Equal(cutoff.Add(-time.Hour)) {
		t.Errorf(
			"expected captured_at %v, got %v",
			cutoff.Add(-time.Hour),
			image.CapturedAt,
		)
	}

	if image.ImageReference != "frames/"+oldImageID+".png" {
		t.Errorf("unexpected image reference: %q", image.ImageReference)
	}
}
