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

	for _, cameraConfig := range cameras {
		if cameraConfig.Status != "active" {
			continue
		}

		camera := Camera{
			URL:                 cameraConfig.RTSPURL,
			AcquisitionInterval: time.Duration(cameraConfig.AcquisitionIntervalSeconds) * time.Second,
			OnFrame: func(frame *image.RGBA) {
				fmt.Printf(
					"[%s] Acquisition received frame: %dx%d\n",
					time.Now().Format("15:04:05"),
					frame.Bounds().Dx(),
					frame.Bounds().Dy(),
				)
			},
		}

		if err := camera.Connect(ctx); err != nil {
			return fmt.Errorf("connect to camera: %w", err)
		}
	}

	<-ctx.Done()

	fmt.Println("Shutdown signal received")
	fmt.Println("Camera Acquisition stopped")

	return nil
}
