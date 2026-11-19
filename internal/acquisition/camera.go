package acquisition

import (
	"context"
	"fmt"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
)

// Camera represents an RTSP camera connection.
type Camera struct {
	URL string
}

// Connect establishes an RTSP connection to the camera.
func (c *Camera) Connect(ctx context.Context) error {
	u, err := base.ParseURL(c.URL)
	if err != nil {
		return fmt.Errorf("parse RTSP URL: %w", err)
	}

	client := &gortsplib.Client{
		Scheme: u.Scheme,
		Host:   u.Host,
	}

	if err := client.Start(); err != nil {
		return fmt.Errorf("start RTSP client: %w", err)
	}
	defer client.Close()

	desc, _, err := client.Describe(u)
	if err != nil {
		return fmt.Errorf("describe RTSP stream: %w", err)
	}

	fmt.Printf("Connected to camera. Media tracks: %d\n", len(desc.Medias))

	return nil
}
