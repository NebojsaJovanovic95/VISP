package acquisition

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
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
