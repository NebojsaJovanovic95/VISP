package acquisition

import (
	"context"
	"fmt"
	"image"
	"log"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/pion/rtp"
)

// Camera represents an RTSP camera connection.
type Camera struct {
	URL                 string
	AcquisitionInterval time.Duration
	OnFrame             func(*image.RGBA, time.Time)
}

// Connect establishes an RTSP connection and receives the camera stream.
func (c *Camera) Connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

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

	var closeOnce sync.Once

	closeClient := func() {
		closeOnce.Do(client.Close)
	}

	stopCancellation := context.AfterFunc(ctx, closeClient)
	defer stopCancellation()
	defer closeClient()

	desc, _, err := client.Describe(u)
	if err != nil {
		return fmt.Errorf("describe RTSP stream: %w", err)
	}

	var forma *format.H265
	media := desc.FindFormat(&forma)
	if media == nil {
		return fmt.Errorf("camera stream does not contain H.265 video")
	}

	stream, err := newH265Stream(forma)
	if err != nil {
		return fmt.Errorf("create H.265 stream: %w", err)
	}
	defer stream.Close()

	_, err = client.Setup(desc.BaseURL, media, 0, 0)
	if err != nil {
		return fmt.Errorf("setup RTSP media: %w", err)
	}

	var lastAcquired time.Time

	client.OnPacketRTP(media, forma, func(pkt *rtp.Packet) {
		img, err := stream.ProcessPacket(pkt)
		if err != nil {
			log.Printf("Process video packet: %v", err)
			return
		}

		if img == nil {
			return
		}

		now := time.Now()

		if c.AcquisitionInterval > 0 && !lastAcquired.IsZero() && now.Sub(lastAcquired) < c.AcquisitionInterval {
			return
		}

		lastAcquired = now

		if c.OnFrame != nil {
			c.OnFrame(img, time.Now())
		}
	})

	_, err = client.Play(nil)
	if err != nil {
		return fmt.Errorf("start RTSP playback: %w", err)
	}

	log.Println("Receiving H.265 stream")

	err = client.Wait()
	if ctx.Err() != nil {
		return nil
	}
	if err != nil {
		return fmt.Errorf("RTSP client stopped: %w", err)
	}

	return nil
}
