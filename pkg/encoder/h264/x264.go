package h264

import (
	"fmt"
	"unsafe"
)

type H264 struct {
	ref *T

	width      int32
	lumaSize   int32
	chromaSize int32
	nnals      int32
	nals       []*Nal

	in, out *Picture
}

type Options struct {
	// Constant Rate Factor (CRF)
	// This method allows the encoder to attempt to achieve a certain output quality for the whole file
	// when output file size is of less importance.
	// The range of the CRF scale is 0–51, where 0 is lossless, 23 is the default, and 51 is the worst quality possible.
	Crf      uint8
	LogLevel int32
	// ultrafast, superfast, veryfast, faster, fast, medium, slow, slower, veryslow, placebo.
	Preset string
	// baseline, main, high, high10, high422, high444.
	Profile string
	// film, animation, grain, stillimage, psnr, ssim, fastdecode, zerolatency.
	Tune string
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
	param.ISyncLookahead = 0
	param.IThreads = 1

	param.Rc.IRcMethod = RcCrf
	param.Rc.FRfConstant = float32(opts.Crf)

	encoder = &H264{
		lumaSize:   param.IWidth * param.IHeight,
		chromaSize: param.IWidth * param.IHeight / 4,
		nals:       make([]*Nal, 1),
		width:      param.IWidth,
		out:        new(Picture),
		in: &Picture{
			Img: Image{
				ICsp:   param.ICsp,
				IPlane: 3,
				IStride: [4]int32{
					0: param.IWidth,
					1: param.IWidth >> 1,
					2: param.IWidth >> 1,
				},
			},
		},
	}

	if encoder.ref = EncoderOpen(&param); encoder.ref == nil {
		err = fmt.Errorf("x264: cannot open the encoder")
	}
	return
}

func LibVersion() int { return int(Build) }

func (e *H264) LoadBuf(yuv []byte) {
	e.in.Img.Plane[0] = uintptr(unsafe.Pointer(&yuv[0]))
	e.in.Img.Plane[1] = uintptr(unsafe.Pointer(&yuv[e.lumaSize]))
	e.in.Img.Plane[2] = uintptr(unsafe.Pointer(&yuv[e.lumaSize+e.chromaSize]))
}

func (e *H264) Encode() []byte {
	e.in.IPts += 1
	if ret := EncoderEncode(e.ref, e.nals, &e.nnals, e.in, e.out); ret > 0 {
		return unsafe.Slice((*byte)(e.nals[0].PPayload), ret)
		//return C.GoBytes(e.nals[0].PPayload, C.int(ret))
	}
	return []byte{}
}

func (e *H264) IntraRefresh() {
	// !to implement
}

func (e *H264) SetFlip(b bool) {
	if b {
		e.in.Img.ICsp |= CspVflip
	} else {
		e.in.Img.ICsp &= ^CspVflip
	}
}

func (e *H264) Shutdown() error {
	EncoderClose(e.ref)
	return nil
}
