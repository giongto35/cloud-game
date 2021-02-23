package h264

import (
	"bytes"
	"log"
	"runtime/debug"

	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/util"
)

// Encoder converts a yuvI420 image to h264 frame.
type Encoder struct {
	Output chan encoder.OutFrame
	Input  chan encoder.InFrame
	done   chan struct{}

	buf *bytes.Buffer
	enc *H264

	width  int
	height int
}

// NewEncoder creates h264 encoder
func NewEncoder(width, height int, options ...Option) (encoder.Encoder, error) {
	enc := &Encoder{
		Output: make(chan encoder.OutFrame, 10),
		Input:  make(chan encoder.InFrame, 2),
		done:   make(chan struct{}),

		buf:    bytes.NewBuffer(make([]byte, 0)),
		width:  width,
		height: height,
	}

	if err := enc.init(options...); err != nil {
		return nil, err
	}

	return enc, nil
}

func (e *Encoder) init(options ...Option) error {
	enc, err := NewH264Encoder(e.buf, e.width, e.height, options...)
	if err != nil {
		panic(err)
	}
	e.enc = enc

	go e.startLooping()
	return nil
}

func (e *Encoder) startLooping() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Recovered panic in encoding ", r)
			log.Println(debug.Stack())
		}
	}()

	size := int(float32(e.width*e.height) * 1.5)
	yuv := make([]byte, size, size)

	for img := range e.Input {
		util.RgbaToYuvInplace(img.Image, yuv, e.width, e.height)
		err := e.enc.Encode(yuv)
		if err != nil {
			log.Println("err encoding ", img.Image, " using h264")
		}
		e.Output <- encoder.OutFrame{Data: e.buf.Bytes(), Timestamp: img.Timestamp}
		e.buf.Reset()
	}
	close(e.Output)
	close(e.done)
}

// Release release memory and stop loop
func (e *Encoder) release() {
	close(e.Input)
	<-e.done
	err := e.enc.Close()
	if err != nil {
		log.Println("Failed to close H264 encoder")
	}
}

func (e *Encoder) GetInputChan() chan encoder.InFrame { return e.Input }

func (e *Encoder) GetOutputChan() chan encoder.OutFrame { return e.Output }

func (e *Encoder) Stop() { e.release() }
