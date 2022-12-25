package h264

/*
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
*/
import "C"
import (
	"fmt"
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
}

func NewEncoder(w, h int, options ...Option) (encoder *H264, err error) {
	libVersion := LibVersion()

	if libVersion < 150 {
		return nil, fmt.Errorf("x264: the library version should be newer than v150, you have got version %v", libVersion)
	}

	opts := &Options{
		Crf:     12,
		Tune:    "zerolatency",
		Preset:  "superfast",
		Profile: "baseline",
	}

	for _, opt := range options {
		opt(opts)
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

	picIn.Img.Plane[0] = C.CBytes(make([]byte, encoder.lumaSize))
	picIn.Img.Plane[1] = C.CBytes(make([]byte, encoder.chromaSize))
	picIn.Img.Plane[2] = C.CBytes(make([]byte, encoder.chromaSize))

	encoder.in = &picIn

	if encoder.ref = EncoderOpen(&param); encoder.ref == nil {
		err = fmt.Errorf("x264: cannot open the encoder")
		return
	}
	return
}

func LibVersion() int { return int(Build) }

func (e *H264) Encode(yuv []byte) []byte {
	const x = 1 << 22
	copy((*[x]byte)(e.in.Img.Plane[0])[:e.lumaSize], yuv[:e.lumaSize])
	copy((*[x]byte)(e.in.Img.Plane[1])[:e.chromaSize], yuv[e.lumaSize:e.lumaSize+e.chromaSize])
	copy((*[x]byte)(e.in.Img.Plane[2])[:e.chromaSize], yuv[e.lumaSize+e.chromaSize:])

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
	e.in.freePlane(0)
	e.in.freePlane(1)
	e.in.freePlane(2)
	EncoderClose(e.ref)
	return nil
}
