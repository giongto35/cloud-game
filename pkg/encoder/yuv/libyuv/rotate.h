/*
 *  Copyright 2011 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#ifndef INCLUDE_LIBYUV_ROTATE_H_
#define INCLUDE_LIBYUV_ROTATE_H_

#include "basic_types.h"

// Supported rotation.
typedef enum RotationMode {
    kRotate0 = 0,      // No rotation.
    kRotate90 = 90,    // Rotate 90 degrees clockwise.
    kRotate180 = 180,  // Rotate 180 degrees.
    kRotate270 = 270,  // Rotate 270 degrees clockwise.
} RotationModeEnum;

// Rotate I420 frame.
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
               enum RotationMode mode);

// Rotate planes by 90, 180, 270. Deprecated.
LIBYUV_API
void RotatePlane90(const uint8_t *src,
                   int src_stride,
                   uint8_t *dst,
                   int dst_stride,
                   int width,
                   int height);

LIBYUV_API
void RotatePlane180(const uint8_t *src,
                    int src_stride,
                    uint8_t *dst,
                    int dst_stride,
                    int width,
                    int height);

LIBYUV_API
void RotatePlane270(const uint8_t *src,
                    int src_stride,
                    uint8_t *dst,
                    int dst_stride,
                    int width,
                    int height);

// The 90 and 270 functions are based on transposes.
// Doing a transpose with reversing the read/write
// order will result in a rotation by +- 90 degrees.
// Deprecated.
LIBYUV_API
void TransposePlane(const uint8_t *src,
                    int src_stride,
                    uint8_t *dst,
                    int dst_stride,
                    int width,
                    int height);

#endif  // INCLUDE_LIBYUV_ROTATE_H_
