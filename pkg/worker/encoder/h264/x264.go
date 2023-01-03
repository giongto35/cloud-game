package h264

/*
#include <stdint.h>
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type H264 struct {
	ref *T

	width      int32
	lumaSize   int32
	chromaSize int32
	csp        int32
	nnals      int32
	nals       []*Nal

	in, out *Picture
	y, u, v []byte
}

type Options struct {
	// Constant Rate Factor (CRF)
	// This method allows the encoder to attempt to achieve a certain output quality for the whole file
	// when output file size is of less importance.
	// The range of the CRF scale is 0–51, where 0 is lossless, 23 is the default, and 51 is the worst quality possible.
	Crf uint8
	// film, animation, grain, stillimage, psnr, ssim, fastdecode, zerolatency.
	Tune string
	// ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo.
	Preset string
	// baseline, main, high, high10, high422, high444.
	Profile  string
	LogLevel int32
}

func NewEncoder(w, h int, opts *Options) (encoder *H264, err error) {
	libVersion := LibVersion()

	if libVersion < 150 {
		return nil, fmt.Errorf("x264: the library version should be newer than v150, you have got version %v", libVersion)
	}

	if opts == nil {
		opts = &Options{
			Crf:     23,
			Tune:    "zerolatency",
			Preset:  "superfast",
			Profile: "baseline",
		}
	}

	param := Param{}
	if opts.Preset != "" && opts.Tune != "" {
		if ParamDefaultPreset(&param, opts.Preset, opts.Tune) < 0 {
			return nil, fmt.Errorf("x264: invalid preset/tune name")
		}
	} else {
		ParamDefault(&param)
	}

	if opts.Profile != "" {
		if ParamApplyProfile(&param, opts.Profile) < 0 {
			return nil, fmt.Errorf("x264: invalid profile name")
		}
	}

	// legacy encoder lacks of this param
	param.IBitdepth = 8

	if libVersion > 155 {
		param.ICsp = CspI420
	} else {
		param.ICsp = 1
	}
	param.IWidth = int32(w)
	param.IHeight = int32(h)
	param.ILogLevel = opts.LogLevel

	param.Rc.IRcMethod = RcCrf
	param.Rc.FRfConstant = float32(opts.Crf)

	encoder = &H264{
		csp:        param.ICsp,
		lumaSize:   int32(w * h),
		chromaSize: int32(w*h) / 4,
		nals:       make([]*Nal, 1),
		width:      int32(w),
		out:        new(Picture),
	}

	// pool
	var picIn Picture

	picIn.Img.ICsp = encoder.csp
	picIn.Img.IPlane = 3
	picIn.Img.IStride[0] = encoder.width
	picIn.Img.IStride[1] = encoder.width >> 1
	picIn.Img.IStride[2] = encoder.width >> 1

	picIn.Img.Plane[0] = C.malloc(C.size_t(encoder.lumaSize))
	picIn.Img.Plane[1] = C.malloc(C.size_t(encoder.chromaSize))
	picIn.Img.Plane[2] = C.malloc(C.size_t(encoder.chromaSize))

	encoder.y = unsafe.Slice((*byte)(picIn.Img.Plane[0]), encoder.lumaSize)
	encoder.u = unsafe.Slice((*byte)(picIn.Img.Plane[1]), encoder.chromaSize)
	encoder.v = unsafe.Slice((*byte)(picIn.Img.Plane[2]), encoder.chromaSize)

	encoder.in = &picIn

	if encoder.ref = EncoderOpen(&param); encoder.ref == nil {
		err = fmt.Errorf("x264: cannot open the encoder")
		return
	}
	return
}

func LibVersion() int { return int(Build) }

func (e *H264) LoadBuf(yuv []byte) {
	copy(e.y, yuv[:e.lumaSize])
	copy(e.u, yuv[e.lumaSize:e.lumaSize+e.chromaSize])
	copy(e.v, yuv[e.lumaSize+e.chromaSize:])
}

func (e *H264) Encode() []byte {
	e.in.IPts += 1
	if ret := EncoderEncode(e.ref, e.nals, &e.nnals, e.in, e.out); ret > 0 {
		return C.GoBytes(e.nals[0].PPayload, C.int(ret))
	}
	return []byte{}
}

func (e *H264) IntraRefresh() {
	// !to implement
}

func (e *H264) Shutdown() error {
	e.y = nil
	e.u = nil
	e.v = nil
	e.in.freePlanes()
	EncoderClose(e.ref)
	return nil
}
