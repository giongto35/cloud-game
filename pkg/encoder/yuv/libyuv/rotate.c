/*
 *  Copyright 2011 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#include "rotate.h"

#include "convert.h"
#include "cpu_id.h"
#include "rotate_row.h"
#include "row.h"

LIBYUV_API
void TransposePlane(const uint8_t *src,
                    int src_stride,
                    uint8_t *dst,
                    int dst_stride,
                    int width,
                    int height) {
    int i = height;

    void (*TransposeWx8)(const uint8_t *src, int src_stride, uint8_t *dst,
                         int dst_stride, int width) = TransposeWx8_C;

#if defined(HAS_TRANSPOSEWX8_SSSE3)
    if (TestCpuFlag(kCpuHasSSSE3)) {
        TransposeWx8 = TransposeWx8_Any_SSSE3;
        if (IS_ALIGNED(width, 8)) {
            TransposeWx8 = TransposeWx8_SSSE3;
        }
    }
#endif
#if defined(HAS_TRANSPOSEWX8_FAST_SSSE3)
    if (TestCpuFlag(kCpuHasSSSE3)) {
        TransposeWx8 = TransposeWx8_Fast_Any_SSSE3;
        if (IS_ALIGNED(width, 16)) {
            TransposeWx8 = TransposeWx8_Fast_SSSE3;
        }
    }
#endif

    // Work across the source in 8x8 tiles
    while (i >= 8) {
        TransposeWx8(src, src_stride, dst, dst_stride, width);
        src += 8 * src_stride;  // Go down 8 rows.
        dst += 8;               // Move over 8 columns.
        i -= 8;
    }

    if (i > 0) {
        TransposeWxH_C(src, src_stride, dst, dst_stride, width, i);
    }
}

LIBYUV_API
void RotatePlane90(const uint8_t *src,
                   int src_stride,
                   uint8_t *dst,
                   int dst_stride,
                   int width,
                   int height) {
    // Rotate by 90 is a transpose with the source read
    // from bottom to top. So set the source pointer to the end
    // of the buffer and flip the sign of the source stride.
    src += src_stride * (height - 1);
    src_stride = -src_stride;
    TransposePlane(src, src_stride, dst, dst_stride, width, height);
}

LIBYUV_API
void RotatePlane270(const uint8_t *src,
                    int src_stride,
                    uint8_t *dst,
                    int dst_stride,
                    int width,
                    int height) {
    // Rotate by 270 is a transpose with the destination written
    // from bottom to top. So set the destination pointer to the end
    // of the buffer and flip the sign of the destination stride.
    dst += dst_stride * (width - 1);
    dst_stride = -dst_stride;
    TransposePlane(src, src_stride, dst, dst_stride, width, height);
}

LIBYUV_API
void RotatePlane180(const uint8_t *src,
                    int src_stride,
                    uint8_t *dst,
                    int dst_stride,
                    int width,
                    int height) {
    // Swap top and bottom row and mirror the content. Uses a temporary row.
    align_buffer_64(row, width);
    const uint8_t *src_bot = src + src_stride * (height - 1);
    uint8_t *dst_bot = dst + dst_stride * (height - 1);
    int half_height = (height + 1) >> 1;
    int y;
    void (*MirrorRow)(const uint8_t *src, uint8_t *dst, int width) = MirrorRow_C;
    void (*CopyRow)(const uint8_t *src, uint8_t *dst, int width) = CopyRow_C;
#if defined(HAS_MIRRORROW_SSSE3)
    if (TestCpuFlag(kCpuHasSSSE3)) {
        MirrorRow = MirrorRow_Any_SSSE3;
        if (IS_ALIGNED(width, 16)) {
            MirrorRow = MirrorRow_SSSE3;
        }
    }
#endif
#if defined(HAS_MIRRORROW_AVX2)
    if (TestCpuFlag(kCpuHasAVX2)) {
        MirrorRow = MirrorRow_Any_AVX2;
        if (IS_ALIGNED(width, 32)) {
            MirrorRow = MirrorRow_AVX2;
        }
    }
#endif
#if defined(HAS_COPYROW_SSE2)
    if (TestCpuFlag(kCpuHasSSE2)) {
        CopyRow = IS_ALIGNED(width, 32) ? CopyRow_SSE2 : CopyRow_Any_SSE2;
    }
#endif
#if defined(HAS_COPYROW_AVX)
    if (TestCpuFlag(kCpuHasAVX)) {
        CopyRow = IS_ALIGNED(width, 64) ? CopyRow_AVX : CopyRow_Any_AVX;
    }
#endif
#if defined(HAS_COPYROW_ERMS)
    if (TestCpuFlag(kCpuHasERMS)) {
        CopyRow = CopyRow_ERMS;
    }
#endif
#if defined(HAS_COPYROW_NEON)
#endif
    // Odd height will harmlessly mirror the middle row twice.
    for (y = 0; y < half_height; ++y) {
        CopyRow(src, row, width);        // Copy top row into buffer
        MirrorRow(src_bot, dst, width);  // Mirror bottom row into top row
        MirrorRow(row, dst_bot, width);  // Mirror buffer into bottom row
        src += src_stride;
        dst += dst_stride;
        src_bot -= src_stride;
        dst_bot -= dst_stride;
    }
    free_aligned_buffer_64(row);
}

LIBYUV_API
int I420Rotate(const uint8_t *src_y,
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
               int height,
               enum RotationMode mode) {
    int halfwidth = (width + 1) >> 1;
    int halfheight = (height + 1) >> 1;
    if ((!src_y && dst_y) || !src_u || !src_v || width <= 0 || height == 0 ||
        !dst_y || !dst_u || !dst_v) {
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

    switch (mode) {
        case kRotate0:
            // copy frame
            return I420Copy(src_y, src_stride_y, src_u, src_stride_u, src_v,
                            src_stride_v, dst_y, dst_stride_y, dst_u, dst_stride_u,
                            dst_v, dst_stride_v, width, height);
        case kRotate90:
            RotatePlane90(src_y, src_stride_y, dst_y, dst_stride_y, width, height);
            RotatePlane90(src_u, src_stride_u, dst_u, dst_stride_u, halfwidth,
                          halfheight);
            RotatePlane90(src_v, src_stride_v, dst_v, dst_stride_v, halfwidth,
                          halfheight);
            return 0;
        case kRotate270:
            RotatePlane270(src_y, src_stride_y, dst_y, dst_stride_y, width, height);
            RotatePlane270(src_u, src_stride_u, dst_u, dst_stride_u, halfwidth,
                           halfheight);
            RotatePlane270(src_v, src_stride_v, dst_v, dst_stride_v, halfwidth,
                           halfheight);
            return 0;
        case kRotate180:
            RotatePlane180(src_y, src_stride_y, dst_y, dst_stride_y, width, height);
            RotatePlane180(src_u, src_stride_u, dst_u, dst_stride_u, halfwidth,
                           halfheight);
            RotatePlane180(src_v, src_stride_v, dst_v, dst_stride_v, halfwidth,
                           halfheight);
            return 0;
        default:
            break;
    }
    return -1;
}
