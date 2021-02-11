package h264

// #include <stdlib.h>
import "C"
import (
	"fmt"
	"io"

	x264 "github.com/sergystepanov/x264-go/v2/x264c/external"
)

type H264 struct {
	ref *x264.T
	w   io.Writer

	width      int32
	lumaSize   int32
	chromaSize int32
	csp        int32
	nnals      int32
	nals       []*x264.Nal
}

func NewH264Encoder(w io.Writer, width, height int, options ...Option) (encoder *H264, err error) {
	opts := &Options{
		Tune:    "zerolatency",
		Preset:  "superfast",
		Profile: "baseline",
	}

	for _, opt := range options {
		opt(opts)
	}

	param := x264.Param{}
	if opts.Preset != "" && opts.Tune != "" {
		ret := x264.ParamDefaultPreset(&param, opts.Preset, opts.Tune)
		if ret < 0 {
			err = fmt.Errorf("x264: invalid preset/tune name")
			return
		}
	} else {
		x264.ParamDefault(&param)
	}

	if opts.Profile != "" {
		ret := x264.ParamApplyProfile(&param, opts.Profile)
		if ret < 0 {
			err = fmt.Errorf("x264: invalid profile name")
			return
		}
	}

	// legacy encoder lacks of this param
	param.IBitdepth = 8
	param.ICsp = x264.CspI420
	param.IWidth = int32(width)
	param.IHeight = int32(height)
	param.BRepeatHeaders = 1
	param.BAnnexb = 1
	param.ILogLevel = opts.LogLevel

	//param.Rc.IRcMethod = x264.X264RcCrf
	//param.Rc.FRfConstant = 23

	encoder = &H264{
		csp:        param.ICsp,
		lumaSize:   int32(width * height),
		chromaSize: int32(width*height) / 4,
		nals:       make([]*x264.Nal, 1),
		w:          w,
		width:      int32(width),
	}

	var picIn x264.Picture
	x264.PictureInit(&picIn)

	if encoder.ref = x264.EncoderOpen(&param); encoder.ref == nil {
		err = fmt.Errorf("x264: cannot open the encoder")
		return
	}

	if ret := x264.EncoderHeaders(encoder.ref, encoder.nals, &encoder.nnals); ret > 0 {
		_, err = encoder.w.Write(C.GoBytes(encoder.nals[0].PPayload, C.int(ret)))
	}

	return
}

func (e *H264) Encode(yuv []byte) (err error) {
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

	if ret := x264.EncoderEncode(e.ref, e.nals, &e.nnals, &picIn, &picOut); ret > 0 {
		_, err = e.w.Write(C.GoBytes(e.nals[0].PPayload, C.int(ret)))
		// ret should be equal to writer writes
	}
	return
}

func (e *H264) Close() error {
	x264.EncoderClose(e.ref)
	return nil
}
