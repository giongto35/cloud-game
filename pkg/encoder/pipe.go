package encoder

import (
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/encoder/yuv"
)

type VideoPipe struct {
	Input  chan InFrame
	Output chan OutFrame
	done   chan struct{}

	encoder Encoder

	// frame size
	w, h int
}

// NewVideoPipe returns new video encoder pipe.
// By default, it waits for RGBA images on the input channel,
// converts them into YUV I420 format,
// encodes with provided video encoder, and
// puts the result into the output channel.
func NewVideoPipe(enc Encoder, w, h int) *VideoPipe {
	return &VideoPipe{
		Input:  make(chan InFrame, 1),
		Output: make(chan OutFrame, 2),
		done:   make(chan struct{}),

		encoder: enc,

		w: w,
		h: h,
	}
}

// Start begins video encoding pipe.
// Should be wrapped into a goroutine.
func (vp *VideoPipe) Start() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Recovered panic in encoding ", r)
		}
		close(vp.Output)
		close(vp.done)
	}()

	yuvProc := yuv.NewYuvImgProcessor(vp.w, vp.h)
	for img := range vp.Input {
		yCbCr := yuvProc.Process(img.Image).Get()
		frame := vp.encoder.Encode(yCbCr)
		if len(frame) > 0 {
			vp.Output <- OutFrame{Data: frame, Duration: img.Duration}
		}
	}
}

func (vp *VideoPipe) Stop() {
	close(vp.Input)
	<-vp.done
	if err := vp.encoder.Shutdown(); err != nil {
		log.Println("error: failed to close the encoder")
	}
}
