// Package libyuv contains the wrapper for: https://chromium.googlesource.com/libyuv/libyuv.
// Libs are downloaded from: https://packages.macports.org/libyuv/.
package libyuv

/*
#cgo !darwin,!st LDFLAGS: -lyuv
#cgo !darwin,st LDFLAGS: -l:libyuv.a -l:libjpeg.a -l:libstdc++.a -static-libgcc

#cgo darwin CFLAGS: -DINCLUDE_LIBYUV_VERSION_H_
#cgo darwin LDFLAGS: -L${SRCDIR} -lstdc++
#cgo darwin,amd64 LDFLAGS: -lyuv_darwin_x86_64 -ljpeg -lstdc++
#cgo darwin,arm64 LDFLAGS: -lyuv_darwin_arm64 -ljpeg -lstdc++

#include <stdint.h>  // for uintptr_t and C99 types

#if !defined(LIBYUV_API)
#define LIBYUV_API
#endif  // LIBYUV_API

#ifndef INCLUDE_LIBYUV_VERSION_H_
#include "libyuv/version.h"
#else
#define LIBYUV_VERSION 1874 // darwin static libs version
#endif  // INCLUDE_LIBYUV_VERSION_H_

#ifdef __cplusplus
namespace libyuv {
extern "C" {
#endif

#define FOURCC(a, b, c, d) \
	(((uint32_t)(a)) | ((uint32_t)(b) << 8) | ((uint32_t)(c) << 16) | ((uint32_t)(d) << 24))

enum FourCC {
  FOURCC_I420 = FOURCC('I', '4', '2', '0'),
  FOURCC_ARGB = FOURCC('A', 'R', 'G', 'B'),
  FOURCC_ABGR = FOURCC('A', 'B', 'G', 'R'),
  FOURCC_RGBO = FOURCC('R', 'G', 'B', 'O'),
  FOURCC_RGBP = FOURCC('R', 'G', 'B', 'P'),  // rgb565 LE.
  FOURCC_ANY = -1,
};

typedef enum RotationMode {
 kRotate0 = 0,      // No rotation.
 kRotate90 = 90,    // Rotate 90 degrees clockwise.
 kRotate180 = 180,  // Rotate 180 degrees.
 kRotate270 = 270,  // Rotate 270 degrees clockwise.
} RotationModeEnum;

LIBYUV_API
int ConvertToI420(const uint8_t* sample,
                 size_t sample_size,
                 uint8_t* dst_y,
                 int dst_stride_y,
                 uint8_t* dst_u,
                 int dst_stride_u,
                 uint8_t* dst_v,
                 int dst_stride_v,
                 int crop_x,
                 int crop_y,
                 int src_width,
                 int src_height,
                 int crop_width,
                 int crop_height,
                 enum RotationMode rotation,
                 uint32_t fourcc);

// Supported filtering.
typedef enum FilterMode {
    kFilterNone = 0,      // Point sample; Fastest.
    kFilterLinear = 1,    // Filter horizontally only.
    kFilterBilinear = 2,  // Faster than box, but lower quality scaling down.
    kFilterBox = 3        // Highest quality.
} FilterModeEnum;

LIBYUV_API
int I420Scale(const uint8_t *src_y,
              int src_stride_y,
              const uint8_t *src_u,
              int src_stride_u,
              const uint8_t *src_v,
              int src_stride_v,
              int src_width,
              int src_height,
              uint8_t *dst_y,
              int dst_stride_y,
              uint8_t *dst_u,
              int dst_stride_u,
              uint8_t *dst_v,
              int dst_stride_v,
              int dst_width,
              int dst_height,
              enum FilterMode filtering);

#ifdef __cplusplus
}  // extern "C"
}  // namespace libyuv
#endif
*/
import "C"
import "fmt"

const FourccRgbp uint32 = C.FOURCC_RGBP
const FourccArgb uint32 = C.FOURCC_ARGB
const FourccAbgr uint32 = C.FOURCC_ABGR
const FourccRgb0 uint32 = C.FOURCC_RGBO

func Y420(src []byte, dst []byte, _, h, stride int, dw, dh int, rot uint, pix uint32, cx, cy int) {
	cw := (dw + 1) / 2
	ch := (dh + 1) / 2
	i0 := dw * dh
	i1 := i0 + cw*ch
	yStride := dw
	cStride := cw

	C.ConvertToI420(
		(*C.uchar)(&src[0]),
		C.size_t(0),
		(*C.uchar)(&dst[0]),
		C.int(yStride),
		(*C.uchar)(&dst[i0]),
		C.int(cStride),
		(*C.uchar)(&dst[i1]),
		C.int(cStride),
		C.int(0),
		C.int(0),
		C.int(stride),
		C.int(h),
		C.int(cx),
		C.int(cy),
		C.enum_RotationMode(rot),
		C.uint32_t(pix))
}

func Y420Scale(src []byte, dst []byte, w, h int, dw, dh int) {
	srcWidthUV, dstWidthUV := (w+1)>>1, (dw+1)>>1
	srcHeightUV, dstHeightUV := (h+1)>>1, (dh+1)>>1

	srcYPlaneSize, dstYPlaneSize := w*h, dw*dh
	srcUVPlaneSize, dstUVPlaneSize := srcWidthUV*srcHeightUV, dstWidthUV*dstHeightUV

	srcStrideY, dstStrideY := w, dw
	srcStrideU, dstStrideU := srcWidthUV, dstWidthUV
	srcStrideV, dstStrideV := srcWidthUV, dstWidthUV

	srcY := (*C.uchar)(&src[0])
	srcU := (*C.uchar)(&src[srcYPlaneSize])
	srcV := (*C.uchar)(&src[srcYPlaneSize+srcUVPlaneSize])

	dstY := (*C.uchar)(&dst[0])
	dstU := (*C.uchar)(&dst[dstYPlaneSize])
	dstV := (*C.uchar)(&dst[dstYPlaneSize+dstUVPlaneSize])

	C.I420Scale(
		srcY,
		C.int(srcStrideY),
		srcU,
		C.int(srcStrideU),
		srcV,
		C.int(srcStrideV),
		C.int(w),
		C.int(h),
		dstY,
		C.int(dstStrideY),
		dstU,
		C.int(dstStrideU),
		dstV,
		C.int(dstStrideV),
		C.int(dw),
		C.int(dh),
		C.enum_FilterMode(C.kFilterNone))
}

func Version() string { return fmt.Sprintf("%v", int(C.LIBYUV_VERSION)) }
