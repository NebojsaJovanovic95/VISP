package acquisition

import (
	"context"
	"errors"
	"image"
	"testing"
	"time"
)

type cameraStatusSubmission struct {
	CameraID string
	Status   string
	Err      error
}

type cameraConfigListerStub struct {
	cameras        []CameraConfig
	err            error
	submissions    []frameSubmission
	heartbeats     chan struct{}
	cameraStatuses []cameraStatusSubmission
}

func (s *cameraConfigListerStub) ListCameras(context.Context) ([]CameraConfig, error) {
	return s.cameras, s.err
}

func (s *cameraConfigListerStub) SubmitFrame(
	_ context.Context,
	metadata frameMetadata,
	frame *image.RGBA,
) (string, error) {
	s.submissions = append(s.submissions, frameSubmission{
		Metadata: metadata,
		Frame:    frame,
	})

	return "test-image-id", nil
}

func (s *cameraConfigListerStub) SendHeartbeat(context.Context) error {
	if s.heartbeats != nil {
		s.heartbeats <- struct{}{}
	}
	return nil
}

func (s *cameraConfigListerStub) SendCameraStatus(
	_ context.Context,
	cameraID string,
	status string,
	err error,
) error {
	s.cameraStatuses = append(s.cameraStatuses, cameraStatusSubmission{
		CameraID: cameraID,
		Status:   status,
		Err:      err,
	})

	return nil
}

func TestRun(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client := cameraConfigListerStub{
		cameras: []CameraConfig{
			{
				CameraID:                   "inactive-camera",
				CameraName:                 "Inactive Camera",
				RTSPURL:                    "rtsp://localhost:554/inactive",
				Status:                     "inactive",
				AcquisitionIntervalSeconds: 5,
			},
			{
				CameraID:                   "active-camera",
				CameraName:                 "Active Camera",
				RTSPURL:                    "rtsp://localhost:554/active",
				Status:                     "active",
				AcquisitionIntervalSeconds: 5,
			},
		},
	}

	var connectedCameras []Camera
	frame := image.NewRGBA(image.Rect(0, 0, 2048, 1152))
	capturedAt := time.Now()

	err := run(ctx, &client, func(_ context.Context, camera Camera) error {
		connectedCameras = append(connectedCameras, camera)

		camera.OnFrame(frame, capturedAt)

		cancel()
		return nil
	})
	if err != nil {
		t.Fatalf("run acquisition: %v", err)
	}

	if len(connectedCameras) != 1 {
		t.Fatalf("expected 1 connected camera, got %d", len(connectedCameras))
	}

	submission := client.submissions[0]

	if submission.Metadata.CameraID != "active-camera" {
		t.Errorf(
			"expected camera ID active-camera, got %s",
			submission.Metadata.CameraID,
		)
	}

	if !submission.Metadata.CapturedAt.Equal(capturedAt) {
		t.Errorf("expected capture time %v, got %v",
			capturedAt,
			submission.Metadata.CapturedAt,
		)
	}

	if submission.Frame != frame {
		t.Error("submitted frame does not match acquired frame")
	}
	if connectedCameras[0].URL != "rtsp://localhost:554/active" {
		t.Errorf("expected active camera URL, got %s", connectedCameras[0].URL)
	}

	if connectedCameras[0].AcquisitionInterval != 5*time.Second {
		t.Errorf(
			"expected acquisition interval %s, got %s",
			5*time.Second,
			connectedCameras[0].AcquisitionInterval,
		)
	}
}

func TestRunListCamerasError(t *testing.T) {
	expectedErr := errors.New("server unavailable")

	client := cameraConfigListerStub{
		err: expectedErr,
	}

	err := run(context.Background(), &client, func(context.Context, Camera) error {
		t.Fatal("camera should not be connected")
		return nil
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRunCameraConnectionError(t *testing.T) {
	expectedErr := errors.New("connection failed")

	client := cameraConfigListerStub{
		cameras: []CameraConfig{
			{
				RTSPURL: "rtsp://localhost:554/active",
				Status:  "active",
			},
		},
	}

	err := run(context.Background(), &client, func(context.Context, Camera) error {
		return expectedErr
	})

	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, err)
	}
}

func TestRunHeartbeat(t *testing.T) {
	heartbeats := make(chan struct{}, 2)
	client := &cameraConfigListerStub{
		heartbeats: heartbeats,
	}

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		runHeartbeat(ctx, client, 10*time.Millisecond)
	}()

	// Verify that two heartbeats are sent.
	for i := 0; i < 2; i++ {
		select {
		case <-heartbeats:
		case <-time.After(time.Second):
			t.Fatal("timed out waiting for heartbeat")
		}
	}

	cancel()

	// Verify that the heartbeat goroutine exits.
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("heartbeat goroutine did not stop")
	}
}

func TestRunCameraStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	expectedErr := errors.New("connection refused")

	client := cameraConfigListerStub{
		cameras: []CameraConfig{
			{
				CameraID:   "test-camera",
				CameraName: "Test Camera",
				RTSPURL:    "rtsp://localhost:554/test",
				Status:     "active",
			},
		},
	}

	err := run(ctx, &client, func(_ context.Context, camera Camera) error {
		camera.OnStatus("disconnected", expectedErr)
		cancel()
		return nil
	})
	if err != nil {
		t.Fatalf("run acquisition: %v", err)
	}

	if len(client.cameraStatuses) != 1 {
		t.Fatalf(
			"expected 1 camera status report, got %d",
			len(client.cameraStatuses),
		)
	}

	report := client.cameraStatuses[0]

	if report.CameraID != "test-camera" {
		t.Errorf("expected camera ID test-camera, got %s", report.CameraID)
	}

	if report.Status != "disconnected" {
		t.Errorf("expected status disconnected, got %s", report.Status)
	}

	if !errors.Is(report.Err, expectedErr) {
		t.Errorf("expected error %v, got %v", expectedErr, report.Err)
	}
}
