package vpx

/*
#cgo pkg-config: vpx
#cgo st LDFLAGS: -l:libvpx.a

#include "vpx/vpx_encoder.h"
#include "vpx/vpx_image.h"
#include "vpx/vp8cx.h"

#include <stdlib.h>
#include <string.h>

#define VP8_FOURCC 0x30385056
#define VP9_FOURCC 0x30395056

typedef struct VpxInterface {
  const char *const name;
  const uint32_t fourcc;
  vpx_codec_iface_t *(*const codec_interface)();
} VpxInterface;

typedef struct FrameBuffer {
  void *ptr;
  int size;
} FrameBuffer;

vpx_codec_err_t call_vpx_codec_enc_config_default(const VpxInterface *encoder, vpx_codec_enc_cfg_t *cfg) {
	return vpx_codec_enc_config_default(encoder->codec_interface(), cfg, 0);
}
vpx_codec_err_t call_vpx_codec_enc_init(vpx_codec_ctx_t *codec, const VpxInterface *encoder, vpx_codec_enc_cfg_t *cfg) {
	return vpx_codec_enc_init(codec, encoder->codec_interface(), cfg, 0);
}

FrameBuffer get_frame_buffer(vpx_codec_ctx_t *codec, vpx_codec_iter_t *iter) {
    // iter has set to NULL when after add new image
    FrameBuffer fb = {NULL, 0};
    const vpx_codec_cx_pkt_t *pkt = vpx_codec_get_cx_data(codec, iter);
	if (pkt != NULL && pkt->kind == VPX_CODEC_CX_FRAME_PKT) {
		fb.ptr = pkt->data.frame.buf;
		fb.size = pkt->data.frame.sz;
	}
    return fb;
}

const VpxInterface vpx_encoders[] = {
	{ "vp8", VP8_FOURCC, &vpx_codec_vp8_cx },
 	{ "vp9", VP9_FOURCC, &vpx_codec_vp9_cx },
};

int vpx_img_plane_width(const vpx_image_t *img, int plane) {
	if (plane > 0 && img->x_chroma_shift > 0)
		return (img->d_w + 1) >> img->x_chroma_shift;
	else
		return img->d_w;
}

int vpx_img_plane_height(const vpx_image_t *img, int plane) {
	if (plane > 0 && img->y_chroma_shift > 0)
		return (img->d_h + 1) >> img->y_chroma_shift;
	else
		return img->d_h;
}

void vpx_img_read(vpx_image_t *dst, void *src) {
	for (int plane = 0; plane < 3; ++plane) {
		unsigned char *buf = dst->planes[plane];
		const int stride = dst->stride[plane];
		const int w = vpx_img_plane_width(dst, plane);
		const int h = vpx_img_plane_height(dst, plane);

		for (int y = 0; y < h; ++y) {
			memcpy(buf, src, w);
			buf += stride;
			src += w;
		}
	}
}
*/
import "C"
import (
	"fmt"
	"unsafe"
)

type Vpx struct {
	frameCount C.int
	image      C.vpx_image_t
	codecCtx   C.vpx_codec_ctx_t
	kfi        C.int
	flipped    bool
	v          int
}

func (vpx *Vpx) SetFlip(b bool) { vpx.flipped = b }

type Options struct {
	// Target bandwidth to use for this stream, in kilobits per second.
	Bitrate uint
	// Force keyframe interval.
	KeyframeInterval uint
}

func NewEncoder(w, h int, th int, version int, opts *Options) (*Vpx, error) {
	idx := 0
	if version == 9 {
		idx = 1
	}
	encoder := &C.vpx_encoders[idx]
	if encoder == nil {
		return nil, fmt.Errorf("couldn't get the encoder")
	}

	if opts == nil {
		opts = &Options{
			Bitrate:          1200,
			KeyframeInterval: 5,
		}
	}

	vpx := Vpx{
		frameCount: C.int(0),
		kfi:        C.int(opts.KeyframeInterval),
		v:          version,
	}

	if C.vpx_img_alloc(&vpx.image, C.VPX_IMG_FMT_I420, C.uint(w), C.uint(h), 1) == nil {
		return nil, fmt.Errorf("vpx_img_alloc failed")
	}

	var cfg C.vpx_codec_enc_cfg_t
	if C.call_vpx_codec_enc_config_default(encoder, &cfg) != 0 {
		return nil, fmt.Errorf("failed to get default codec config")
	}

	cfg.g_w = C.uint(w)
	cfg.g_h = C.uint(h)
	if th != 0 {
		cfg.g_threads = C.uint(th)
	}
	cfg.g_lag_in_frames = 0
	cfg.rc_target_bitrate = C.uint(opts.Bitrate)
	cfg.g_error_resilient = C.VPX_ERROR_RESILIENT_DEFAULT

	if C.call_vpx_codec_enc_init(&vpx.codecCtx, encoder, &cfg) != 0 {
		return nil, fmt.Errorf("failed to initialize encoder")
	}

	return &vpx, nil
}

// Encode encodes yuv image with the VPX8 encoder.
// see: https://chromium.googlesource.com/webm/libvpx/+/master/examples/simple_encoder.c
func (vpx *Vpx) Encode(yuv []byte) []byte {
	C.vpx_img_read(&vpx.image, unsafe.Pointer(&yuv[0]))
	if vpx.flipped {
		C.vpx_img_flip(&vpx.image)
	}

	var flags C.int
	if vpx.kfi > 0 && vpx.frameCount%vpx.kfi == 0 {
		flags |= C.VPX_EFLAG_FORCE_KF
	}
	if C.vpx_codec_encode(&vpx.codecCtx, &vpx.image, C.vpx_codec_pts_t(vpx.frameCount), 1, C.vpx_enc_frame_flags_t(flags), C.VPX_DL_REALTIME) != 0 {
		fmt.Println("Failed to encode frame")
	}
	vpx.frameCount++

	var iter C.vpx_codec_iter_t
	fb := C.get_frame_buffer(&vpx.codecCtx, &iter)
	if fb.ptr == nil {
		return []byte{}
	}
	return C.GoBytes(fb.ptr, fb.size)
}

func (vpx *Vpx) Info() string {
	return fmt.Sprintf("vpx (%v): %v", vpx.v, C.GoString(C.vpx_codec_version_str()))
}

func (vpx *Vpx) IntraRefresh() {
	// !to implement
}

func (vpx *Vpx) Shutdown() error {
	//if &vpx.image != nil {
	C.vpx_img_free(&vpx.image)
	//}
	C.vpx_codec_destroy(&vpx.codecCtx)
	vpx.flipped = false
	return nil
}
