package acquisition

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// Run starts the Camera Acquisition component.
func Run() error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	fmt.Println("Camera Acquisition started")

	camera := Camera{
		URL: "rtsp://192.168.18.2:554/user=admin&password=&channel=1&stream=0.sdp?",
	}

	if err := camera.Connect(ctx); err != nil {
		return fmt.Errorf("connect to camera: %w", err)
	}

	<-ctx.Done()

	fmt.Println("Shutdown signal received")
	fmt.Println("Camera Acquisition stopped")

	return nil
}
