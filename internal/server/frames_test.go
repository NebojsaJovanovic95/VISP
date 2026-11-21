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
