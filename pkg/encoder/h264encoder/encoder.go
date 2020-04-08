package h264encoder

import (
	"bytes"
	"image"
	"log"
	"runtime/debug"

	"github.com/gen2brain/x264-go"
	"github.com/giongto35/cloud-game/pkg/encoder"
)

const chanSize = 2

// H264Encoder yuvI420 image to vp8 video
type H264Encoder struct {
	Output chan []byte      // frame
	Input  chan *image.RGBA // yuvI420

	buf *bytes.Buffer
	enc *x264.Encoder

	// C
	width  int
	height int
	fps    int
}

// NewH264Encoder create h264 encoder
func NewH264Encoder(width, height, fps int) (encoder.Encoder, error) {
	v := &H264Encoder{
		Output: make(chan []byte, 5*chanSize),
		Input:  make(chan *image.RGBA, chanSize),

		buf:    bytes.NewBuffer(make([]byte, 0)),
		width:  width,
		height: height,
		fps:    fps,
	}

	if err := v.init(); err != nil {
		return nil, err
	}

	return v, nil
}

func (v *H264Encoder) init() error {
	opts := &x264.Options{
		Width:     v.width,
		Height:    v.height,
		FrameRate: v.fps,
		Tune:      "zerolatency",
		Preset:    "veryfast",
		Profile:   "baseline",
		//LogLevel:  x264.LogDebug,
	}

	enc, err := x264.NewEncoder(v.buf, opts)
	if err != nil {
		panic(err)
	}
	v.enc = enc

	go v.startLooping()
	return nil
}

func (v *H264Encoder) startLooping() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Recovered panic in encoding ", r)
			log.Println(debug.Stack())
		}
	}()

	for img := range v.Input {
		err := v.enc.Encode(img)
		if err != nil {
			log.Println("err encoding ", img, " using h264")
		}
		v.Output <- v.buf.Bytes()
		v.buf.Reset()
	}
}

// Release release memory and stop loop
func (v *H264Encoder) release() {
	log.Println("Releasing encoder")
	// TODO: Bug here, after close it will signal
	close(v.Output)
	err := v.enc.Close()
	if err != nil {
		log.Println("Failed to close H264 encoder")
	}
}

// GetInputChan returns input channel
func (v *H264Encoder) GetInputChan() chan *image.RGBA {
	return v.Input
}

// GetInputChan returns output channel
func (v *H264Encoder) GetOutputChan() chan []byte {
	return v.Output
}

// GetDoneChan returns done channel
func (v *H264Encoder) Stop() {
	v.release()
}
