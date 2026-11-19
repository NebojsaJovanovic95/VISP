package acquisition

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/bluenviron/gortsplib/v5"
	"github.com/bluenviron/gortsplib/v5/pkg/base"
	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtph265"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h265"
	"github.com/pion/rtp"
)

// Camera represents an RTSP camera connection.
type Camera struct {
	URL string
}

// Connect receives and depacketizes the camera's H.265 stream.
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

	rtpDecoder, err := forma.CreateDecoder()
	if err != nil {
		return fmt.Errorf("create H.265 RTP decoder: %w", err)
	}

	h265Dec := &h265Decoder{}
	if err := h265Dec.initialize(); err != nil {
		return fmt.Errorf("initialize H.265 decoder: %w", err)
	}
	defer h265Dec.close()

	// Provide codec parameters from the RTSP description, if available.
	if forma.VPS != nil {
		h265Dec.decode([][]byte{forma.VPS})
	}
	if forma.SPS != nil {
		h265Dec.decode([][]byte{forma.SPS})
	}
	if forma.PPS != nil {
		h265Dec.decode([][]byte{forma.PPS})
	}

	_, err = client.Setup(desc.BaseURL, media, 0, 0)
	if err != nil {
		return fmt.Errorf("setup RTSP media: %w", err)
	}

	var firstRandomAccess bool
	var firstDecodedFrame sync.Once

	client.OnPacketRTP(media, forma, func(pkt *rtp.Packet) {
		au, err := rtpDecoder.Decode(pkt)
		if err != nil {
			if !errors.Is(err, rtph265.ErrNonStartingPacketAndNoPrevious) &&
				!errors.Is(err, rtph265.ErrMorePacketsNeeded) {
				log.Printf("H.265 RTP decoding error: %v", err)
			}
			return
		}

		// Wait for a random-access frame before starting video decoding.
		if !firstRandomAccess {
			if !h265.IsRandomAccess(au) {
				return
			}
			firstRandomAccess = true
		}

		img, err := h265Dec.decode(au)
		if err != nil {
			log.Printf("H.265 frame decoding error: %v", err)
			return
		}

		if img == nil {
			return
		}

		firstDecodedFrame.Do(func() {
			log.Printf(
				"Successfully decoded first frame: %dx%d",
				img.Bounds().Dx(),
				img.Bounds().Dy(),
			)
		})
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
