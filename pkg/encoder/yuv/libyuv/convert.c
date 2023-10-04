/*
 *  Copyright 2011 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#include "convert.h"

#include "basic_types.h"
#include "cpu_id.h"
#include "planar_functions.h"
#include "row.h"

// Subsample amount uses a shift.
//   v is value
//   a is amount to add to round up
//   s is shift to subsample down
#define SUBSAMPLE(v, a, s) (v < 0) ? (-((-v + a) >> s)) : ((v + a) >> s)

static __inline int Abs(int v) {
    return v >= 0 ? v : -v;
}

// Copy I420 with optional flipping.
// TODO(fbarchard): Use Scale plane which supports mirroring, but ensure
// is does row coalescing.
LIBYUV_API
int I420Copy(const uint8_t *src_y,
             int src_stride_y,
             const uint8_t *src_u,
             int src_stride_u,
             const uint8_t *src_v,
             int src_stride_v,
             uint8_t *dst_y,
             int dst_stride_y,
             uint8_t *dst_u,
             int dst_stride_u,
             uint8_t *dst_v,
             int dst_stride_v,
             int width,
             int height) {
    int halfwidth = (width + 1) >> 1;
    int halfheight = (height + 1) >> 1;
    if ((!src_y && dst_y) || !src_u || !src_v || !dst_u || !dst_v || width <= 0 ||
        height == 0) {
        return -1;
    }
    // Negative height means invert the image.
    if (height < 0) {
        height = -height;
        halfheight = (height + 1) >> 1;
        src_y = src_y + (height - 1) * src_stride_y;
        src_u = src_u + (halfheight - 1) * src_stride_u;
        src_v = src_v + (halfheight - 1) * src_stride_v;
        src_stride_y = -src_stride_y;
        src_stride_u = -src_stride_u;
        src_stride_v = -src_stride_v;
    }

    if (dst_y) {
        CopyPlane(src_y, src_stride_y, dst_y, dst_stride_y, width, height);
    }
    // Copy UV planes.
    CopyPlane(src_u, src_stride_u, dst_u, dst_stride_u, halfwidth, halfheight);
    CopyPlane(src_v, src_stride_v, dst_v, dst_stride_v, halfwidth, halfheight);
    return 0;
}

// Convert ARGB to I420.
LIBYUV_API
int ARGBToI420(const uint8_t *src_argb,
               int src_stride_argb,
               uint8_t *dst_y,
               int dst_stride_y,
               uint8_t *dst_u,
               int dst_stride_u,
               uint8_t *dst_v,
               int dst_stride_v,
               int width,
               int height) {
    int y;
    void (*ARGBToUVRow)(const uint8_t *src_argb0, int src_stride_argb,
                        uint8_t *dst_u, uint8_t *dst_v, int width) =
    ARGBToUVRow_C;
    void (*ARGBToYRow)(const uint8_t *src_argb, uint8_t *dst_y, int width) =
    ARGBToYRow_C;
    if (!src_argb || !dst_y || !dst_u || !dst_v || width <= 0 || height == 0) {
        return -1;
    }
    // Negative height means invert the image.
    if (height < 0) {
        height = -height;
        src_argb = src_argb + (height - 1) * src_stride_argb;
        src_stride_argb = -src_stride_argb;
    }
#if defined(HAS_ARGBTOYROW_SSSE3)
    if (TestCpuFlag(kCpuHasSSSE3)) {
        ARGBToYRow = ARGBToYRow_Any_SSSE3;
        if (IS_ALIGNED(width, 16)) {
            ARGBToYRow = ARGBToYRow_SSSE3;
        }
    }
#endif
#if defined(HAS_ARGBTOUVROW_SSSE3)
    if (TestCpuFlag(kCpuHasSSSE3)) {
        ARGBToUVRow = ARGBToUVRow_Any_SSSE3;
        if (IS_ALIGNED(width, 16)) {
            ARGBToUVRow = ARGBToUVRow_SSSE3;
        }
    }
#endif
#if defined(HAS_ARGBTOYROW_AVX2)
    if (TestCpuFlag(kCpuHasAVX2)) {
        ARGBToYRow = ARGBToYRow_Any_AVX2;
        if (IS_ALIGNED(width, 32)) {
            ARGBToYRow = ARGBToYRow_AVX2;
        }
    }
#endif
#if defined(HAS_ARGBTOUVROW_AVX2)
    if (TestCpuFlag(kCpuHasAVX2)) {
        ARGBToUVRow = ARGBToUVRow_Any_AVX2;
        if (IS_ALIGNED(width, 32)) {
            ARGBToUVRow = ARGBToUVRow_AVX2;
        }
    }
#endif

    for (y = 0; y < height - 1; y += 2) {
        ARGBToUVRow(src_argb, src_stride_argb, dst_u, dst_v, width);
        ARGBToYRow(src_argb, dst_y, width);
        ARGBToYRow(src_argb + src_stride_argb, dst_y + dst_stride_y, width);
        src_argb += src_stride_argb * 2;
        dst_y += dst_stride_y * 2;
        dst_u += dst_stride_u;
        dst_v += dst_stride_v;
    }
    if (height & 1) {
        ARGBToUVRow(src_argb, 0, dst_u, dst_v, width);
        ARGBToYRow(src_argb, dst_y, width);
    }
    return 0;
}

// Convert ABGR to I420.
LIBYUV_API
int ABGRToI420(const uint8_t *src_abgr,
               int src_stride_abgr,
               uint8_t *dst_y,
               int dst_stride_y,
               uint8_t *dst_u,
               int dst_stride_u,
               uint8_t *dst_v,
               int dst_stride_v,
               int width,
               int height) {
    int y;
    void (*ABGRToUVRow)(const uint8_t *src_abgr0, int src_stride_abgr,
                        uint8_t *dst_u, uint8_t *dst_v, int width) =
    ABGRToUVRow_C;
    void (*ABGRToYRow)(const uint8_t *src_abgr, uint8_t *dst_y, int width) =
    ABGRToYRow_C;
    if (!src_abgr || !dst_y || !dst_u || !dst_v || width <= 0 || height == 0) {
        return -1;
    }
    // Negative height means invert the image.
    if (height < 0) {
        height = -height;
        src_abgr = src_abgr + (height - 1) * src_stride_abgr;
        src_stride_abgr = -src_stride_abgr;
    }
#if defined(HAS_ABGRTOYROW_SSSE3)
    if (TestCpuFlag(kCpuHasSSSE3)) {
        ABGRToYRow = ABGRToYRow_Any_SSSE3;
        if (IS_ALIGNED(width, 16)) {
            ABGRToYRow = ABGRToYRow_SSSE3;
        }
    }
#endif
#if defined(HAS_ABGRTOUVROW_SSSE3)
    if (TestCpuFlag(kCpuHasSSSE3)) {
        ABGRToUVRow = ABGRToUVRow_Any_SSSE3;
        if (IS_ALIGNED(width, 16)) {
            ABGRToUVRow = ABGRToUVRow_SSSE3;
        }
    }
#endif
#if defined(HAS_ABGRTOYROW_AVX2)
    if (TestCpuFlag(kCpuHasAVX2)) {
        ABGRToYRow = ABGRToYRow_Any_AVX2;
        if (IS_ALIGNED(width, 32)) {
            ABGRToYRow = ABGRToYRow_AVX2;
        }
    }
#endif
#if defined(HAS_ABGRTOUVROW_AVX2)
    if (TestCpuFlag(kCpuHasAVX2)) {
        ABGRToUVRow = ABGRToUVRow_Any_AVX2;
        if (IS_ALIGNED(width, 32)) {
            ABGRToUVRow = ABGRToUVRow_AVX2;
        }
    }
#endif

    for (y = 0; y < height - 1; y += 2) {
        ABGRToUVRow(src_abgr, src_stride_abgr, dst_u, dst_v, width);
        ABGRToYRow(src_abgr, dst_y, width);
        ABGRToYRow(src_abgr + src_stride_abgr, dst_y + dst_stride_y, width);
        src_abgr += src_stride_abgr * 2;
        dst_y += dst_stride_y * 2;
        dst_u += dst_stride_u;
        dst_v += dst_stride_v;
    }
    if (height & 1) {
        ABGRToUVRow(src_abgr, 0, dst_u, dst_v, width);
        ABGRToYRow(src_abgr, dst_y, width);
    }
    return 0;
}

// Convert RGB565 to I420.
LIBYUV_API
int RGB565ToI420(const uint8_t *src_rgb565,
                 int src_stride_rgb565,
                 uint8_t *dst_y,
                 int dst_stride_y,
                 uint8_t *dst_u,
                 int dst_stride_u,
                 uint8_t *dst_v,
                 int dst_stride_v,
                 int width,
                 int height) {
    int y;
    void (*RGB565ToARGBRow)(const uint8_t *src_rgb, uint8_t *dst_argb,
                            int width) = RGB565ToARGBRow_C;
    void (*ARGBToUVRow)(const uint8_t *src_argb0, int src_stride_argb,
                        uint8_t *dst_u, uint8_t *dst_v, int width) =
    ARGBToUVRow_C;
    void (*ARGBToYRow)(const uint8_t *src_argb, uint8_t *dst_y, int width) =
    ARGBToYRow_C;
    if (!src_rgb565 || !dst_y || !dst_u || !dst_v || width <= 0 || height == 0) {
        return -1;
    }
    // Negative height means invert the image.
    if (height < 0) {
        height = -height;
        src_rgb565 = src_rgb565 + (height - 1) * src_stride_rgb565;
        src_stride_rgb565 = -src_stride_rgb565;
    }

#if defined(HAS_RGB565TOARGBROW_SSE2)
    if (TestCpuFlag(kCpuHasSSE2)) {
        RGB565ToARGBRow = RGB565ToARGBRow_Any_SSE2;
        if (IS_ALIGNED(width, 8)) {
            RGB565ToARGBRow = RGB565ToARGBRow_SSE2;
        }
    }
#endif
#if defined(HAS_RGB565TOARGBROW_AVX2)
    if (TestCpuFlag(kCpuHasAVX2)) {
      RGB565ToARGBRow = RGB565ToARGBRow_Any_AVX2;
      if (IS_ALIGNED(width, 16)) {
        RGB565ToARGBRow = RGB565ToARGBRow_AVX2;
      }
    }
#endif
#if defined(HAS_ARGBTOYROW_SSSE3)
    if (TestCpuFlag(kCpuHasSSSE3)) {
        ARGBToYRow = ARGBToYRow_Any_SSSE3;
        if (IS_ALIGNED(width, 16)) {
            ARGBToYRow = ARGBToYRow_SSSE3;
        }
    }
#endif
#if defined(HAS_ARGBTOUVROW_SSSE3)
    if (TestCpuFlag(kCpuHasSSSE3)) {
        ARGBToUVRow = ARGBToUVRow_Any_SSSE3;
        if (IS_ALIGNED(width, 16)) {
            ARGBToUVRow = ARGBToUVRow_SSSE3;
        }
    }
#endif
#if defined(HAS_ARGBTOYROW_AVX2)
    if (TestCpuFlag(kCpuHasAVX2)) {
        ARGBToYRow = ARGBToYRow_Any_AVX2;
        if (IS_ALIGNED(width, 32)) {
            ARGBToYRow = ARGBToYRow_AVX2;
        }
    }
#endif
#if defined(HAS_ARGBTOUVROW_AVX2)
    if (TestCpuFlag(kCpuHasAVX2)) {
        ARGBToUVRow = ARGBToUVRow_Any_AVX2;
        if (IS_ALIGNED(width, 32)) {
            ARGBToUVRow = ARGBToUVRow_AVX2;
        }
    }
#endif
    {
#if !(defined(HAS_RGB565TOYROW_NEON))
        // Allocate 2 rows of ARGB.
        const int row_size = (width * 4 + 31) & ~31;
        align_buffer_64(row, row_size * 2);
#endif
        for (y = 0; y < height - 1; y += 2) {
#if (defined(HAS_RGB565TOYROW_NEON))
#else
            RGB565ToARGBRow(src_rgb565, row, width);
            RGB565ToARGBRow(src_rgb565 + src_stride_rgb565, row + row_size, width);
            ARGBToUVRow(row, row_size, dst_u, dst_v, width);
            ARGBToYRow(row, dst_y, width);
            ARGBToYRow(row + row_size, dst_y + dst_stride_y, width);
#endif
            src_rgb565 += src_stride_rgb565 * 2;
            dst_y += dst_stride_y * 2;
            dst_u += dst_stride_u;
            dst_v += dst_stride_v;
        }
        if (height & 1) {
#if (defined(HAS_RGB565TOYROW_NEON))
#else
            RGB565ToARGBRow(src_rgb565, row, width);
            ARGBToUVRow(row, 0, dst_u, dst_v, width);
            ARGBToYRow(row, dst_y, width);
#endif
        }
#if !(defined(HAS_RGB565TOYROW_NEON))
        free_aligned_buffer_64(row);
#endif
    }
    return 0;
}
