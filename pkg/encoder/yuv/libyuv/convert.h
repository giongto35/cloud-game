/*
 *  Copyright 2011 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#ifndef INCLUDE_LIBYUV_CONVERT_H_
#define INCLUDE_LIBYUV_CONVERT_H_

#include "rotate.h"  // For enum RotationMode.

// Copy I420 to I420.
#define I420ToI420 I420Copy
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
             int height);

// ARGB little endian (bgra in memory) to I420.
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
               int height);

// ABGR little endian (rgba in memory) to I420.
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
               int height);

// RGB16 (RGBP fourcc) little endian to I420.
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
                 int height);

// Convert camera sample to I420 with cropping, rotation and vertical flip.
// "src_size" is needed to parse MJPG.
// "dst_stride_y" number of bytes in a row of the dst_y plane.
//   Normally this would be the same as dst_width, with recommended alignment
//   to 16 bytes for better efficiency.
//   If rotation of 90 or 270 is used, stride is affected. The caller should
//   allocate the I420 buffer according to rotation.
// "dst_stride_u" number of bytes in a row of the dst_u plane.
//   Normally this would be the same as (dst_width + 1) / 2, with
//   recommended alignment to 16 bytes for better efficiency.
//   If rotation of 90 or 270 is used, stride is affected.
// "crop_x" and "crop_y" are starting position for cropping.
//   To center, crop_x = (src_width - dst_width) / 2
//              crop_y = (src_height - dst_height) / 2
// "src_width" / "src_height" is size of src_frame in pixels.
//   "src_height" can be negative indicating a vertically flipped image source.
// "crop_width" / "crop_height" is the size to crop the src to.
//    Must be less than or equal to src_width/src_height
//    Cropping parameters are pre-rotation.
// "rotation" can be 0, 90, 180 or 270.
// "fourcc" is a fourcc. ie 'I420', 'YUY2'
// Returns 0 for successful; -1 for invalid parameter. Non-zero for failure.
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
                  uint32_t fourcc);

#endif  // INCLUDE_LIBYUV_CONVERT_H_