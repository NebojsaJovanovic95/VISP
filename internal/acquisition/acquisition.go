package acquisition

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/signal"
	"syscall"
	"time"
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
		URL:                 "rtsp://192.168.18.2:554/user=admin&password=&channel=1&stream=0.sdp?",
		AcquisitionInterval: 5 * time.Second,
		OnFrame: func(frame *image.RGBA) {
			fmt.Printf(
				"[%s] Acquisition received frame: %dx%d\n",
				time.Now().Format("15:04:05"),
				frame.Bounds().Dx(),
				frame.Bounds().Dy(),
			)
		}}

	if err := camera.Connect(ctx); err != nil {
		return fmt.Errorf("connect to camera: %w", err)
	}

	<-ctx.Done()

	fmt.Println("Shutdown signal received")
	fmt.Println("Camera Acquisition stopped")

	return nil
}
