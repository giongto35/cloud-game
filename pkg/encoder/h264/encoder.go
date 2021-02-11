package h264

import (
	"bytes"
	"log"
	"runtime/debug"

	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/util"
)

const chanSize = 2

// H264Encoder yuvI420 image to vp8 video
type Encoder struct {
	Output chan encoder.OutFrame
	Input  chan encoder.InFrame
	done   chan struct{}

	buf *bytes.Buffer
	enc *H264

	// C
	width  int
	height int
	fps    int
}

// NewEncoder creates h264 encoder
func NewEncoder(width, height int) (encoder.Encoder, error) {
	v := &Encoder{
		Output: make(chan encoder.OutFrame, 5*chanSize),
		Input:  make(chan encoder.InFrame, chanSize),
		done:   make(chan struct{}),

		buf:    bytes.NewBuffer(make([]byte, 0)),
		width:  width,
		height: height,
	}

	if err := v.init(); err != nil {
		return nil, err
	}

	return v, nil
}

func (v *Encoder) init() error {
	enc, err := NewH264Encoder(v.buf, v.width, v.height)
	if err != nil {
		panic(err)
	}
	v.enc = enc

	go v.startLooping()
	return nil
}

func (v *Encoder) startLooping() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Recovered panic in encoding ", r)
			log.Println(debug.Stack())
		}
	}()

	size := int(float32(v.width*v.height) * 1.5)
	yuv := make([]byte, size, size)

	for img := range v.Input {
		util.RgbaToYuvInplace(img.Image, yuv, v.width, v.height)
		err := v.enc.Encode(yuv)
		if err != nil {
			log.Println("err encoding ", img.Image, " using h264")
		}
		v.Output <- encoder.OutFrame{Data: v.buf.Bytes(), Timestamp: img.Timestamp}
		v.buf.Reset()
	}
	close(v.Output)
	close(v.done)
}

// Release release memory and stop loop
func (v *Encoder) release() {
	close(v.Input)
	<-v.done
	err := v.enc.Close()
	if err != nil {
		log.Println("Failed to close H264 encoder")
	}
}

// GetInputChan returns input channel
func (v *Encoder) GetInputChan() chan encoder.InFrame {
	return v.Input
}

// GetInputChan returns output channel
func (v *Encoder) GetOutputChan() chan encoder.OutFrame {
	return v.Output
}

// GetDoneChan returns done channel
func (v *Encoder) Stop() {
	v.release()
}
