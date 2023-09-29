package encoder

import (
	"image"
	"sync"
	"sync/atomic"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/encoder/yuv"
)

type (
	InFrame  *image.RGBA
	OutFrame []byte
	Encoder  interface {
		LoadBuf(input []byte)
		Encode() []byte
		IntraRefresh()
		SetFlip(bool)
		Shutdown() error
	}
)

type VideoEncoder struct {
	encoder Encoder
	log     *logger.Logger
	stopped atomic.Bool
	y       yuv.ImgProcessor
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
func NewVideoEncoder(enc Encoder, w, h int, concurrency int, log *logger.Logger) *VideoEncoder {
	y := yuv.NewYuvImgProcessor(w, h, &yuv.Options{Threads: concurrency})
	if concurrency > 0 {
		log.Info().Msgf("Use concurrent image processor: %v", concurrency)
	}
	return &VideoEncoder{encoder: enc, y: y, log: log}
}

func (vp *VideoEncoder) Encode(img InFrame) OutFrame {
	vp.mu.Lock()
	defer vp.mu.Unlock()
	if vp.stopped.Load() {
		return nil
	}

	yCbCr := vp.y.Process(img)
	vp.encoder.LoadBuf(yCbCr)
	vp.y.Put(&yCbCr)

	if frame := vp.encoder.Encode(); len(frame) > 0 {
		return frame
	}
	return nil
}

func (vp *VideoEncoder) SetFlip(b bool) { vp.encoder.SetFlip(b) }

func (vp *VideoEncoder) Stop() {
	vp.stopped.Store(true)
	vp.mu.Lock()
	defer vp.mu.Unlock()

	if err := vp.encoder.Shutdown(); err != nil {
		vp.log.Error().Err(err).Msg("failed to close the encoder")
	}
}
