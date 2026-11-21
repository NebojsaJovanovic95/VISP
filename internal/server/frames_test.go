package server

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReceiveFrameHandler(t *testing.T) {
	frameDir := t.TempDir()

	server := &Server{
		frameDir: frameDir,
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

	filename := filepath.Join(frameDir, "frame-"+response.ImageID+".png")

	savedImage, err := os.Open(filename)
	if err != nil {
		t.Fatalf("open saved image: %v", err)
	}
	defer savedImage.Close()

	config, err := png.DecodeConfig(savedImage)
	if err != nil {
		t.Fatalf("decode saved PNG: %v", err)
	}

	if config.Width != 2 || config.Height != 2 {
		t.Errorf("expected image dimensions 2x2, got %dx%d",
			config.Width, config.Height)
	}
}

func TestReceiveFrameHandlerInvalidMultipart(t *testing.T) {
	server := &Server{
		frameDir: t.TempDir(),
	}

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
			server := &Server{
				frameDir: t.TempDir(),
			}

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
	server := &Server{
		frameDir: t.TempDir(),
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
	server := &Server{
		frameDir: t.TempDir(),
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

	entries, err := os.ReadDir(server.frameDir)
	if err != nil {
		t.Fatalf("read frame directory: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected no saved files, got %d", len(entries))
	}
}
