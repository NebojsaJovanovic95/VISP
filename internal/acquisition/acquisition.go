package acquisition

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NebojsaJovanovic95/VISP/internal/config"
)

type cameraConfigLister interface {
	ListCameras(context.Context) ([]CameraConfig, error)
	SubmitFrame(context.Context, frameMetadata, *image.RGBA) (string, error)
	SendHeartbeat(context.Context) error
}

// Run starts the Camera Acquisition component.
func Run() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	cfg, err := config.LoadAcquisition()
	if err != nil {
		return fmt.Errorf("load acquisition configuration: %w", err)
	}

	client := NewClient(cfg.ServerURL)

	return run(ctx, client, func(ctx context.Context, camera Camera) error {
		return camera.Connect(ctx)
	})
}

func run(
	ctx context.Context,
	client cameraConfigLister,
	connectCamera func(context.Context, Camera) error,
) error {
	fmt.Println("Camera Acquisition started")
	fmt.Println("Fetching camera configurations from VISP Server")

	cameras, err := client.ListCameras(ctx)
	if err != nil {
		return fmt.Errorf("retrieve camera configurations: %w", err)
	}

	fmt.Printf("Retrieved %d camera(s)\n", len(cameras))

	for _, cameraConfig := range cameras {
		fmt.Printf(
			"Camera: %s | ID: %s | Status: %s | Acquisition interval: %s\n",
			cameraConfig.CameraName,
			cameraConfig.CameraID,
			cameraConfig.Status,
			time.Duration(cameraConfig.AcquisitionIntervalSeconds*float64(time.Second)),
		)
		fmt.Printf("URI: %s\n", cameraConfig.RTSPURL)
	}

	go runHeartbeat(ctx, client, 30*time.Second)

	frames := make(chan frameSubmission, 10)
	workerDone := make(chan struct{})

	go func() {
		defer close(workerDone)

		for submission := range frames {
			imageId, err := client.SubmitFrame(ctx, submission.Metadata, submission.Frame)
			if err != nil {
				fmt.Printf("Submit frame: %v\n", err)
				continue
			}

			fmt.Printf("Frame submitted: %s\n", imageId)
		}
	}()

	defer func() {
		close(frames)
		<-workerDone
	}()

	for _, cameraConfig := range cameras {
		if cameraConfig.Status != "active" {
			continue
		}

		camera := Camera{
			URL:                 cameraConfig.RTSPURL,
			AcquisitionInterval: time.Duration(cameraConfig.AcquisitionIntervalSeconds) * time.Second,
			OnFrame: func(frame *image.RGBA, capturedAt time.Time) {
				metadata := frameMetadata{
					CameraID:   cameraConfig.CameraID,
					CapturedAt: capturedAt,
				}

				select {
				case frames <- frameSubmission{
					Metadata: metadata,
					Frame:    frame,
				}:
				case <-ctx.Done():
					return
				}
			},
			OnStatus: func(status string, err error) {
				fmt.Printf("Camera %s status: %s", cameraConfig.CameraName, status)

				if err != nil {
					fmt.Printf(" (%v)", err)
				}

				fmt.Println()
			},
		}

		if err := connectCamera(ctx, camera); err != nil {
			return fmt.Errorf("connect to camera: %w", err)
		}
	}

	<-ctx.Done()

	fmt.Println("Shutdown signal received")
	fmt.Println("Camera Acquisition stopped")

	return nil
}

func runHeartbeat(
	ctx context.Context,
	client cameraConfigLister,
	interval time.Duration,
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := client.SendHeartbeat(ctx); err != nil {
				fmt.Printf("Send heartbeat: %v\n", err)
			}
		}
	}
}
