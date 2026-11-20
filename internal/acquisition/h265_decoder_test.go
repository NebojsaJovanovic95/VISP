package acquisition

import (
	"strings"
	"testing"
)

func TestH265DecoderDecodeEmptyAccessUnit(t *testing.T) {
	decoder := &h265Decoder{}

	_, err := decoder.decode(nil)
	if err == nil {
		t.Fatal("expected an error for an empty access unit")
	}

	if !strings.Contains(err.Error(), "empty H.265 access unit") {
		t.Errorf("expected empty access unit error, got %v", err)
	}
}

func TestH265DecoderInitializeAndClose(t *testing.T) {
	decoder := &h265Decoder{}

	if err := decoder.initialize(); err != nil {
		t.Fatalf("initialize decoder: %v", err)
	}

	decoder.close()
}
