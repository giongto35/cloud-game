/*
 *  Copyright 2013 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#ifndef INCLUDE_LIBYUV_ROTATE_ROW_H_
#define INCLUDE_LIBYUV_ROTATE_ROW_H_

#include "basic_types.h"

#if defined(__pnacl__) || defined(__CLR_VER) || \
    (defined(__native_client__) && defined(__x86_64__)) || \
    (defined(__i386__) && !defined(__SSE__) && !defined(__clang__))
#define LIBYUV_DISABLE_X86
#endif
#if defined(__native_client__)
#define LIBYUV_DISABLE_NEON
#endif
// MemorySanitizer does not support assembly code yet. http://crbug.com/344505
#if defined(__has_feature)
#if __has_feature(memory_sanitizer) && !defined(LIBYUV_DISABLE_NEON)
#define LIBYUV_DISABLE_NEON
#endif
#if __has_feature(memory_sanitizer) && !defined(LIBYUV_DISABLE_X86)
#define LIBYUV_DISABLE_X86
#endif
#endif

// The following are available for GCC 32 or 64 bit:
#if !defined(LIBYUV_DISABLE_X86) && (defined(__i386__) || defined(__x86_64__))
#define HAS_TRANSPOSEWX8_SSSE3
#endif

// The following are available for 64 bit GCC:
#if !defined(LIBYUV_DISABLE_X86) && defined(__x86_64__)
#define HAS_TRANSPOSEWX8_FAST_SSSE3
#define HAS_TRANSPOSEUVWX8_SSE2
#endif

void TransposeWxH_C(const uint8_t *src,
                    int src_stride,
                    uint8_t *dst,
                    int dst_stride,
                    int width,
                    int height);

void TransposeWx8_C(const uint8_t *src,
                    int src_stride,
                    uint8_t *dst,
                    int dst_stride,
                    int width);

void TransposeWx8_SSSE3(const uint8_t *src,
                        int src_stride,
                        uint8_t *dst,
                        int dst_stride,
                        int width);

void TransposeWx8_Fast_SSSE3(const uint8_t *src,
                             int src_stride,
                             uint8_t *dst,
                             int dst_stride,
                             int width);

void TransposeWx8_Any_SSSE3(const uint8_t *src,
                            int src_stride,
                            uint8_t *dst,
                            int dst_stride,
                            int width);

void TransposeWx8_Fast_Any_SSSE3(const uint8_t *src,
                                 int src_stride,
                                 uint8_t *dst,
                                 int dst_stride,
                                 int width);

void TransposeUVWx8_C(const uint8_t *src,
                      int src_stride,
                      uint8_t *dst_a,
                      int dst_stride_a,
                      uint8_t *dst_b,
                      int dst_stride_b,
                      int width);

void TransposeUVWx8_SSE2(const uint8_t *src,
                         int src_stride,
                         uint8_t *dst_a,
                         int dst_stride_a,
                         uint8_t *dst_b,
                         int dst_stride_b,
                         int width);

void TransposeUVWx8_Any_SSE2(const uint8_t *src,
                             int src_stride,
                             uint8_t *dst_a,
                             int dst_stride_a,
                             uint8_t *dst_b,
                             int dst_stride_b,
                             int width);

#endif  // INCLUDE_LIBYUV_ROTATE_ROW_H_
