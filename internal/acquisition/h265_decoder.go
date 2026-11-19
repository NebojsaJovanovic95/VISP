package acquisition

// #cgo pkg-config: libavcodec libavutil libswscale
// #include <libavcodec/avcodec.h>
// #include <libavutil/imgutils.h>
// #include <libswscale/swscale.h>
import "C"

import (
	"fmt"
	"image"
	"runtime"
	"unsafe"

	"github.com/bluenviron/mediacommon/v2/pkg/codecs/h264"
)

// h265Decoder wraps FFmpeg's H.265 decoder.
type h265Decoder struct {
	codecCtx     *C.AVCodecContext
	yuvFrame     *C.AVFrame
	rgbaFrame    *C.AVFrame
	rgbaFramePtr []uint8
	swsCtx       *C.struct_SwsContext
}

// initialize initializes the H.265 decoder.
func (d *h265Decoder) initialize() error {
	codec := C.avcodec_find_decoder(C.AV_CODEC_ID_H265)
	if codec == nil {
		return fmt.Errorf("find H.265 decoder")
	}

	d.codecCtx = C.avcodec_alloc_context3(codec)
	if d.codecCtx == nil {
		return fmt.Errorf("allocate codec context")
	}

	if res := C.avcodec_open2(d.codecCtx, codec, nil); res < 0 {
		return fmt.Errorf("open H.265 decoder: %d", int(res))
	}

	d.yuvFrame = C.av_frame_alloc()
	if d.yuvFrame == nil {
		return fmt.Errorf("allocate YUV frame")
	}

	return nil
}

// close releases decoder resources.
func (d *h265Decoder) close() {
	if d.swsCtx != nil {
		C.sws_freeContext(d.swsCtx)
		d.swsCtx = nil
	}

	if d.rgbaFrame != nil {
		C.av_frame_free(&d.rgbaFrame)
	}

	if d.yuvFrame != nil {
		C.av_frame_free(&d.yuvFrame)
	}

	if d.codecCtx != nil {
		C.avcodec_free_context(&d.codecCtx)
	}
}

// reinitializeOutput prepares the RGBA frame and color converter.
func (d *h265Decoder) reinitializeOutput() error {
	if d.swsCtx != nil {
		C.sws_freeContext(d.swsCtx)
		d.swsCtx = nil
	}

	if d.rgbaFrame != nil {
		C.av_frame_free(&d.rgbaFrame)
	}

	d.rgbaFrame = C.av_frame_alloc()
	if d.rgbaFrame == nil {
		return fmt.Errorf("allocate RGBA frame")
	}

	d.rgbaFrame.format = C.AV_PIX_FMT_RGBA
	d.rgbaFrame.width = d.yuvFrame.width
	d.rgbaFrame.height = d.yuvFrame.height
	d.rgbaFrame.color_range = C.AVCOL_RANGE_JPEG

	if res := C.av_frame_get_buffer(d.rgbaFrame, 1); res < 0 {
		return fmt.Errorf("allocate RGBA frame buffer: %d", int(res))
	}

	d.swsCtx = C.sws_getContext(
		d.yuvFrame.width,
		d.yuvFrame.height,
		C.enum_AVPixelFormat(d.yuvFrame.format),
		d.rgbaFrame.width,
		d.rgbaFrame.height,
		C.enum_AVPixelFormat(d.rgbaFrame.format),
		C.SWS_BILINEAR,
		nil,
		nil,
		nil,
	)
	if d.swsCtx == nil {
		return fmt.Errorf("create color converter")
	}

	size := C.av_image_get_buffer_size(
		C.enum_AVPixelFormat(d.rgbaFrame.format),
		d.rgbaFrame.width,
		d.rgbaFrame.height,
		1,
	)
	if size < 0 {
		return fmt.Errorf("calculate RGBA buffer size: %d", int(size))
	}

	d.rgbaFramePtr = (*[1 << 30]uint8)(
		unsafe.Pointer(d.rgbaFrame.data[0]),
	)[:int(size):int(size)]

	return nil
}

// decode converts an H.265 access unit into an RGBA image.
func (d *h265Decoder) decode(au [][]byte) (*image.RGBA, error) {
	if len(au) == 0 {
		return nil, fmt.Errorf("empty H.265 access unit")
	}

	// Convert NAL units into Annex-B format.
	annexb, err := h264.AnnexB(au).Marshal()
	if err != nil {
		return nil, fmt.Errorf("marshal H.265 access unit: %w", err)
	}

	if len(annexb) == 0 {
		return nil, fmt.Errorf("empty Annex-B data")
	}

	// Send the encoded access unit to FFmpeg.
	var pkt C.AVPacket

	ptr := &annexb[0]
	var pinner runtime.Pinner
	pinner.Pin(ptr)

	pkt.data = (*C.uint8_t)(unsafe.Pointer(ptr))
	pkt.size = C.int(len(annexb))

	res := C.avcodec_send_packet(d.codecCtx, &pkt)
	pinner.Unpin()

	if res < 0 {
		return nil, fmt.Errorf("send packet to H.265 decoder: %d", int(res))
	}

	// Retrieve a decoded frame.
	res = C.avcodec_receive_frame(d.codecCtx, d.yuvFrame)
	if res < 0 {
		// FFmpeg may need more input before producing a frame.
		return nil, nil
	}

	// Recreate output resources if the frame dimensions change.
	if d.rgbaFrame == nil ||
		d.rgbaFrame.width != d.yuvFrame.width ||
		d.rgbaFrame.height != d.yuvFrame.height {
		if err := d.reinitializeOutput(); err != nil {
			return nil, err
		}
	}

	// Convert YUV into RGBA.
	res = C.sws_scale(
		d.swsCtx,
		(**C.uint8_t)(unsafe.Pointer(&d.yuvFrame.data[0])),
		(*C.int)(unsafe.Pointer(&d.yuvFrame.linesize[0])),
		0,
		d.yuvFrame.height,
		(**C.uint8_t)(unsafe.Pointer(&d.rgbaFrame.data[0])),
		(*C.int)(unsafe.Pointer(&d.rgbaFrame.linesize[0])),
	)
	if res < 0 {
		return nil, fmt.Errorf("convert frame to RGBA: %d", int(res))
	}

	width := int(d.rgbaFrame.width)
	height := int(d.rgbaFrame.height)

	// Copy pixels so subsequent decoding doesn't overwrite this image.
	pixels := append([]uint8(nil), d.rgbaFramePtr...)

	return &image.RGBA{
		Pix:    pixels,
		Stride: width * 4,
		Rect:   image.Rect(0, 0, width, height),
	}, nil
}
