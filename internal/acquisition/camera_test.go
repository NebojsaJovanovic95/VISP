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

func TestCameraReportsConnectionFailure(t *testing.T) {
	var reportedStatus string
	var reportedErr error

	camera := Camera{
		URL: "invalid-url",
		OnStatus: func(status string, err error) {
			reportedStatus = status
			reportedErr = err
		},
	}

	err := camera.Connect(context.Background())

	if err == nil {
		t.Fatal("expected connection error")
	}

	if reportedStatus != "disconnected" {
		t.Errorf("expected status disconnected, got %q", reportedStatus)
	}

	if reportedErr == nil {
		t.Error("expected reported error")
	}
}
