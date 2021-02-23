package libvpx

// https://chromium.googlesource.com/webm/libvpx/+/master/examples/simple_encoder.c

/*
#cgo pkg-config: vpx
#include <stdlib.h>
#include "vpx/vpx_encoder.h"
#include "tools_common.h"

typedef struct GoBytes {
  void *bs;
  int size;
} GoBytesType;

vpx_codec_err_t call_vpx_codec_enc_config_default(const VpxInterface *encoder, vpx_codec_enc_cfg_t *cfg) {
	return vpx_codec_enc_config_default(encoder->codec_interface(), cfg, 0);
}
vpx_codec_err_t call_vpx_codec_enc_init(vpx_codec_ctx_t *codec, const VpxInterface *encoder, vpx_codec_enc_cfg_t *cfg) {
	return vpx_codec_enc_init(codec, encoder->codec_interface(), cfg, 0);
}
GoBytesType get_frame_buffer(vpx_codec_ctx_t *codec, vpx_codec_iter_t *iter) {
	// iter has set to NULL when after add new image
	GoBytesType bytes = {NULL, 0};
  const vpx_codec_cx_pkt_t *pkt = vpx_codec_get_cx_data(codec, iter);
	if (pkt != NULL && pkt->kind == VPX_CODEC_CX_FRAME_PKT) {
		bytes.bs = pkt->data.frame.buf;
		bytes.size = pkt->data.frame.sz;
	}
  return bytes;
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
	codecIter  C.vpx_codec_iter_t

	kfi C.int
}

func NewEncoder(width, height, fps, bitrate, kfi int) (*Vpx, error) {
	codec := C.CString("vp8")
	defer C.free(unsafe.Pointer(codec))
	encoder := C.get_vpx_encoder_by_name(codec)
	if encoder == nil {
		return nil, fmt.Errorf("get_vpx_encoder_by_name failed")
	}

	vpx := Vpx{
		frameCount: C.int(0),
		kfi:        C.int(kfi),
	}

	if C.vpx_img_alloc(&vpx.image, C.VPX_IMG_FMT_I420, C.uint(width), C.uint(height), 0) == nil {
		return nil, fmt.Errorf("vpx_img_alloc failed")
	}

	var cfg C.vpx_codec_enc_cfg_t
	if C.call_vpx_codec_enc_config_default(encoder, &cfg) != 0 {
		return nil, fmt.Errorf("failed to get default codec config")
	}

	cfg.g_w = C.uint(width)
	cfg.g_h = C.uint(height)
	cfg.g_timebase.num = 1
	cfg.g_timebase.den = C.int(fps)
	cfg.rc_target_bitrate = C.uint(bitrate)
	cfg.g_error_resilient = 1

	if C.call_vpx_codec_enc_init(&vpx.codecCtx, encoder, &cfg) != 0 {
		return nil, fmt.Errorf("failed to initialize encoder")
	}

	return &vpx, nil
}

func (vpx *Vpx) Encode(yuv []byte) ([]byte, error) {
	vpx.codecIter = nil
	C.vpx_img_read(&vpx.image, unsafe.Pointer(&yuv[0]))

	var flags C.int
	if vpx.kfi > 0 && vpx.frameCount%vpx.kfi == 0 {
		flags |= C.VPX_EFLAG_FORCE_KF
	}
	if C.vpx_codec_encode(&vpx.codecCtx, &vpx.image, C.vpx_codec_pts_t(vpx.frameCount), 1, C.vpx_enc_frame_flags_t(flags), C.VPX_DL_REALTIME) != 0 {
		fmt.Println("Failed to encode frame")
	}
	vpx.frameCount++

	goBytes := C.get_frame_buffer(&vpx.codecCtx, &vpx.codecIter)
	if goBytes.bs == nil {
		return []byte{}, nil
	}
	return C.GoBytes(goBytes.bs, goBytes.size), nil
}

func (vpx *Vpx) Close() error {
	C.vpx_img_free(&vpx.image)
	C.vpx_codec_destroy(&vpx.codecCtx)
	return nil
}
