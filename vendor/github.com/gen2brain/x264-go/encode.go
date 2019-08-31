// Package x264 provides H.264/MPEG-4 AVC codec encoder based on [x264](https://www.videolan.org/developers/x264.html) library.
package x264

import "C"

import (
	"fmt"
	"image"
	"io"

	"github.com/gen2brain/x264-go/x264c"
)

// Logging constants.
const (
	LogNone int32 = iota - 1
	LogError
	LogWarning
	LogInfo
	LogDebug
)

// Options represent encoding options.
type Options struct {
	// Frame width.
	Width int
	// Frame height.
	Height int
	// Frame rate.
	FrameRate int
	// Tunings: film, animation, grain, stillimage, psnr, ssim, fastdecode, zerolatency.
	Tune string
	// Presets: ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo.
	Preset string
	// Profiles: baseline, main, high, high10, high422, high444.
	Profile string
	// Log level.
	LogLevel int32
}

// Encoder type.
type Encoder struct {
	e *x264c.T
	w io.Writer

	img  *YCbCr
	opts *Options

	csp int32
	pts int64

	nnals int32
	nals  []*x264c.Nal
}

// NewEncoder returns new x264 encoder.
func NewEncoder(w io.Writer, opts *Options) (e *Encoder, err error) {
	e = &Encoder{}

	e.w = w
	e.pts = 0
	e.opts = opts

	e.csp = x264c.CspI420

	e.nals = make([]*x264c.Nal, 3)
	e.img = NewYCbCr(image.Rect(0, 0, e.opts.Width, e.opts.Height))

	param := x264c.Param{}

	if e.opts.Preset != "" && e.opts.Profile != "" {
		ret := x264c.ParamDefaultPreset(&param, e.opts.Preset, e.opts.Tune)
		if ret < 0 {
			err = fmt.Errorf("x264: invalid preset/tune name")
			return
		}
	} else {
		x264c.ParamDefault(&param)
	}

	param.IWidth = int32(e.opts.Width)
	param.IHeight = int32(e.opts.Height)

	param.ICsp = e.csp
	param.BVfrInput = 0
	param.BRepeatHeaders = 1
	param.BAnnexb = 1

	param.ILogLevel = e.opts.LogLevel

	if e.opts.FrameRate > 0 {
		param.IFpsNum = uint32(e.opts.FrameRate)
		param.IFpsDen = 1

		param.IKeyintMax = int32(e.opts.FrameRate)
		param.BIntraRefresh = 1
	}

	if e.opts.Profile != "" {
		ret := x264c.ParamApplyProfile(&param, e.opts.Profile)
		if ret < 0 {
			err = fmt.Errorf("x264: invalid profile name")
			return
		}
	}

	e.e = x264c.EncoderOpen(&param)
	if e.e == nil {
		err = fmt.Errorf("x264: cannot open the encoder")
		return
	}

	ret := x264c.EncoderHeaders(e.e, e.nals, &e.nnals)
	if ret < 0 {
		err = fmt.Errorf("x264: cannot encode headers")
		return
	}

	if ret > 0 {
		b := C.GoBytes(e.nals[0].PPayload, C.int(ret))
		n, er := e.w.Write(b)
		if er != nil {
			err = er
			return
		}

		if int(ret) != n {
			err = fmt.Errorf("x264: error writing headers, size=%d, n=%d", ret, n)
		}
	}

	return
}

// Encode encodes image.
func (e *Encoder) Encode(im image.Image) (err error) {
	var picIn, picOut x264c.Picture

	e.img.ToYCbCr(im)

	ret := x264c.PictureAlloc(&picIn, e.csp, int32(e.opts.Width), int32(e.opts.Height))
	if ret < 0 {
		err = fmt.Errorf("x264: cannot allocate picture")
		return
	}

	defer x264c.PictureClean(&picIn)

	picIn.Img.Plane[0] = C.CBytes(e.img.Y)
	picIn.Img.Plane[1] = C.CBytes(e.img.Cb)
	picIn.Img.Plane[2] = C.CBytes(e.img.Cr)

	picIn.IPts = e.pts
	e.pts++

	ret = x264c.EncoderEncode(e.e, e.nals, &e.nnals, &picIn, &picOut)
	if ret < 0 {
		err = fmt.Errorf("x264: cannot encode picture")
		return
	}

	if ret > 0 {
		b := C.GoBytes(e.nals[0].PPayload, C.int(ret))

		n, er := e.w.Write(b)
		if er != nil {
			err = er
			return
		}

		if int(ret) != n {
			err = fmt.Errorf("x264: error writing payload, size=%d, n=%d", ret, n)
		}
	}

	return
}

// Flush flushes encoder.
func (e *Encoder) Flush() (err error) {
	var picOut x264c.Picture

	for x264c.EncoderDelayedFrames(e.e) > 0 {
		ret := x264c.EncoderEncode(e.e, e.nals, &e.nnals, nil, &picOut)
		if ret < 0 {
			err = fmt.Errorf("x264: cannot encode picture")
			return
		}

		if ret > 0 {
			b := C.GoBytes(e.nals[0].PPayload, C.int(ret))

			n, er := e.w.Write(b)
			if er != nil {
				err = er
				return
			}

			if int(ret) != n {
				err = fmt.Errorf("x264: error writing payload, size=%d, n=%d", ret, n)
			}
		}
	}

	return
}

// Close closes encoder.
func (e *Encoder) Close() error {
	x264c.EncoderClose(e.e)
	return nil
}
