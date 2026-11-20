package acquisition

import (
	"context"
	"fmt"
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

	for _, camera := range cameras {
		fmt.Printf(
			"Camera: %s | ID: %s | Status: %s | Acquisition interval: %s\n",
			camera.CameraName,
			camera.CameraID,
			camera.Status,
			time.Duration(camera.AcquisitionIntervalSeconds*float64(time.Second)),
		)
	}

	<-ctx.Done()

	fmt.Println("Shutdown signal received")
	fmt.Println("Camera Acquisition stopped")

	return nil
}
