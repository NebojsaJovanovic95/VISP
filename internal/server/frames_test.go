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
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NebojsaJovanovic95/VISP/internal/config"
	"github.com/NebojsaJovanovic95/VISP/internal/storage"
)

func TestReceiveFrameHandler(t *testing.T) {
	var uploadedPath string
	var uploadedContentType string
	var uploadedImage []byte

	s3Server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodHead:
				if r.URL.Path != "/visp" {
					t.Errorf("expected bucket path %q, got %q", "/visp", r.URL.Path)
				}
				w.WriteHeader(http.StatusOK)

			case http.MethodPut:
				uploadedPath = r.URL.Path
				uploadedContentType = r.Header.Get("Content-Type")

				var err error
				uploadedImage, err = io.ReadAll(r.Body)
				if err != nil {
					t.Errorf("read uploaded image: %v", err)
				}

				w.WriteHeader(http.StatusOK)

			default:
				t.Errorf("unexpected HTTP method: %s", r.Method)
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
		},
	))
	defer s3Server.Close()

	objectStorage, err := storage.New(
		context.Background(),
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

	server := &Server{
		storage: objectStorage,
	}

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

	expectedKey := "frames/" + response.ImageID + ".png"

	if uploadedPath != "/visp/"+expectedKey {
		t.Errorf("expected uploaded path %q, got %q",
			"/visp/"+expectedKey, uploadedPath)
	}

	if uploadedContentType != "image/png" {
		t.Errorf("expected content type %q, got %q",
			"image/png", uploadedContentType)
	}

	imageConfig, err := png.DecodeConfig(bytes.NewReader(uploadedImage))
	if err != nil {
		t.Fatalf("decode uploaded PNG: %v", err)
	}

	if imageConfig.Width != 2 || imageConfig.Height != 2 {
		t.Errorf("expected image dimensions 2x2, got %dx%d",
			imageConfig.Width, imageConfig.Height)
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
