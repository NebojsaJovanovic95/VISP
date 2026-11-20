package acquisition

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCameraConnectCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	camera := Camera{
		URL: "rtsp://localhost:554/stream",
	}

	err := camera.Connect(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestCameraConnectInvalidURL(t *testing.T) {
	camera := Camera{
		URL: "invalid-url",
	}

	err := camera.Connect(context.Background())
	if err == nil {
		t.Fatal("expected an error for invalid RTSP URL")
	}

	if !strings.Contains(err.Error(), "parse RTSP URL") {
		t.Errorf("expected URL parsing error, got %v", err)
	}
}
