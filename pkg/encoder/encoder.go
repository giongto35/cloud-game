package encoder

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/giongto35/cloud-game/v3/pkg/encoder/yuv"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
)

type (
	InFrame  yuv.RawFrame
	OutFrame []byte
	Encoder  interface {
		LoadBuf(input []byte)
		Encode() []byte
		IntraRefresh()
		SetFlip(bool)
		Shutdown() error
	}
)

type Video struct {
	codec   Encoder
	log     *logger.Logger
	stopped atomic.Bool
	y       yuv.Conv
	pf      yuv.PixFmt
	rot     uint
	mu      sync.Mutex
}

type VideoCodec string

const (
	H264 VideoCodec = "h264"
	VP8  VideoCodec = "vp8"
)

// NewVideoEncoder returns new video encoder.
// By default, it waits for RGBA images on the input channel,
// converts them into YUV I420 format,
// encodes with provided video encoder, and
// puts the result into the output channel.
func NewVideoEncoder(codec Encoder, w, h int, scale float64, log *logger.Logger) *Video {
	return &Video{codec: codec, y: yuv.NewYuvConv(w, h, scale), log: log}
}

func (v *Video) Encode(frame InFrame) OutFrame {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.stopped.Load() {
		return nil
	}

	yCbCr := v.y.Process(yuv.RawFrame(frame), v.rot, v.pf)
	v.codec.LoadBuf(yCbCr)
	v.y.Put(&yCbCr)

	if bytes := v.codec.Encode(); len(bytes) > 0 {
		return bytes
	}
	return nil
}

func (v *Video) Info() string { return fmt.Sprintf("libyuv: %v", v.y.Version()) }

func (v *Video) SetPixFormat(f uint32) {
	switch f {
	case 1:
		v.pf = yuv.PixFmt(yuv.FourccArgb)
	case 2:
		v.pf = yuv.PixFmt(yuv.FourccRgbp)
	default:
		v.pf = yuv.PixFmt(yuv.FourccAbgr)
	}
}

// SetRot sets the rotation angle of the frames.
func (v *Video) SetRot(r uint) {
	switch r {
	// de-rotate
	case 90:
		v.rot = 270
	case 270:
		v.rot = 90
	default:
		v.rot = r
	}
}

// SetFlip tells the encoder to flip the frames vertically.
func (v *Video) SetFlip(b bool) { v.codec.SetFlip(b) }

func (v *Video) Stop() {
	v.stopped.Store(true)
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rot = 0

	if err := v.codec.Shutdown(); err != nil {
		v.log.Error().Err(err).Msg("failed to close the encoder")
	}
}
