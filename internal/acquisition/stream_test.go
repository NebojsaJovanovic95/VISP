package acquisition

import (
	"testing"

	"github.com/bluenviron/gortsplib/v5/pkg/format"
)

func TestNewH265Stream(t *testing.T) {
	stream, err := newH265Stream(&format.H265{})
	if err != nil {
		t.Fatalf("create H.265 stream: %v", err)
	}

	stream.Close()
}
