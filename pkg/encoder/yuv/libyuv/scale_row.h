/*
 *  Copyright 2013 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#ifndef INCLUDE_LIBYUV_SCALE_ROW_H_
#define INCLUDE_LIBYUV_SCALE_ROW_H_

#include "basic_types.h"
#include "scale.h"

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
// GCC >= 4.7.0 required for AVX2.
#if defined(__GNUC__) && (defined(__x86_64__) || defined(__i386__))
#if (__GNUC__ > 4) || (__GNUC__ == 4 && (__GNUC_MINOR__ >= 7))
#define GCC_HAS_AVX2 1
#endif  // GNUC >= 4.7
#endif  // __GNUC__

// The following are available on all x86 platforms:
#if !defined(LIBYUV_DISABLE_X86) && \
    (defined(_M_IX86) || defined(__x86_64__) || defined(__i386__))
#define HAS_FIXEDDIV1_X86
#define HAS_FIXEDDIV_X86
#define HAS_SCALEADDROW_SSE2
#define HAS_SCALECOLSUP2_SSE2
#define HAS_SCALEFILTERCOLS_SSSE3
#define HAS_SCALEROWDOWN2_SSSE3
#define HAS_SCALEROWDOWN34_SSSE3
#define HAS_SCALEROWDOWN38_SSSE3
#define HAS_SCALEROWDOWN4_SSSE3
#endif

// The following are available for gcc/clang x86 platforms:
// TODO(fbarchard): Port to Visual C
#if !defined(LIBYUV_DISABLE_X86) && (defined(__x86_64__) || defined(__i386__))
#define HAS_SCALEUVROWDOWN2BOX_SSSE3
#define HAS_SCALEROWUP2_LINEAR_SSE2
#define HAS_SCALEROWUP2_LINEAR_SSSE3
#define HAS_SCALEROWUP2_BILINEAR_SSE2
#define HAS_SCALEROWUP2_BILINEAR_SSSE3
#define HAS_SCALEROWUP2_LINEAR_12_SSSE3
#define HAS_SCALEROWUP2_BILINEAR_12_SSSE3
#define HAS_SCALEROWUP2_LINEAR_16_SSE2
#define HAS_SCALEROWUP2_BILINEAR_16_SSE2
#define HAS_SCALEUVROWUP2_LINEAR_SSSE3
#define HAS_SCALEUVROWUP2_BILINEAR_SSSE3
#define HAS_SCALEUVROWUP2_LINEAR_16_SSE41
#define HAS_SCALEUVROWUP2_BILINEAR_16_SSE41
#endif

// The following are available for gcc/clang x86 platforms, but
// require clang 3.4 or gcc 4.7.
// TODO(fbarchard): Port to Visual C
#if !defined(LIBYUV_DISABLE_X86) && \
    (defined(__x86_64__) || defined(__i386__)) && \
    (defined(CLANG_HAS_AVX2) || defined(GCC_HAS_AVX2))
#define HAS_SCALEUVROWDOWN2BOX_AVX2
#define HAS_SCALEROWUP2_LINEAR_AVX2
#define HAS_SCALEROWUP2_BILINEAR_AVX2
#define HAS_SCALEROWUP2_LINEAR_12_AVX2
#define HAS_SCALEROWUP2_BILINEAR_12_AVX2
#define HAS_SCALEROWUP2_LINEAR_16_AVX2
#define HAS_SCALEROWUP2_BILINEAR_16_AVX2
#define HAS_SCALEUVROWUP2_LINEAR_AVX2
#define HAS_SCALEUVROWUP2_BILINEAR_AVX2
#define HAS_SCALEUVROWUP2_LINEAR_16_AVX2
#define HAS_SCALEUVROWUP2_BILINEAR_16_AVX2
#endif

// The following are available on all x86 platforms, but
// require VS2012, clang 3.4 or gcc 4.7.
// The code supports NaCL but requires a new compiler and validator.
#if !defined(LIBYUV_DISABLE_X86) && \
    (defined(VISUALC_HAS_AVX2) || defined(CLANG_HAS_AVX2) || \
     defined(GCC_HAS_AVX2))
#define HAS_SCALEADDROW_AVX2
#define HAS_SCALEROWDOWN2_AVX2
#define HAS_SCALEROWDOWN4_AVX2
#endif

// Scale ARGB vertically with bilinear interpolation.
void ScalePlaneVertical(int src_height,
                        int dst_width,
                        int dst_height,
                        int src_stride,
                        int dst_stride,
                        const uint8_t *src_argb,
                        uint8_t *dst_argb,
                        int x,
                        int y,
                        int dy,
                        int bpp,
                        enum FilterMode filtering);

// Simplify the filtering based on scale factors.
enum FilterMode ScaleFilterReduce(int src_width,
                                  int src_height,
                                  int dst_width,
                                  int dst_height,
                                  enum FilterMode filtering);

// Divide num by div and return as 16.16 fixed point result.
int FixedDiv_X86(int num, int div);

int FixedDiv1_X86(int num, int div);

#ifdef HAS_FIXEDDIV_X86
#define FixedDiv FixedDiv_X86
#define FixedDiv1 FixedDiv1_X86
#endif

// Compute slope values for stepping.
void ScaleSlope(int src_width,
                int src_height,
                int dst_width,
                int dst_height,
                enum FilterMode filtering,
                int *x,
                int *y,
                int *dx,
                int *dy);

void ScaleRowDown2_C(const uint8_t *src_ptr,
                     ptrdiff_t src_stride,
                     uint8_t *dst,
                     int dst_width);

void ScaleRowDown2Linear_C(const uint8_t *src_ptr,
                           ptrdiff_t src_stride,
                           uint8_t *dst,
                           int dst_width);

void ScaleRowDown2Box_C(const uint8_t *src_ptr,
                        ptrdiff_t src_stride,
                        uint8_t *dst,
                        int dst_width);

void ScaleRowDown2Box_Odd_C(const uint8_t *src_ptr,
                            ptrdiff_t src_stride,
                            uint8_t *dst,
                            int dst_width);

void ScaleRowDown4_C(const uint8_t *src_ptr,
                     ptrdiff_t src_stride,
                     uint8_t *dst,
                     int dst_width);

void ScaleRowDown4Box_C(const uint8_t *src_ptr,
                        ptrdiff_t src_stride,
                        uint8_t *dst,
                        int dst_width);

void ScaleRowDown34_C(const uint8_t *src_ptr,
                      ptrdiff_t src_stride,
                      uint8_t *dst,
                      int dst_width);

void ScaleRowDown34_0_Box_C(const uint8_t *src_ptr,
                            ptrdiff_t src_stride,
                            uint8_t *d,
                            int dst_width);

void ScaleRowDown34_1_Box_C(const uint8_t *src_ptr,
                            ptrdiff_t src_stride,
                            uint8_t *d,
                            int dst_width);

void ScaleRowUp2_Linear_C(const uint8_t *src_ptr,
                          uint8_t *dst_ptr,
                          int dst_width);

void ScaleRowUp2_Bilinear_C(const uint8_t *src_ptr,
                            ptrdiff_t src_stride,
                            uint8_t *dst_ptr,
                            ptrdiff_t dst_stride,
                            int dst_width);

void ScaleRowUp2_Linear_16_C(const uint16_t *src_ptr,
                             uint16_t *dst_ptr,
                             int dst_width);

void ScaleRowUp2_Bilinear_16_C(const uint16_t *src_ptr,
                               ptrdiff_t src_stride,
                               uint16_t *dst_ptr,
                               ptrdiff_t dst_stride,
                               int dst_width);

void ScaleRowUp2_Linear_Any_C(const uint8_t *src_ptr,
                              uint8_t *dst_ptr,
                              int dst_width);

void ScaleRowUp2_Bilinear_Any_C(const uint8_t *src_ptr,
                                ptrdiff_t src_stride,
                                uint8_t *dst_ptr,
                                ptrdiff_t dst_stride,
                                int dst_width);

void ScaleRowUp2_Linear_16_Any_C(const uint16_t *src_ptr,
                                 uint16_t *dst_ptr,
                                 int dst_width);

void ScaleRowUp2_Bilinear_16_Any_C(const uint16_t *src_ptr,
                                   ptrdiff_t src_stride,
                                   uint16_t *dst_ptr,
                                   ptrdiff_t dst_stride,
                                   int dst_width);

void ScaleCols_C(uint8_t *dst_ptr,
                 const uint8_t *src_ptr,
                 int dst_width,
                 int x,
                 int dx);

void ScaleColsUp2_C(uint8_t *dst_ptr,
                    const uint8_t *src_ptr,
                    int dst_width,
                    int,
                    int);

void ScaleFilterCols_C(uint8_t *dst_ptr,
                       const uint8_t *src_ptr,
                       int dst_width,
                       int x,
                       int dx);

void ScaleFilterCols64_C(uint8_t *dst_ptr,
                         const uint8_t *src_ptr,
                         int dst_width,
                         int x32,
                         int dx);

void ScaleRowDown38_C(const uint8_t *src_ptr,
                      ptrdiff_t src_stride,
                      uint8_t *dst,
                      int dst_width);

void ScaleRowDown38_3_Box_C(const uint8_t *src_ptr,
                            ptrdiff_t src_stride,
                            uint8_t *dst_ptr,
                            int dst_width);

void ScaleRowDown38_2_Box_C(const uint8_t *src_ptr,
                            ptrdiff_t src_stride,
                            uint8_t *dst_ptr,
                            int dst_width);

void ScaleAddRow_C(const uint8_t *src_ptr, uint16_t *dst_ptr, int src_width);

void ScaleUVRowDown2_C(const uint8_t *src_uv,
                       ptrdiff_t src_stride,
                       uint8_t *dst_uv,
                       int dst_width);

void ScaleUVRowDown2Linear_C(const uint8_t *src_uv,
                             ptrdiff_t src_stride,
                             uint8_t *dst_uv,
                             int dst_width);

void ScaleUVRowDown2Box_C(const uint8_t *src_uv,
                          ptrdiff_t src_stride,
                          uint8_t *dst_uv,
                          int dst_width);

void ScaleUVRowDownEven_C(const uint8_t *src_uv,
                          ptrdiff_t src_stride,
                          int src_stepx,
                          uint8_t *dst_uv,
                          int dst_width);

void ScaleUVRowUp2_Linear_C(const uint8_t *src_ptr,
                            uint8_t *dst_ptr,
                            int dst_width);

void ScaleUVRowUp2_Bilinear_C(const uint8_t *src_ptr,
                              ptrdiff_t src_stride,
                              uint8_t *dst_ptr,
                              ptrdiff_t dst_stride,
                              int dst_width);

void ScaleUVRowUp2_Linear_Any_C(const uint8_t *src_ptr,
                                uint8_t *dst_ptr,
                                int dst_width);

void ScaleUVRowUp2_Bilinear_Any_C(const uint8_t *src_ptr,
                                  ptrdiff_t src_stride,
                                  uint8_t *dst_ptr,
                                  ptrdiff_t dst_stride,
                                  int dst_width);

void ScaleUVRowUp2_Linear_16_C(const uint16_t *src_ptr,
                               uint16_t *dst_ptr,
                               int dst_width);

void ScaleUVRowUp2_Bilinear_16_C(const uint16_t *src_ptr,
                                 ptrdiff_t src_stride,
                                 uint16_t *dst_ptr,
                                 ptrdiff_t dst_stride,
                                 int dst_width);

void ScaleUVRowUp2_Linear_16_Any_C(const uint16_t *src_ptr,
                                   uint16_t *dst_ptr,
                                   int dst_width);

void ScaleUVRowUp2_Bilinear_16_Any_C(const uint16_t *src_ptr,
                                     ptrdiff_t src_stride,
                                     uint16_t *dst_ptr,
                                     ptrdiff_t dst_stride,
                                     int dst_width);

// Specialized scalers for x86.
void ScaleRowDown2_SSSE3(const uint8_t *src_ptr,
                         ptrdiff_t src_stride,
                         uint8_t *dst_ptr,
                         int dst_width);

void ScaleRowDown2Linear_SSSE3(const uint8_t *src_ptr,
                               ptrdiff_t src_stride,
                               uint8_t *dst_ptr,
                               int dst_width);

void ScaleRowDown2Box_SSSE3(const uint8_t *src_ptr,
                            ptrdiff_t src_stride,
                            uint8_t *dst_ptr,
                            int dst_width);

void ScaleRowDown2_AVX2(const uint8_t *src_ptr,
                        ptrdiff_t src_stride,
                        uint8_t *dst_ptr,
                        int dst_width);

void ScaleRowDown2Linear_AVX2(const uint8_t *src_ptr,
                              ptrdiff_t src_stride,
                              uint8_t *dst_ptr,
                              int dst_width);

void ScaleRowDown2Box_AVX2(const uint8_t *src_ptr,
                           ptrdiff_t src_stride,
                           uint8_t *dst_ptr,
                           int dst_width);

void ScaleRowDown4_SSSE3(const uint8_t *src_ptr,
                         ptrdiff_t src_stride,
                         uint8_t *dst_ptr,
                         int dst_width);

void ScaleRowDown4Box_SSSE3(const uint8_t *src_ptr,
                            ptrdiff_t src_stride,
                            uint8_t *dst_ptr,
                            int dst_width);

void ScaleRowDown4_AVX2(const uint8_t *src_ptr,
                        ptrdiff_t src_stride,
                        uint8_t *dst_ptr,
                        int dst_width);

void ScaleRowDown4Box_AVX2(const uint8_t *src_ptr,
                           ptrdiff_t src_stride,
                           uint8_t *dst_ptr,
                           int dst_width);

void ScaleRowDown34_SSSE3(const uint8_t *src_ptr,
                          ptrdiff_t src_stride,
                          uint8_t *dst_ptr,
                          int dst_width);

void ScaleRowDown34_1_Box_SSSE3(const uint8_t *src_ptr,
                                ptrdiff_t src_stride,
                                uint8_t *dst_ptr,
                                int dst_width);

void ScaleRowDown34_0_Box_SSSE3(const uint8_t *src_ptr,
                                ptrdiff_t src_stride,
                                uint8_t *dst_ptr,
                                int dst_width);

void ScaleRowDown38_SSSE3(const uint8_t *src_ptr,
                          ptrdiff_t src_stride,
                          uint8_t *dst_ptr,
                          int dst_width);

void ScaleRowDown38_3_Box_SSSE3(const uint8_t *src_ptr,
                                ptrdiff_t src_stride,
                                uint8_t *dst_ptr,
                                int dst_width);

void ScaleRowDown38_2_Box_SSSE3(const uint8_t *src_ptr,
                                ptrdiff_t src_stride,
                                uint8_t *dst_ptr,
                                int dst_width);

void ScaleRowUp2_Linear_SSE2(const uint8_t *src_ptr,
                             uint8_t *dst_ptr,
                             int dst_width);

void ScaleRowUp2_Bilinear_SSE2(const uint8_t *src_ptr,
                               ptrdiff_t src_stride,
                               uint8_t *dst_ptr,
                               ptrdiff_t dst_stride,
                               int dst_width);

void ScaleRowUp2_Linear_12_SSSE3(const uint16_t *src_ptr,
                                 uint16_t *dst_ptr,
                                 int dst_width);

void ScaleRowUp2_Bilinear_12_SSSE3(const uint16_t *src_ptr,
                                   ptrdiff_t src_stride,
                                   uint16_t *dst_ptr,
                                   ptrdiff_t dst_stride,
                                   int dst_width);

void ScaleRowUp2_Linear_16_SSE2(const uint16_t *src_ptr,
                                uint16_t *dst_ptr,
                                int dst_width);

void ScaleRowUp2_Bilinear_16_SSE2(const uint16_t *src_ptr,
                                  ptrdiff_t src_stride,
                                  uint16_t *dst_ptr,
                                  ptrdiff_t dst_stride,
                                  int dst_width);

void ScaleRowUp2_Linear_SSSE3(const uint8_t *src_ptr,
                              uint8_t *dst_ptr,
                              int dst_width);

void ScaleRowUp2_Bilinear_SSSE3(const uint8_t *src_ptr,
                                ptrdiff_t src_stride,
                                uint8_t *dst_ptr,
                                ptrdiff_t dst_stride,
                                int dst_width);

void ScaleRowUp2_Linear_AVX2(const uint8_t *src_ptr,
                             uint8_t *dst_ptr,
                             int dst_width);

void ScaleRowUp2_Bilinear_AVX2(const uint8_t *src_ptr,
                               ptrdiff_t src_stride,
                               uint8_t *dst_ptr,
                               ptrdiff_t dst_stride,
                               int dst_width);

void ScaleRowUp2_Linear_12_AVX2(const uint16_t *src_ptr,
                                uint16_t *dst_ptr,
                                int dst_width);

void ScaleRowUp2_Bilinear_12_AVX2(const uint16_t *src_ptr,
                                  ptrdiff_t src_stride,
                                  uint16_t *dst_ptr,
                                  ptrdiff_t dst_stride,
                                  int dst_width);

void ScaleRowUp2_Linear_16_AVX2(const uint16_t *src_ptr,
                                uint16_t *dst_ptr,
                                int dst_width);

void ScaleRowUp2_Bilinear_16_AVX2(const uint16_t *src_ptr,
                                  ptrdiff_t src_stride,
                                  uint16_t *dst_ptr,
                                  ptrdiff_t dst_stride,
                                  int dst_width);

void ScaleRowUp2_Linear_Any_SSE2(const uint8_t *src_ptr,
                                 uint8_t *dst_ptr,
                                 int dst_width);

void ScaleRowUp2_Bilinear_Any_SSE2(const uint8_t *src_ptr,
                                   ptrdiff_t src_stride,
                                   uint8_t *dst_ptr,
                                   ptrdiff_t dst_stride,
                                   int dst_width);

void ScaleRowUp2_Linear_12_Any_SSSE3(const uint16_t *src_ptr,
                                     uint16_t *dst_ptr,
                                     int dst_width);

void ScaleRowUp2_Bilinear_12_Any_SSSE3(const uint16_t *src_ptr,
                                       ptrdiff_t src_stride,
                                       uint16_t *dst_ptr,
                                       ptrdiff_t dst_stride,
                                       int dst_width);

void ScaleRowUp2_Linear_16_Any_SSE2(const uint16_t *src_ptr,
                                    uint16_t *dst_ptr,
                                    int dst_width);

void ScaleRowUp2_Bilinear_16_Any_SSE2(const uint16_t *src_ptr,
                                      ptrdiff_t src_stride,
                                      uint16_t *dst_ptr,
                                      ptrdiff_t dst_stride,
                                      int dst_width);

void ScaleRowUp2_Linear_Any_SSSE3(const uint8_t *src_ptr,
                                  uint8_t *dst_ptr,
                                  int dst_width);

void ScaleRowUp2_Bilinear_Any_SSSE3(const uint8_t *src_ptr,
                                    ptrdiff_t src_stride,
                                    uint8_t *dst_ptr,
                                    ptrdiff_t dst_stride,
                                    int dst_width);

void ScaleRowUp2_Linear_Any_AVX2(const uint8_t *src_ptr,
                                 uint8_t *dst_ptr,
                                 int dst_width);

void ScaleRowUp2_Bilinear_Any_AVX2(const uint8_t *src_ptr,
                                   ptrdiff_t src_stride,
                                   uint8_t *dst_ptr,
                                   ptrdiff_t dst_stride,
                                   int dst_width);

void ScaleRowUp2_Linear_12_Any_AVX2(const uint16_t *src_ptr,
                                    uint16_t *dst_ptr,
                                    int dst_width);

void ScaleRowUp2_Bilinear_12_Any_AVX2(const uint16_t *src_ptr,
                                      ptrdiff_t src_stride,
                                      uint16_t *dst_ptr,
                                      ptrdiff_t dst_stride,
                                      int dst_width);

void ScaleRowUp2_Linear_16_Any_AVX2(const uint16_t *src_ptr,
                                    uint16_t *dst_ptr,
                                    int dst_width);

void ScaleRowUp2_Bilinear_16_Any_AVX2(const uint16_t *src_ptr,
                                      ptrdiff_t src_stride,
                                      uint16_t *dst_ptr,
                                      ptrdiff_t dst_stride,
                                      int dst_width);

void ScaleRowDown2_Any_SSSE3(const uint8_t *src_ptr,
                             ptrdiff_t src_stride,
                             uint8_t *dst_ptr,
                             int dst_width);

void ScaleRowDown2Linear_Any_SSSE3(const uint8_t *src_ptr,
                                   ptrdiff_t src_stride,
                                   uint8_t *dst_ptr,
                                   int dst_width);

void ScaleRowDown2Box_Any_SSSE3(const uint8_t *src_ptr,
                                ptrdiff_t src_stride,
                                uint8_t *dst_ptr,
                                int dst_width);

void ScaleRowDown2Box_Odd_SSSE3(const uint8_t *src_ptr,
                                ptrdiff_t src_stride,
                                uint8_t *dst_ptr,
                                int dst_width);

void ScaleRowDown2_Any_AVX2(const uint8_t *src_ptr,
                            ptrdiff_t src_stride,
                            uint8_t *dst_ptr,
                            int dst_width);

void ScaleRowDown2Linear_Any_AVX2(const uint8_t *src_ptr,
                                  ptrdiff_t src_stride,
                                  uint8_t *dst_ptr,
                                  int dst_width);

void ScaleRowDown2Box_Any_AVX2(const uint8_t *src_ptr,
                               ptrdiff_t src_stride,
                               uint8_t *dst_ptr,
                               int dst_width);

void ScaleRowDown2Box_Odd_AVX2(const uint8_t *src_ptr,
                               ptrdiff_t src_stride,
                               uint8_t *dst_ptr,
                               int dst_width);

void ScaleRowDown4_Any_SSSE3(const uint8_t *src_ptr,
                             ptrdiff_t src_stride,
                             uint8_t *dst_ptr,
                             int dst_width);

void ScaleRowDown4Box_Any_SSSE3(const uint8_t *src_ptr,
                                ptrdiff_t src_stride,
                                uint8_t *dst_ptr,
                                int dst_width);

void ScaleRowDown4_Any_AVX2(const uint8_t *src_ptr,
                            ptrdiff_t src_stride,
                            uint8_t *dst_ptr,
                            int dst_width);

void ScaleRowDown4Box_Any_AVX2(const uint8_t *src_ptr,
                               ptrdiff_t src_stride,
                               uint8_t *dst_ptr,
                               int dst_width);

void ScaleRowDown34_Any_SSSE3(const uint8_t *src_ptr,
                              ptrdiff_t src_stride,
                              uint8_t *dst_ptr,
                              int dst_width);

void ScaleRowDown34_1_Box_Any_SSSE3(const uint8_t *src_ptr,
                                    ptrdiff_t src_stride,
                                    uint8_t *dst_ptr,
                                    int dst_width);

void ScaleRowDown34_0_Box_Any_SSSE3(const uint8_t *src_ptr,
                                    ptrdiff_t src_stride,
                                    uint8_t *dst_ptr,
                                    int dst_width);

void ScaleRowDown38_Any_SSSE3(const uint8_t *src_ptr,
                              ptrdiff_t src_stride,
                              uint8_t *dst_ptr,
                              int dst_width);

void ScaleRowDown38_3_Box_Any_SSSE3(const uint8_t *src_ptr,
                                    ptrdiff_t src_stride,
                                    uint8_t *dst_ptr,
                                    int dst_width);

void ScaleRowDown38_2_Box_Any_SSSE3(const uint8_t *src_ptr,
                                    ptrdiff_t src_stride,
                                    uint8_t *dst_ptr,
                                    int dst_width);

void ScaleAddRow_SSE2(const uint8_t *src_ptr, uint16_t *dst_ptr, int src_width);

void ScaleAddRow_AVX2(const uint8_t *src_ptr, uint16_t *dst_ptr, int src_width);

void ScaleAddRow_Any_SSE2(const uint8_t *src_ptr,
                          uint16_t *dst_ptr,
                          int src_width);

void ScaleAddRow_Any_AVX2(const uint8_t *src_ptr,
                          uint16_t *dst_ptr,
                          int src_width);

void ScaleFilterCols_SSSE3(uint8_t *dst_ptr,
                           const uint8_t *src_ptr,
                           int dst_width,
                           int x,
                           int dx);

void ScaleColsUp2_SSE2(uint8_t *dst_ptr,
                       const uint8_t *src_ptr,
                       int dst_width,
                       int x,
                       int dx);

// UV Row functions
void ScaleUVRowDown2Box_SSSE3(const uint8_t *src_ptr,
                              ptrdiff_t src_stride,
                              uint8_t *dst_uv,
                              int dst_width);

void ScaleUVRowDown2Box_AVX2(const uint8_t *src_ptr,
                             ptrdiff_t src_stride,
                             uint8_t *dst_uv,
                             int dst_width);

void ScaleUVRowDown2Box_Any_SSSE3(const uint8_t *src_ptr,
                                  ptrdiff_t src_stride,
                                  uint8_t *dst_ptr,
                                  int dst_width);

void ScaleUVRowDown2Box_Any_AVX2(const uint8_t *src_ptr,
                                 ptrdiff_t src_stride,
                                 uint8_t *dst_ptr,
                                 int dst_width);

void ScaleUVRowUp2_Linear_SSSE3(const uint8_t *src_ptr,
                                uint8_t *dst_ptr,
                                int dst_width);

void ScaleUVRowUp2_Bilinear_SSSE3(const uint8_t *src_ptr,
                                  ptrdiff_t src_stride,
                                  uint8_t *dst_ptr,
                                  ptrdiff_t dst_stride,
                                  int dst_width);

void ScaleUVRowUp2_Linear_Any_SSSE3(const uint8_t *src_ptr,
                                    uint8_t *dst_ptr,
                                    int dst_width);

void ScaleUVRowUp2_Bilinear_Any_SSSE3(const uint8_t *src_ptr,
                                      ptrdiff_t src_stride,
                                      uint8_t *dst_ptr,
                                      ptrdiff_t dst_stride,
                                      int dst_width);

void ScaleUVRowUp2_Linear_AVX2(const uint8_t *src_ptr,
                               uint8_t *dst_ptr,
                               int dst_width);

void ScaleUVRowUp2_Bilinear_AVX2(const uint8_t *src_ptr,
                                 ptrdiff_t src_stride,
                                 uint8_t *dst_ptr,
                                 ptrdiff_t dst_stride,
                                 int dst_width);

void ScaleUVRowUp2_Linear_Any_AVX2(const uint8_t *src_ptr,
                                   uint8_t *dst_ptr,
                                   int dst_width);

void ScaleUVRowUp2_Bilinear_Any_AVX2(const uint8_t *src_ptr,
                                     ptrdiff_t src_stride,
                                     uint8_t *dst_ptr,
                                     ptrdiff_t dst_stride,
                                     int dst_width);

void ScaleUVRowUp2_Linear_16_SSE41(const uint16_t *src_ptr,
                                   uint16_t *dst_ptr,
                                   int dst_width);

void ScaleUVRowUp2_Bilinear_16_SSE41(const uint16_t *src_ptr,
                                     ptrdiff_t src_stride,
                                     uint16_t *dst_ptr,
                                     ptrdiff_t dst_stride,
                                     int dst_width);

void ScaleUVRowUp2_Linear_16_Any_SSE41(const uint16_t *src_ptr,
                                       uint16_t *dst_ptr,
                                       int dst_width);

void ScaleUVRowUp2_Bilinear_16_Any_SSE41(const uint16_t *src_ptr,
                                         ptrdiff_t src_stride,
                                         uint16_t *dst_ptr,
                                         ptrdiff_t dst_stride,
                                         int dst_width);

void ScaleUVRowUp2_Linear_16_AVX2(const uint16_t *src_ptr,
                                  uint16_t *dst_ptr,
                                  int dst_width);

void ScaleUVRowUp2_Bilinear_16_AVX2(const uint16_t *src_ptr,
                                    ptrdiff_t src_stride,
                                    uint16_t *dst_ptr,
                                    ptrdiff_t dst_stride,
                                    int dst_width);

void ScaleUVRowUp2_Linear_16_Any_AVX2(const uint16_t *src_ptr,
                                      uint16_t *dst_ptr,
                                      int dst_width);

void ScaleUVRowUp2_Bilinear_16_Any_AVX2(const uint16_t *src_ptr,
                                        ptrdiff_t src_stride,
                                        uint16_t *dst_ptr,
                                        ptrdiff_t dst_stride,
                                        int dst_width);

#endif  // INCLUDE_LIBYUV_SCALE_ROW_H_