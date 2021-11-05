package h264

import "C"
import "fmt"

type H264 struct {
	ref *T

	width      int32
	lumaSize   int32
	chromaSize int32
	csp        int32
	nnals      int32
	nals       []*Nal

	// keep monotonic pts to suppress warnings
	pts int64
}

func NewEncoder(width, height int, options ...Option) (encoder *H264, err error) {
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
	param.IWidth = int32(width)
	param.IHeight = int32(height)
	param.ILogLevel = opts.LogLevel

	param.Rc.IRcMethod = RcCrf
	param.Rc.FRfConstant = float32(opts.Crf)

	encoder = &H264{
		csp:        param.ICsp,
		lumaSize:   int32(width * height),
		chromaSize: int32(width*height) / 4,
		nals:       make([]*Nal, 1),
		width:      int32(width),
	}

	if encoder.ref = EncoderOpen(&param); encoder.ref == nil {
		err = fmt.Errorf("x264: cannot open the encoder")
		return
	}
	return
}

func LibVersion() int { return int(Build) }

func (e *H264) Encode(yuv []byte) []byte {
	var picIn, picOut Picture

	picIn.Img.ICsp = e.csp
	picIn.Img.IPlane = 3
	picIn.Img.IStride[0] = e.width
	picIn.Img.IStride[1] = e.width / 2
	picIn.Img.IStride[2] = e.width / 2

	picIn.Img.Plane[0] = C.CBytes(yuv[:e.lumaSize])
	picIn.Img.Plane[1] = C.CBytes(yuv[e.lumaSize : e.lumaSize+e.chromaSize])
	picIn.Img.Plane[2] = C.CBytes(yuv[e.lumaSize+e.chromaSize:])

	picIn.IPts = e.pts
	e.pts++

	defer func() {
		picIn.freePlane(0)
		picIn.freePlane(1)
		picIn.freePlane(2)
	}()

	if ret := EncoderEncode(e.ref, e.nals, &e.nnals, &picIn, &picOut); ret > 0 {
		return C.GoBytes(e.nals[0].PPayload, C.int(ret))
	}
	return []byte{}
}

func (e *H264) Shutdown() error {
	EncoderClose(e.ref)
	return nil
}
