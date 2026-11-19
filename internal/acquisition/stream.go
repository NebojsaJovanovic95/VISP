package acquisition

import (
	"errors"
	"fmt"
	"image"

	"github.com/bluenviron/gortsplib/v5/pkg/format"
	"github.com/bluenviron/gortsplib/v5/pkg/format/rtph265"
	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h265"
	"github.com/pion/rtp"
)

// h265Stream receives RTP packets and decodes them into images.
type h265Stream struct {
	rtpDecoder        *rtph265.Decoder
	h265Decoder       *h265Decoder
	firstRandomAccess bool
}

// newH265Stream creates and initializes an H.265 stream processor.
func newH265Stream(forma *format.H265) (*h265Stream, error) {
	rtpDecoder, err := forma.CreateDecoder()
	if err != nil {
		return nil, fmt.Errorf("create RTP decoder: %w", err)
	}

	videoDecoder := &h265Decoder{}
	if err := videoDecoder.initialize(); err != nil {
		return nil, fmt.Errorf("initialize H.265 decoder: %w", err)
	}

	// Provide codec parameters from the RTSP description.
	if forma.VPS != nil {
		videoDecoder.decode([][]byte{forma.VPS})
	}
	if forma.SPS != nil {
		videoDecoder.decode([][]byte{forma.SPS})
	}
	if forma.PPS != nil {
		videoDecoder.decode([][]byte{forma.PPS})
	}

	return &h265Stream{
		rtpDecoder:  rtpDecoder,
		h265Decoder: videoDecoder,
	}, nil
}

// Close releases the video decoder resources.
func (s *h265Stream) Close() {
	s.h265Decoder.close()
}

// ProcessPacket converts an RTP packet into a decoded image.
func (s *h265Stream) ProcessPacket(pkt *rtp.Packet) (*image.RGBA, error) {
	au, err := s.rtpDecoder.Decode(pkt)
	if err != nil {
		if errors.Is(err, rtph265.ErrNonStartingPacketAndNoPrevious) ||
			errors.Is(err, rtph265.ErrMorePacketsNeeded) {
			return nil, nil
		}

		return nil, fmt.Errorf("decode RTP packet: %w", err)
	}

	// Wait for a random-access unit before starting video decoding.
	if !s.firstRandomAccess {
		if !h265.IsRandomAccess(au) {
			return nil, nil
		}

		s.firstRandomAccess = true
	}

	img, err := s.h265Decoder.decode(au)
	if err != nil {
		return nil, fmt.Errorf("decode H.265 access unit: %w", err)
	}

	return img, nil
}
