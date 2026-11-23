package acquisition

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"mime"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestListCameras(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected method %s, got %s", http.MethodGet, r.Method)
			}

			if r.URL.Path != "/api/cameras" {
				t.Errorf("expected path %q, got %q", "/api/cameras", r.URL.Path)
			}

			w.Header().Set("Content-Type", "application/json")
			fmt.Fprintln(w, `[
				{
					"camera_id": "test-camera-id",
					"camera_name": "Test Camera",
					"rtsp_url": "rtsp://localhost:554/stream",
					"location": "Test Location",
					"status": "active",
					"acquisition_interval_seconds": 5
				}
			]`)
		},
	))
	defer server.Close()

	client := NewClient(server.URL)

	cameras, err := client.ListCameras(context.Background())
	if err != nil {
		t.Fatalf("list cameras: %v", err)
	}

	if len(cameras) != 1 {
		t.Fatalf("expected 1 camera, got %d", len(cameras))
	}

	camera := cameras[0]

	if camera.CameraID != "test-camera-id" {
		t.Errorf("expected camera ID %q, got %q", "test-camera-id", camera.CameraID)
	}

	if camera.CameraName != "Test Camera" {
		t.Errorf("expected camera name %q, got %q", "Test Camera", camera.CameraName)
	}

	if camera.AcquisitionIntervalSeconds != 5 {
		t.Errorf("expected acquisition interval 5, got %f", camera.AcquisitionIntervalSeconds)
	}

	if camera.Location == nil || *camera.Location != "Test Location" {
		t.Errorf("expected location %q, got %v", "Test Location", camera.Location)
	}
}

func TestListCamerasServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		},
	))
	defer server.Close()

	client := NewClient(server.URL)

	_, err := client.ListCameras(context.Background())
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestListCamerasInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintln(w, "not valid JSON")
		},
	))
	defer server.Close()

	client := NewClient(server.URL)

	_, err := client.ListCameras(context.Background())
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestSubmitFrame(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected method %s, got %s", http.MethodPost, r.Method)
			}

			if r.URL.Path != "/api/frames" {
				t.Errorf("expected path %q, got %q", "/api/frames", r.URL.Path)
			}

			mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
			if err != nil {
				t.Fatalf("parse content type: %v", err)
			}

			if mediaType != "multipart/form-data" {
				t.Errorf("expected multipart/form-data, got %q", mediaType)
			}

			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}

			var metadata frameMetadata

			if err := json.NewDecoder(strings.NewReader(r.FormValue("metadata"))).Decode(&metadata); err != nil {
				t.Fatalf("decode metadata: %v", err)
			}

			if metadata.CameraID != "test-camera-id" {
				t.Errorf("expected camera ID %q, got %q", "test-camera-id", metadata.CameraID)
			}

			imageFile, _, err := r.FormFile("image")
			if err != nil {
				t.Fatalf("get image field: %v", err)
			}
			defer imageFile.Close()

			decodedImage, err := png.Decode(imageFile)
			if err != nil {
				t.Fatalf("decode submitted PNG: %v", err)
			}

			if decodedImage.Bounds().Dx() != 640 || decodedImage.Bounds().Dy() != 480 {
				t.Errorf("unexpected image dimensions: %v", decodedImage.Bounds())
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			io.WriteString(w, `{"image_id":"test-image-id"}`)
		},
	))
	defer server.Close()

	client := NewClient(server.URL)

	frame := image.NewRGBA(image.Rect(0, 0, 640, 480))
	capturedAt := time.Now()

	metadata := frameMetadata{
		CameraID:   "test-camera-id",
		CapturedAt: capturedAt,
	}

	imageID, err := client.SubmitFrame(context.Background(), metadata, frame)
	if err != nil {
		t.Fatalf("submit frame: %v", err)
	}

	if imageID != "test-image-id" {
		t.Errorf("expected image ID %q, got %q", "test-image-id", imageID)
	}
}

func TestSubmitFrameServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		},
	))
	defer server.Close()

	client := NewClient(server.URL)

	frame := image.NewRGBA(image.Rect(0, 0, 640, 480))
	metadata := frameMetadata{
		CameraID:   "test-camera-id",
		CapturedAt: time.Now(),
	}

	_, err := client.SubmitFrame(context.Background(), metadata, frame)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestSubmitFrameInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintln(w, "not valid JSON")
		},
	))
	defer server.Close()

	client := NewClient(server.URL)

	frame := image.NewRGBA(image.Rect(0, 0, 640, 480))
	metadata := frameMetadata{
		CameraID:   "test-camera-id",
		CapturedAt: time.Now(),
	}

	_, err := client.SubmitFrame(context.Background(), metadata, frame)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestSendHeartbeat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected method %s, got %s", http.MethodPost, r.Method)
		}

		if r.URL.Path != "/api/heartbeat" {
			t.Errorf("expected path /api/heartbeat, got %s", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	if err := client.SendHeartbeat(context.Background()); err != nil {
		t.Fatalf("SendHeartbeat returned unexpected error: %v", err)
	}
}

func TestSendHeartbeatServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	err := client.SendHeartbeat(context.Background())
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestSendCameraStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected method POST, got %s", r.Method)
		}

		expectedPath := "/api/cameras/camera-123/status"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}

		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got %s", contentType)
		}

		var request cameraStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request body: %v", err)
			return
		}

		if request.Status != "connected" {
			t.Errorf("expected status connected, got %s", request.Status)
		}

		if request.Error != "" {
			t.Errorf("expected no error, got %s", request.Error)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	if err := client.SendCameraStatus(context.Background(), "camera-123", "connected", nil); err != nil {
		t.Fatalf("SendCameraStatus returned error: %v", err)
	}
}

func TestSendCameraStatusWithError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request cameraStatusRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request body: %v", err)
			return
		}

		if request.Status != "disconnected" {
			t.Errorf("expected status disconnected, got %s", request.Status)
		}

		if request.Error != "connection refused" {
			t.Errorf("expected error connection refused, got %s", request.Error)
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	statusErr := errors.New("connection refused")

	if err := client.SendCameraStatus(context.Background(), "camera-123", "disconnected", statusErr); err != nil {
		t.Fatalf("SendCameraStatus returned error: %v", err)
	}
}

func TestSendCameraStatusServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	err := client.SendCameraStatus(context.Background(), "camera-123", "connected", nil)
	if err == nil {
		t.Fatal("expected error when server returns 500")
	}
}
