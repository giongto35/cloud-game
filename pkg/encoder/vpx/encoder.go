package vpx

import (
	"log"

	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/vpx/libvpx"
	"github.com/giongto35/cloud-game/v2/pkg/encoder/vpx/options"
	"github.com/giongto35/cloud-game/v2/pkg/util"
)

// Encoder converts yuvI420 image to a vp8 video frame.
type Encoder struct {
	Output chan encoder.OutFrame
	Input  chan encoder.InFrame
	done   chan struct{}

	enc *libvpx.Vpx

	width  int
	height int
}

func NewEncoder(w, h int, options ...options.Option) (encoder.Encoder, error) {
	v := &Encoder{
		Output: make(chan encoder.OutFrame, 10),
		Input:  make(chan encoder.InFrame, 2),
		done:   make(chan struct{}),

		width:  w,
		height: h,
	}

	if err := v.init(options...); err != nil {
		return nil, err
	}

	return v, nil
}

func (v *Encoder) init(options ...options.Option) error {
	enc, err := libvpx.NewEncoder(v.width, v.height, options...)
	if err != nil {
		return err
	}
	v.enc = enc

	go v.startLooping()

	return nil
}

func (v *Encoder) startLooping() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Recovered panic in encoding ", r)
		}
	}()

	size := int(float32(v.width*v.height) * 1.5)
	yuv := make([]byte, size, size)

	for img := range v.Input {
		util.RgbaToYuvInplace(img.Image, yuv, v.width, v.height)

		result, err := v.enc.Encode(yuv)
		if err != nil {
			log.Printf("frame encoding error, skip")
		}

		if len(result) == 0 {
			continue
		}

		// if buffer is full skip frame
		if len(v.Output) >= cap(v.Output) {
			continue
		}

		v.Output <- encoder.OutFrame{Data: result, Timestamp: img.Timestamp}
	}
	close(v.Output)
	close(v.done)
}

func (v *Encoder) release() {
	close(v.Input)
	<-v.done
	if v.enc.Close() != nil {
		log.Println("Failed to close VPX encoder")
	}
}

func (v *Encoder) GetInputChan() chan encoder.InFrame { return v.Input }

func (v *Encoder) GetOutputChan() chan encoder.OutFrame { return v.Output }

func (v *Encoder) Stop() { v.release() }
