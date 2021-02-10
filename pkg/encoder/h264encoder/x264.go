package h264encoder

/*
#include <stdlib.h>
*/
import "C"
import (
	"fmt"
	"io"

	x264 "github.com/sergystepanov/x264-go/v2/x264c/external"
)

// Options represent encoding options.
type Options struct {
	// Frame width.
	Width int32
	// Frame height.
	Height int32
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

type Encoder struct {
	h264 *x264.T
	w    io.Writer

	width, height int32
	lumaSize      int32
	chromaSize    int32
	csp           int32
	nnals         int32
	nals          []*x264.X264NalT
}

// NewEncoder returns new x264 encoder.
func NewEncoder(w io.Writer, opts *Options) (encoder *Encoder, err error) {
	encoder = &Encoder{
		csp:        x264.X264CspI420,
		lumaSize:   opts.Width * opts.Height,
		chromaSize: opts.Width * opts.Height / 4,
		nals:       make([]*x264.X264NalT, 1),
		w:          w,
		width:      opts.Width,
		height:     opts.Height,
	}

	param := x264.X264ParamT{}
	if opts.Preset != "" && opts.Profile != "" {
		ret := x264.ParamDefaultPreset(&param, opts.Preset, opts.Tune)
		if ret < 0 {
			err = fmt.Errorf("x264: invalid preset/tune name")
			return
		}
	} else {
		x264.ParamDefault(&param)
	}

	param.IBitdepth = 8
	param.ICsp = encoder.csp
	param.IWidth = opts.Width
	param.IHeight = opts.Height
	//param.BVfrInput = 0
	param.BRepeatHeaders = 1
	param.BAnnexb = 1
	param.ILogLevel = opts.LogLevel
	//param.IKeyintMax = 60
	//param.BIntraRefresh = 1
	//param.IFpsNum = 60
	//param.IFpsDen = 1

	//param.Rc.IRcMethod = x264.X264RcCrf
	//param.Rc.FRfConstant = 23

	//param.BVfrInput = 1
	//param.ITimebaseNum = 1
	//param.ITimebaseDen = 1000

	if opts.Profile != "" {
		ret := x264.ParamApplyProfile(&param, opts.Profile)
		if ret < 0 {
			err = fmt.Errorf("x264: invalid profile name")
			return
		}
	}

	var picIn x264.Picture
	x264.PictureInit(&picIn)

	if encoder.h264 = x264.EncoderOpen(&param); encoder.h264 == nil {
		err = fmt.Errorf("x264: cannot open the encoder")
		return
	}

	if ret := x264.EncoderHeaders(encoder.h264, encoder.nals, &encoder.nnals); ret > 0 {
		_, err = encoder.w.Write(C.GoBytes(encoder.nals[0].PPayload, C.int(ret)))
	}

	return
}

func (e *Encoder) Encode(yuv []byte) (err error) {
	var picIn, picOut x264.Picture

	picIn.Img.ICsp = e.csp
	picIn.Img.IPlane = 3
	picIn.Img.IStride[0] = e.width
	picIn.Img.IStride[1] = e.width / 2
	picIn.Img.IStride[2] = e.width / 2

	picIn.Img.Plane[0] = C.CBytes(yuv[:e.lumaSize])
	picIn.Img.Plane[1] = C.CBytes(yuv[e.lumaSize : e.lumaSize+e.chromaSize])
	picIn.Img.Plane[2] = C.CBytes(yuv[e.lumaSize+e.chromaSize:])

	defer func() {
		C.free(picIn.Img.Plane[0])
		C.free(picIn.Img.Plane[1])
		C.free(picIn.Img.Plane[2])
	}()

	if ret := x264.EncoderEncode(e.h264, e.nals, &e.nnals, &picIn, &picOut); ret > 0 {
		_, err = e.w.Write(C.GoBytes(e.nals[0].PPayload, C.int(ret)))
		// ret should be equal to writer writes
	}
	return
}

func (e *Encoder) Close() error {
	x264.EncoderClose(e.h264)
	return nil
}
