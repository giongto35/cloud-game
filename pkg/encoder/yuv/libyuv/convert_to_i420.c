/*
 *  Copyright 2011 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#include <stdlib.h>

#include "convert.h"
#include "video_common.h"

// Convert camera sample to I420 with cropping, rotation and vertical flip.
// src_width is used for source stride computation
// src_height is used to compute location of planes, and indicate inversion
// sample_size is measured in bytes and is the size of the frame.
//   With MJPEG it is the compressed size of the frame.
LIBYUV_API
int ConvertToI420(const uint8_t *sample,
                  size_t sample_size,
                  uint8_t *dst_y,
                  int dst_stride_y,
                  uint8_t *dst_u,
                  int dst_stride_u,
                  uint8_t *dst_v,
                  int dst_stride_v,
                  int crop_x,
                  int crop_y,
                  int src_width,
                  int src_height,
                  int crop_width,
                  int crop_height,
                  enum RotationMode rotation,
                  uint32_t fourcc) {
    uint32_t format = CanonicalFourCC(fourcc);
    const uint8_t *src;
    // TODO(nisse): Why allow crop_height < 0?
    const int abs_crop_height = (crop_height < 0) ? -crop_height : crop_height;
    int r = 0;
    LIBYUV_BOOL need_buf =
            (rotation && format != FOURCC_I420 && format != FOURCC_NV12 &&
             format != FOURCC_NV21 && format != FOURCC_YV12) ||
            dst_y == sample;
    uint8_t *tmp_y = dst_y;
    uint8_t *tmp_u = dst_u;
    uint8_t *tmp_v = dst_v;
    int tmp_y_stride = dst_stride_y;
    int tmp_u_stride = dst_stride_u;
    int tmp_v_stride = dst_stride_v;
    uint8_t *rotate_buffer = NULL;
    const int inv_crop_height =
            (src_height < 0) ? -abs_crop_height : abs_crop_height;

    if (!dst_y || !dst_u || !dst_v || !sample || src_width <= 0 ||
        crop_width <= 0 || src_height == 0 || crop_height == 0) {
        return -1;
    }

    // One pass rotation is available for some formats. For the rest, convert
    // to I420 (with optional vertical flipping) into a temporary I420 buffer,
    // and then rotate the I420 to the final destination buffer.
    // For in-place conversion, if destination dst_y is same as source sample,
    // also enable temporary buffer.
    if (need_buf) {
        int y_size = crop_width * abs_crop_height;
        int uv_size = ((crop_width + 1) / 2) * ((abs_crop_height + 1) / 2);
        rotate_buffer = (uint8_t *) malloc(y_size + uv_size * 2); /* NOLINT */
        if (!rotate_buffer) {
            return 1;  // Out of memory runtime error.
        }
        dst_y = rotate_buffer;
        dst_u = dst_y + y_size;
        dst_v = dst_u + uv_size;
        dst_stride_y = crop_width;
        dst_stride_u = dst_stride_v = ((crop_width + 1) / 2);
    }

    switch (format) {
        // Single plane formats
        case FOURCC_RGBP:
            src = sample + (src_width * crop_y + crop_x) * 2;
            r = RGB565ToI420(src, src_width * 2, dst_y, dst_stride_y, dst_u,
                             dst_stride_u, dst_v, dst_stride_v, crop_width,
                             inv_crop_height);
            break;
        case FOURCC_ARGB:
            src = sample + (src_width * crop_y + crop_x) * 4;
            r = ARGBToI420(src, src_width * 4, dst_y, dst_stride_y, dst_u,
                           dst_stride_u, dst_v, dst_stride_v, crop_width,
                           inv_crop_height);
            break;
        case FOURCC_ABGR:
            src = sample + (src_width * crop_y + crop_x) * 4;
            r = ABGRToI420(src, src_width * 4, dst_y, dst_stride_y, dst_u,
                           dst_stride_u, dst_v, dst_stride_v, crop_width,
                           inv_crop_height);
            break;
        default:
            r = -1;  // unknown fourcc - return failure code.
    }

    if (need_buf) {
        if (!r) {
            r = I420Rotate(dst_y, dst_stride_y, dst_u, dst_stride_u, dst_v,
                           dst_stride_v, tmp_y, tmp_y_stride, tmp_u, tmp_u_stride,
                           tmp_v, tmp_v_stride, crop_width, abs_crop_height,
                           rotation);
        }
        free(rotate_buffer);
    }

    return r;
}
