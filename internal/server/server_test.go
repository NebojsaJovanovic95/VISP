package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/NebojsaJovanovic95/VISP/internal/config"
	"github.com/NebojsaJovanovic95/VISP/internal/database"
)

func TestHealthHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	healthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "OK" {
		t.Errorf("expected body %q, got %q", "OK", rec.Body.String())
	}
}

func TestHealthRoute(t *testing.T) {
	server := New(nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, rec.Code)
	}

	if rec.Body.String() != "OK" {
		t.Errorf("expected body %q, got %q", "OK", rec.Body.String())
	}
}

func TestListCamerasRoute(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	server := New(db)

	req := httptest.NewRequest(http.MethodGet, "/api/cameras", nil)
	rec := httptest.NewRecorder()

	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s",
			http.StatusOK, rec.Code, rec.Body.String())
	}

	if contentType := rec.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", contentType)
	}

	var cameras []cameraResponse

	if err := json.NewDecoder(rec.Body).Decode(&cameras); err != nil {
		t.Fatalf("decode camera response: %v", err)
	}

	if cameras == nil {
		t.Fatal("expected JSON array, got nil")
	}

	for _, camera := range cameras {
		if camera.CameraID == "" {
			t.Error("expected camera ID")
		}

		if camera.CameraName == "" {
			t.Error("expected camera name")
		}

		if camera.RTSPURL == "" {
			t.Error("expected RTSP URL")
		}

		if camera.AcquisitionIntervalSeconds <= 0 {
			t.Errorf("expected positive acquisition interval, got %f",
				camera.AcquisitionIntervalSeconds)
		}
	}
}

func TestListCamerasDatabaseError(t *testing.T) {
	if os.Getenv("POSTGRES_PASSWORD") == "" {
		t.Skip("PostgreSQL environment variables are not configured")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("load configuration: %v", err)
	}

	db, err := database.New(ctx, cfg.Database)
	if err != nil {
		t.Fatalf("connect to PostgreSQL: %v", err)
	}

	// Close the pool to simulate a database failure. Deliberetly not deffered!
	db.Close()

	server := New(db)

	req := httptest.NewRequest(http.MethodGet, "/api/cameras", nil)
	rec := httptest.NewRecorder()

	server.Router().ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d",
			http.StatusInternalServerError, rec.Code)
	}
}
