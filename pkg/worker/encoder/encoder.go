package encoder

import (
	"image"

	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/worker/encoder/yuv"
)

type (
	InFrame  *image.RGBA
	OutFrame []byte
	Encoder  interface {
		LoadBuf(input []byte)
		Encode() []byte
		IntraRefresh()
		Shutdown() error
	}
)

type VideoEncoder struct {
	encoder Encoder

	y yuv.ImgProcessor

	// frame size
	w, h int
	log  *logger.Logger
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
	return &VideoEncoder{encoder: enc, y: y, w: w, h: h, log: log}
}

func (vp VideoEncoder) Encode(img InFrame) OutFrame {
	yCbCr := vp.y.Process(img)
	vp.encoder.LoadBuf(yCbCr)
	vp.y.Put(&yCbCr)

	if frame := vp.encoder.Encode(); len(frame) > 0 {
		return frame
	}
	return nil
}

// Start begins video encoding pipe.
// Should be wrapped into a goroutine.
func (vp VideoEncoder) Start() {}

func (vp VideoEncoder) Stop() {
	if err := vp.encoder.Shutdown(); err != nil {
		vp.log.Error().Err(err).Msg("failed to close the encoder")
	}
}
