package h264

/*
// See: [x264](https://www.videolan.org/developers/x264.html)
#cgo !st pkg-config: x264
#cgo st LDFLAGS: -l:libx264.a

#include "stdint.h"
#include "x264.h"
#include <stdlib.h>

typedef struct
{
	x264_t *h;
	x264_nal_t *nal; // array of NALs
	int i_nal;       // number of NALs
	int y;           // Y size
	int uv;          // U or V size
	x264_picture_t pic;
	x264_picture_t pic_out;
} h264;

h264 *h264_new(x264_param_t *param)
{
	h264 tmp;
	x264_picture_t pic;

	tmp.h = x264_encoder_open(param);
	if (!tmp.h)
		return NULL;

	x264_picture_init(&pic);
	pic.img.i_csp = param->i_csp;
	pic.img.i_plane = 3;
	pic.img.i_stride[0] = param->i_width;
	pic.img.i_stride[1] = param->i_width >> 1;
	pic.img.i_stride[2] = param->i_width >> 1;
	tmp.pic = pic;

	// crashes during x264_picture_clean :/
	//if (x264_picture_alloc(&pic, param->i_csp, param->i_width, param->i_height) < 0)
    //   return NULL;

	tmp.y = param->i_width * param->i_height;
	tmp.uv = tmp.y >> 2;

	h264 *h = malloc(sizeof(h264));
    *h = tmp;
    return h;
}

int h264_encode(h264 *h, uint8_t *yuv)
{
	h->pic.img.plane[0] = yuv;
	h->pic.img.plane[1] = h->pic.img.plane[0] + h->y;
	h->pic.img.plane[2] = h->pic.img.plane[1] + h->uv;
	h->pic.i_pts += 1;
	return x264_encoder_encode(h->h, &h->nal, &h->i_nal, &h->pic, &h->pic_out);
}

void h264_destroy(h264 *h)
{
	if (h == NULL) return;
	x264_encoder_close(h->h);
	free(h);
}
*/
import "C"

import (
	"fmt"
	"strings"
	"unsafe"
)

type H264 struct {
	h *C.h264
}

type Options struct {
	Mode string
	// Constant Rate Factor (CRF)
	// This method allows the encoder to attempt to achieve a certain output quality for the whole file
	// when output file size is of less importance.
	// The range of the CRF scale is 0–51, where 0 is lossless, 23 is the default, and 51 is the worst quality possible.
	Crf uint8
	// vbv-maxrate
	MaxRate int
	// vbv-bufsize
	BufSize  int
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
			Mode:    "crf",
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

	if strings.ToLower(opts.Mode) == "cbr" {
		param.rc.i_rc_method = C.X264_RC_ABR
		param.i_nal_hrd = C.X264_NAL_HRD_CBR
	}

	if opts.MaxRate > 0 {
		param.rc.i_bitrate = C.int(opts.MaxRate)
		param.rc.i_vbv_max_bitrate = C.int(opts.MaxRate)
	}
	if opts.BufSize > 0 {
		param.rc.i_vbv_buffer_size = C.int(opts.BufSize)
	}

	h264 := C.h264_new(&param)
	if h264 == nil {
		return nil, fmt.Errorf("x264: cannot open the encoder")
	}
	return &H264{h264}, nil
}

func (e *H264) Encode(yuv []byte) []byte {
	bytes := C.h264_encode(e.h, (*C.uchar)(unsafe.SliceData(yuv)))
	// we merge multiple NALs stored in **nal into a single byte stream
	// ret contains the total size of NALs in bytes, i.e. each e.nal[...].p_payload * i_payload
	return unsafe.Slice((*byte)(e.h.nal.p_payload), bytes)
}

func (e *H264) IntraRefresh() {
	// !to implement
}

func (e *H264) Info() string { return fmt.Sprintf("x264: v%v", Version()) }

func (e *H264) SetFlip(b bool) {
	if b {
		(*e.h).pic.img.i_csp |= C.X264_CSP_VFLIP
	} else {
		(*e.h).pic.img.i_csp &= ^C.X264_CSP_VFLIP
	}
}

func (e *H264) Shutdown() error {
	if e.h != nil {
		C.h264_destroy(e.h)
	}
	return nil
}

func Version() int { return int(C.X264_BUILD) }
