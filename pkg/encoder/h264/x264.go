package h264

/*
// See: [x264](https://www.videolan.org/developers/x264.html)
#cgo !st pkg-config: x264
#cgo st LDFLAGS: -l:libx264.a

#include "stdint.h"
#include "x264.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"runtime"
	"unsafe"
)

type H264 struct {
	ref *C.x264_t

	nal     *C.x264_nal_t // array of NALs
	cNal    *C.int        // number of NALs
	y       int           // Y size
	uv      int           // U or V size
	in, out *C.x264_picture_t

	p runtime.Pinner
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
	ver := Version()

	if ver < 150 {
		return nil, fmt.Errorf("x264: the library version should be newer than v150, you have got version %v", ver)
	}

	if opts == nil {
		opts = &Options{
			Crf:     23,
			Tune:    "zerolatency",
			Preset:  "superfast",
			Profile: "baseline",
		}
	}

	param := C.x264_param_t{}

	if opts.Preset != "" && opts.Tune != "" {
		preset := C.CString(opts.Preset)
		tune := C.CString(opts.Tune)
		defer C.free(unsafe.Pointer(preset))
		defer C.free(unsafe.Pointer(tune))
		if C.x264_param_default_preset(&param, preset, tune) < 0 {
			return nil, fmt.Errorf("x264: invalid preset/tune name")
		}
	} else {
		C.x264_param_default(&param)
	}

	if opts.Profile != "" {
		profile := C.CString(opts.Profile)
		defer C.free(unsafe.Pointer(profile))
		if C.x264_param_apply_profile(&param, profile) < 0 {
			return nil, fmt.Errorf("x264: invalid profile name")
		}
	}

	param.i_bitdepth = 8
	if ver > 155 {
		param.i_csp = C.X264_CSP_I420
	} else {
		param.i_csp = 1
	}
	param.i_width = C.int(w)
	param.i_height = C.int(h)
	param.i_log_level = C.int(opts.LogLevel)
	param.i_keyint_max = 120
	param.i_sync_lookahead = 0
	param.i_threads = C.int(th)
	if th != 1 {
		param.b_sliced_threads = 1
	}
	param.rc.i_rc_method = C.X264_RC_CRF
	param.rc.f_rf_constant = C.float(opts.Crf)

	encoder = &H264{
		y:    w * h,
		uv:   w * h / 4,
		cNal: new(C.int),
		nal:  new(C.x264_nal_t),
		out:  new(C.x264_picture_t),
		in: &C.x264_picture_t{
			img: C.x264_image_t{
				i_csp:    param.i_csp,
				i_plane:  3,
				i_stride: [4]C.int{0: C.int(w), 1: C.int(w >> 1), 2: C.int(w >> 1)},
			},
		},
		ref: C.x264_encoder_open(&param),
	}

	if encoder.ref == nil {
		err = fmt.Errorf("x264: cannot open the encoder")
	}
	return
}

func (e *H264) Encode(yuv []byte) []byte {
	e.in.img.plane[0] = (*C.uchar)(unsafe.Pointer(&yuv[0]))
	e.in.img.plane[1] = (*C.uchar)(unsafe.Pointer(&yuv[e.y]))
	e.in.img.plane[2] = (*C.uchar)(unsafe.Pointer(&yuv[e.y+e.uv]))

	e.in.i_pts += 1

	e.p.Pin(e.in.img.plane[0])
	e.p.Pin(e.in.img.plane[1])
	e.p.Pin(e.in.img.plane[2])

	e.p.Pin(e.nal)
	bytes := C.x264_encoder_encode(e.ref, &e.nal, e.cNal, e.in, e.out)
	e.p.Unpin()

	// we merge multiple NALs stored in **nal into a single byte stream
	// ret contains the total size of NALs in bytes, i.e. each e.nal[...].p_payload * i_payload
	return unsafe.Slice((*byte)(e.nal.p_payload), bytes)
}

func (e *H264) IntraRefresh() {
	// !to implement
}

func (e *H264) Info() string { return fmt.Sprintf("x264: v%v", Version()) }

func (e *H264) SetFlip(b bool) {
	if b {
		e.in.img.i_csp |= C.X264_CSP_VFLIP
	} else {
		e.in.img.i_csp &= ^C.X264_CSP_VFLIP
	}
}

func (e *H264) Shutdown() error {
	C.x264_encoder_close(e.ref)
	e.p.Unpin()
	return nil
}

func Version() int { return int(C.X264_BUILD) }
