package vpxencoder

import (
	"fmt"
	"log"
	"time"
	"unsafe"

	"github.com/giongto35/cloud-game/pkg/config"
)

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

const chanSize = 2

// NewVpxEncoder create vp8 encoder
func NewVpxEncoder(w, h, fps, bitrate, keyframe int) (*VpxEncoder, error) {
	v := &VpxEncoder{
		Output: make(chan []byte, 5*chanSize),
		Input:  make(chan []byte, chanSize),

		IsRunning: true,
		Done:      false,
		// C
		width:            C.uint(w),
		height:           C.uint(h),
		fps:              C.int(fps),
		bitrate:          C.uint(bitrate),
		keyFrameInterval: C.int(keyframe),
		frameCount:       C.int(0),
	}

	if err := v.init(); err != nil {
		return nil, err
	}

	return v, nil
}

// VpxEncoder yuvI420 image to vp8 video
type VpxEncoder struct {
	Output chan []byte // frame
	Input  chan []byte // yuvI420

	IsRunning bool
	Done      bool
	// C
	width            C.uint
	height           C.uint
	fps              C.int
	bitrate          C.uint
	keyFrameInterval C.int
	frameCount       C.int
	vpxCodexCtx      C.vpx_codec_ctx_t
	vpxImage         C.vpx_image_t
	vpxCodexIter     C.vpx_codec_iter_t
}

func (v *VpxEncoder) init() error {
	v.frameCount = 0

	codecName := C.CString("vp8")
	encoder := C.get_vpx_encoder_by_name(codecName)
	C.free(unsafe.Pointer(codecName))

	if encoder == nil {
		return fmt.Errorf("get_vpx_encoder_by_name failed")
	}
	if C.vpx_img_alloc(&v.vpxImage, C.VPX_IMG_FMT_I420, v.width, v.height, 0) == nil {
		return fmt.Errorf("vpx_img_alloc failed")
	}

	var cfg C.vpx_codec_enc_cfg_t
	if C.call_vpx_codec_enc_config_default(encoder, &cfg) != 0 {
		return fmt.Errorf("Failed to get default codec config")
	}
	cfg.g_w = v.width
	cfg.g_h = v.height
	cfg.g_timebase.num = 1
	cfg.g_timebase.den = v.fps
	cfg.rc_target_bitrate = v.bitrate
	cfg.g_error_resilient = 1

	if C.call_vpx_codec_enc_init(&v.vpxCodexCtx, encoder, &cfg) != 0 {
		return fmt.Errorf("Failed to initialize encoder")
	}
	v.IsRunning = true
	go v.startLooping()
	return nil
}

func (v *VpxEncoder) startLooping() {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Recovered panic in encoding ", r)
		}
	}()

	for yuv := range v.Input {
		if v.Done == true {
			// The first time we see IsRunning set to false, we release and return
			v.Release()
			return
		}
		beginEncoding := time.Now()

		// Add Image
		v.vpxCodexIter = nil
		C.vpx_img_read(&v.vpxImage, unsafe.Pointer(&yuv[0]))

		var flags C.int
		if v.keyFrameInterval > 0 && v.frameCount%v.keyFrameInterval == 0 {
			flags |= C.VPX_EFLAG_FORCE_KF
		}
		if C.vpx_codec_encode(&v.vpxCodexCtx, &v.vpxImage, C.vpx_codec_pts_t(v.frameCount), 1, C.vpx_enc_frame_flags_t(flags), C.VPX_DL_REALTIME) != 0 {
			fmt.Println("Failed to encode frame")
		}
		v.frameCount++

		// Get Frame
		goBytes := C.get_frame_buffer(&v.vpxCodexCtx, &v.vpxCodexIter)
		if goBytes.bs == nil {
			continue
		}
		bs := C.GoBytes(goBytes.bs, goBytes.size)
		// if buffer is full skip frame
		if len(v.Output) >= cap(v.Output) {
			continue
		}
		v.Output <- bs

		if *config.IsMonitor {
			log.Println("Encoding time: ", time.Now().Sub(beginEncoding))
		}
	}
	if v.Done == true {
		// The first time we see IsRunning set to false, we release and return
		v.Release()
		return
	}
}

// Release release memory and stop loop
func (v *VpxEncoder) Release() {
	if v.IsRunning {
		v.IsRunning = false
		log.Println("Releasing encoder")
		C.vpx_img_free(&v.vpxImage)
		C.vpx_codec_destroy(&v.vpxCodexCtx)
		// TODO: Bug here, after close it will signal
		close(v.Output)
		if v.Input != nil {
			close(v.Input)
		}
	}
	// TODO: Can we merge IsRunning and Done together
}
