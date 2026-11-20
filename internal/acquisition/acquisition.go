package acquisition

import (
	"context"
	"fmt"
	"image"
	"os"
	"os/signal"
	"sync"
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

	var firstFrame sync.Once

	camera := Camera{
		URL: "rtsp://192.168.18.2:554/user=admin&password=&channel=1&stream=0.sdp?",
		OnFrame: func(frame *image.RGBA) {
			firstFrame.Do(func() {
				fmt.Printf(
					"Acquisition received frame: %dx%d\n",
					frame.Bounds().Dx(),
					frame.Bounds().Dy(),
				)
			})
		},
	}

	if err := camera.Connect(ctx); err != nil {
		return fmt.Errorf("connect to camera: %w", err)
	}

	<-ctx.Done()

	fmt.Println("Shutdown signal received")
	fmt.Println("Camera Acquisition stopped")

	return nil
}
