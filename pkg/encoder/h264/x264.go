package h264

import (
	"fmt"
	"unsafe"
)

type H264 struct {
	ref *T

	pnals   *Nal  // array of NALs
	nnals   int32 // number of NALs
	y       int32 // Y size
	uv      int32 // U or V size
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

func NewEncoder(w, h int, th int, opts *Options) (encoder *H264, err error) {
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

	ww, hh := int32(w), int32(h)

	param.IBitdepth = 8
	if libVersion > 155 {
		param.ICsp = CspI420
	} else {
		param.ICsp = 1
	}
	param.IWidth = ww
	param.IHeight = hh
	param.ILogLevel = opts.LogLevel
	param.ISyncLookahead = 0
	param.IThreads = int32(th)
	if th != 1 {
		param.BSlicedThreads = 1
	}
	param.Rc.IRcMethod = RcCrf
	param.Rc.FRfConstant = float32(opts.Crf)

	encoder = &H264{
		y:     ww * hh,
		uv:    ww * hh / 4,
		pnals: new(Nal),
		out:   new(Picture),
		in: &Picture{
			Img: Image{ICsp: param.ICsp, IPlane: 3, IStride: [4]int32{0: ww, 1: ww >> 1, 2: ww >> 1}},
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
	e.in.Img.Plane[1] = uintptr(unsafe.Pointer(&yuv[e.y]))
	e.in.Img.Plane[2] = uintptr(unsafe.Pointer(&yuv[e.y+e.uv]))
}

func (e *H264) Encode() (b []byte) {
	e.in.IPts += 1
	bytes := EncoderEncode(e.ref, &e.pnals, &e.nnals, e.in, e.out)
	if bytes > 0 {
		// we merge multiple NALs stored in **pnals into a single byte stream
		// ret contains the total size of NALs in bytes, i.e. each e.pnals[...].PPayload * IPayload
		b = unsafe.Slice((*byte)(e.pnals.PPayload), bytes)
	}
	return
}

func (e *H264) IntraRefresh() {
	// !to implement
}

func (e *H264) Info() string { return fmt.Sprintf("x264: v%v", LibVersion()) }

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
