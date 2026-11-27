package server

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
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
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestReceiveFrameHandler(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cfg, err := config.LoadServer()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	// Create the database connection used by the server.
	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Open a separate connection for test fixture setup and verification.
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

	// Create a camera for the frame submission.
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
		"Frame Handler Test",
		"rtsp://localhost/test",
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

	// Store objects by their S3 object key.
	type storedObject struct {
		contentType string
		body        []byte
	}

	objects := make(map[string]storedObject)

	// Simulate the S3-compatible storage service.
	s3Server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := strings.TrimPrefix(r.URL.Path, "/visp/")

			switch r.Method {
			case http.MethodHead:
				if r.URL.Path != "/visp" {
					t.Errorf("expected bucket path %q, got %q", "/visp", r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)

			case http.MethodPut:
				body, err := io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read uploaded image: %v", err)
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				objects[key] = storedObject{
					contentType: r.Header.Get("Content-Type"),
					body:        body,
				}

				w.WriteHeader(http.StatusOK)

			case http.MethodGet:
				object, exists := objects[key]
				if !exists {
					w.WriteHeader(http.StatusNotFound)
					return
				}

				w.Header().Set("Content-Type", object.contentType)
				w.WriteHeader(http.StatusOK)

				if _, err := w.Write(object.body); err != nil {
					t.Errorf("write downloaded image: %v", err)
				}

			case http.MethodDelete:
				delete(objects, key)
				w.WriteHeader(http.StatusNoContent)

			default:
				t.Errorf("unexpected HTTP method: %s", r.Method)
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		},
	))
	defer s3Server.Close()

	objectStorage, err := storage.New(
		ctx,
		config.ObjectStorageConfig{
			Endpoint:  s3Server.URL,
			Region:    "us-east-1",
			Bucket:    "visp",
			AccessKey: "test-access-key",
			SecretKey: "test-secret-key",
		},
	)
	if err != nil {
		t.Fatalf("initialize storage: %v", err)
	}

	server := New(
		db,
		objectStorage,
		cfg.ImageRetentionPeriod,
		cfg.ImageRetentionCheckInterval,
	)

	// Construct the frame submission request.
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	metadata := frameMetadata{
		CameraID:   cameraID,
		CapturedAt: time.Date(2026, 11, 20, 18, 30, 0, 0, time.UTC),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	if err := writer.WriteField("metadata", string(metadataJSON)); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	imagePart, err := writer.CreateFormFile("image", "frame.png")
	if err != nil {
		t.Fatalf("create image part: %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})

	if err := png.Encode(imagePart, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/frames", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s",
			http.StatusCreated, rec.Code, rec.Body.String())
	}

	var response struct {
		ImageID string `json:"image_id"`
	}

	if err := json.NewDecoder(rec.Body).Decode(&response); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if response.ImageID == "" {
		t.Fatal("expected image_id in response")
	}

	// Verify the object-storage upload.
	expectedKey := "frames/" + response.ImageID + ".png"

	uploadedObject, exists := objects[expectedKey]
	if !exists {
		t.Fatalf("expected object %q to exist in storage", expectedKey)
	}

	if uploadedObject.contentType != "image/png" {
		t.Errorf("expected content type %q, got %q",
			"image/png", uploadedObject.contentType)
	}

	uploadedImage := uploadedObject.body

	imageConfig, err := png.DecodeConfig(bytes.NewReader(uploadedImage))
	if err != nil {
		t.Fatalf("decode uploaded PNG: %v", err)
	}

	if imageConfig.Width != 2 || imageConfig.Height != 2 {
		t.Errorf("expected image dimensions 2x2, got %dx%d",
			imageConfig.Width, imageConfig.Height)
	}

	// Verify that the image metadata was persisted in PostgreSQL.
	var (
		gotReference     string
		gotCameraID      string
		gotCaptured      time.Time
		gotFileAvailable bool
	)

	err = testPool.QueryRow(ctx, `
		SELECT image_reference, camera_id::text, captured_at, file_available
		FROM images
		WHERE image_id = $1
	`, response.ImageID).Scan(
		&gotReference,
		&gotCameraID,
		&gotCaptured,
		&gotFileAvailable,
	)
	if err != nil {
		t.Fatalf("query persisted image: %v", err)
	}

	if gotReference != expectedKey {
		t.Errorf("expected image reference %q, got %q",
			expectedKey, gotReference)
	}

	if gotCameraID != cameraID {
		t.Errorf("expected camera ID %q, got %q",
			cameraID, gotCameraID)
	}

	if !gotFileAvailable {
		t.Error("expected image file to be available after upload")
	}

	// Verify that the image can be retrieved.
	imageReq := httptest.NewRequest(
		http.MethodGet,
		"/api/images/"+response.ImageID,
		nil,
	)

	imageRec := httptest.NewRecorder()
	server.Router().ServeHTTP(imageRec, imageReq)

	if imageRec.Code != http.StatusOK {
		t.Fatalf("expected status %d when retrieving image, got %d: %s",
			http.StatusOK, imageRec.Code, imageRec.Body.String())
	}

	if got := imageRec.Header().Get("Content-Type"); got != "image/png" {
		t.Errorf("expected content type %q, got %q", "image/png", got)
	}

	if !bytes.Equal(imageRec.Body.Bytes(), uploadedImage) {
		t.Error("retrieved image does not match uploaded image")
	}

	// Mark the physical image as unavailable while retaining its metadata.
	commandTag, err := testPool.Exec(ctx, `
		UPDATE images
		SET file_available = FALSE
		WHERE image_id = $1
	`, response.ImageID)
	if err != nil {
		t.Fatalf("mark image unavailable: %v", err)
	}
	if commandTag.RowsAffected() != 1 {
		t.Fatalf("expected to update one image, updated %d", commandTag.RowsAffected())
	}

	unavailableReq := httptest.NewRequest(
		http.MethodGet,
		"/api/images/"+response.ImageID,
		nil,
	)

	unavailableRec := httptest.NewRecorder()
	server.Router().ServeHTTP(unavailableRec, unavailableReq)

	if unavailableRec.Code != http.StatusGone {
		t.Errorf("expected status %d for unavailable image, got %d: %s",
			http.StatusGone, unavailableRec.Code, unavailableRec.Body.String())
	}

	missingImageID := uuid.NewString()

	missingReq := httptest.NewRequest(
		http.MethodGet,
		"/api/images/"+missingImageID,
		nil,
	)

	missingRec := httptest.NewRecorder()
	server.Router().ServeHTTP(missingRec, missingReq)

	if missingRec.Code != http.StatusNotFound {
		t.Errorf(
			"expected status %d for missing image, got %d",
			http.StatusNotFound,
			missingRec.Code,
		)
	}
}

func TestReceiveFrameHandlerInvalidMultipart(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/frames",
		bytes.NewBufferString("invalid request"),
	)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}

func TestReceiveFrameHandlerInvalidMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata string
	}{
		{
			name:     "missing metadata",
			metadata: "",
		},
		{
			name:     "invalid JSON",
			metadata: "{invalid json}",
		},
		{
			name:     "missing camera ID",
			metadata: `{"captured_at":"2026-11-20T18:30:00Z"}`,
		},
		{
			name:     "missing capture timestamp",
			metadata: `{"camera_id":"camera-123"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := &Server{}

			var body bytes.Buffer
			writer := multipart.NewWriter(&body)

			if tt.metadata != "" {
				if err := writer.WriteField("metadata", tt.metadata); err != nil {
					t.Fatalf("write metadata: %v", err)
				}
			}

			if err := writer.Close(); err != nil {
				t.Fatalf("close multipart writer: %v", err)
			}

			req := httptest.NewRequest(
				http.MethodPost,
				"/api/frames",
				&body,
			)
			req.Header.Set("Content-Type", writer.FormDataContentType())

			rec := httptest.NewRecorder()
			server.Router().ServeHTTP(rec, req)

			if rec.Code != http.StatusBadRequest {
				t.Errorf(
					"expected status %d, got %d",
					http.StatusBadRequest,
					rec.Code,
				)
			}
		})
	}
}

func TestReceiveFrameHandlerMissingImage(t *testing.T) {
	server := &Server{}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	metadata := frameMetadata{
		CameraID:   "camera-123",
		CapturedAt: time.Date(2026, 11, 20, 18, 30, 0, 0, time.UTC),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	if err := writer.WriteField("metadata", string(metadataJSON)); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/frames", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s",
			http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestReceiveFrameHandlerInvalidImage(t *testing.T) {
	server := &Server{}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	metadata := frameMetadata{
		CameraID:   "camera-123",
		CapturedAt: time.Date(2026, 11, 20, 18, 30, 0, 0, time.UTC),
	}

	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		t.Fatalf("marshal metadata: %v", err)
	}

	if err := writer.WriteField("metadata", string(metadataJSON)); err != nil {
		t.Fatalf("write metadata: %v", err)
	}

	imagePart, err := writer.CreateFormFile("image", "frame.png")
	if err != nil {
		t.Fatalf("create image part: %v", err)
	}

	if _, err := imagePart.Write([]byte("not a PNG image")); err != nil {
		t.Fatalf("write image: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/frames", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d: %s",
			http.StatusBadRequest, rec.Code, rec.Body.String())
	}
}

func TestGetImageHandlerInvalidID(t *testing.T) {
	server := &Server{}

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/images/invalid-id",
		nil,
	)

	rec := httptest.NewRecorder()
	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf(
			"expected status %d, got %d",
			http.StatusBadRequest,
			rec.Code,
		)
	}
}
