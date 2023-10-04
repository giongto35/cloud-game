/*
 *  Copyright 2011 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#ifndef INCLUDE_LIBYUV_ROW_H_
#define INCLUDE_LIBYUV_ROW_H_

#include <stddef.h>  // For NULL
#include <stdlib.h>  // For malloc

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

// GCC >= 4.7.0 required for AVX2.
#if defined(__GNUC__) && (defined(__x86_64__) || defined(__i386__))
#if (__GNUC__ > 4) || (__GNUC__ == 4 && (__GNUC_MINOR__ >= 7))
#define GCC_HAS_AVX2 1
#endif  // GNUC >= 4.7
#endif  // __GNUC__

// The following are available on all x86 platforms:
#if !defined(LIBYUV_DISABLE_X86) && \
    (defined(_M_IX86) || defined(__x86_64__) || defined(__i386__))
// Conversions:
#define HAS_ABGRTOYROW_SSSE3
#define HAS_ARGBTOYROW_SSSE3
#define HAS_BGRATOYROW_SSSE3
#define HAS_COPYROW_ERMS
#define HAS_COPYROW_SSE2
#define HAS_INTERPOLATEROW_SSSE3
#define HAS_MIRRORROW_SSSE3
#define HAS_MIRRORSPLITUVROW_SSSE3
#if !defined(LIBYUV_BIT_EXACT)
#define HAS_ABGRTOUVROW_SSSE3
#define HAS_ARGBTOUVROW_SSSE3
#endif

// Effects:
#define HAS_ARGBGRAYROW_SSSE3
#define HAS_ARGBMIRRORROW_SSE2

#endif

// The following are available on all x86 platforms, but
// require VS2012, clang 3.4 or gcc 4.7.
#if !defined(LIBYUV_DISABLE_X86) && \
    (defined(VISUALC_HAS_AVX2) || defined(CLANG_HAS_AVX2) || \
     defined(GCC_HAS_AVX2))
#define HAS_ARGBEXTRACTALPHAROW_AVX2
#define HAS_ARGBMIRRORROW_AVX2
#define HAS_ARGBTOYROW_AVX2
#define HAS_COPYROW_AVX
#define HAS_INTERPOLATEROW_AVX2
#define HAS_MIRRORROW_AVX2
#if !defined(LIBYUV_BIT_EXACT)
#define HAS_ARGBTOUVROW_AVX2
#endif

#endif

// The following are available for gcc/clang x86 platforms:
// TODO(fbarchard): Port to Visual C
#if !defined(LIBYUV_DISABLE_X86) && (defined(__x86_64__) || defined(__i386__))
#define HAS_MIRRORUVROW_SSSE3

#endif

// The following are available for AVX2 gcc/clang x86 platforms:
// TODO(fbarchard): Port to Visual C
#if !defined(LIBYUV_DISABLE_X86) && \
    (defined(__x86_64__) || defined(__i386__)) && \
    (defined(CLANG_HAS_AVX2) || defined(GCC_HAS_AVX2))
#define HAS_ABGRTOYROW_AVX2
#define HAS_MIRRORUVROW_AVX2
#if !defined(LIBYUV_BIT_EXACT)
#define HAS_ABGRTOUVROW_AVX2
#endif

#endif

#if defined(_MSC_VER) && !defined(__CLR_VER) && !defined(__clang__)
                                                                                                                        #if defined(VISUALC_HAS_AVX2)
#define SIMD_ALIGNED(var) __declspec(align(32)) var
#else
#define SIMD_ALIGNED(var) __declspec(align(16)) var
#endif
#define LIBYUV_NOINLINE __declspec(noinline)
typedef __declspec(align(16)) int16_t vec16[8];
typedef __declspec(align(16)) int32_t vec32[4];
typedef __declspec(align(16)) float vecf32[4];
typedef __declspec(align(16)) int8_t vec8[16];
typedef __declspec(align(16)) uint16_t uvec16[8];
typedef __declspec(align(16)) uint32_t uvec32[4];
typedef __declspec(align(16)) uint8_t uvec8[16];
typedef __declspec(align(32)) int16_t lvec16[16];
typedef __declspec(align(32)) int32_t lvec32[8];
typedef __declspec(align(32)) int8_t lvec8[32];
typedef __declspec(align(32)) uint16_t ulvec16[16];
typedef __declspec(align(32)) uint32_t ulvec32[8];
typedef __declspec(align(32)) uint8_t ulvec8[32];
#elif !defined(__pnacl__) && (defined(__GNUC__) || defined(__clang__))
// Caveat GCC 4.2 to 4.7 have a known issue using vectors with const.
#if defined(CLANG_HAS_AVX2) || defined(GCC_HAS_AVX2)
#define SIMD_ALIGNED(var) var __attribute__((aligned(32)))
#else
#define SIMD_ALIGNED(var) var __attribute__((aligned(16)))
#endif
#define LIBYUV_NOINLINE __attribute__((noinline))
typedef int16_t __attribute__((vector_size(16))) vec16;
typedef int32_t __attribute__((vector_size(16))) vec32;
typedef float __attribute__((vector_size(16))) vecf32;
typedef int8_t __attribute__((vector_size(16))) vec8;
typedef uint16_t __attribute__((vector_size(16))) uvec16;
typedef uint32_t __attribute__((vector_size(16))) uvec32;
typedef uint8_t __attribute__((vector_size(16))) uvec8;
typedef int16_t __attribute__((vector_size(32))) lvec16;
typedef int32_t __attribute__((vector_size(32))) lvec32;
typedef int8_t __attribute__((vector_size(32))) lvec8;
typedef uint16_t __attribute__((vector_size(32))) ulvec16;
typedef uint32_t __attribute__((vector_size(32))) ulvec32;
typedef uint8_t __attribute__((vector_size(32))) ulvec8;
#else
#define SIMD_ALIGNED(var) var
#define LIBYUV_NOINLINE
typedef int16_t vec16[8];
typedef int32_t vec32[4];
typedef float vecf32[4];
typedef int8_t vec8[16];
typedef uint16_t uvec16[8];
typedef uint32_t uvec32[4];
typedef uint8_t uvec8[16];
typedef int16_t lvec16[16];
typedef int32_t lvec32[8];
typedef int8_t lvec8[32];
typedef uint16_t ulvec16[16];
typedef uint32_t ulvec32[8];
typedef uint8_t ulvec8[32];
#endif

#if !defined(__aarch64__) || !defined(__arm__)
// This struct is for Intel color conversion.
struct YuvConstants {
    uint8_t kUVToB[32];
    uint8_t kUVToG[32];
    uint8_t kUVToR[32];
    int16_t kYToRgb[16];
    int16_t kYBiasToRgb[16];
};
#endif

#define IS_ALIGNED(p, a) (!((uintptr_t)(p) & ((a)-1)))

#define align_buffer_64(var, size)                                         \
  void* var##_mem = malloc((size) + 63);                      /* NOLINT */ \
  uint8_t* var = (uint8_t*)(((intptr_t)var##_mem + 63) & ~63) /* NOLINT */

#define free_aligned_buffer_64(var) \
  free(var##_mem);                  \
  var = NULL

#if defined(__APPLE__) || defined(__x86_64__) || defined(__llvm__)
#define OMITFP
#else
#define OMITFP __attribute__((optimize("omit-frame-pointer")))
#endif

// NaCL macros for GCC x86 and x64.
#if defined(__native_client__)
#define LABELALIGN ".p2align 5\n"
#else
#define LABELALIGN
#endif

void ARGBToYRow_AVX2(const uint8_t *src_argb, uint8_t *dst_y, int width);

void ARGBToYRow_Any_AVX2(const uint8_t *src_ptr, uint8_t *dst_ptr, int width);

void ABGRToYRow_AVX2(const uint8_t *src_abgr, uint8_t *dst_y, int width);

void ABGRToYRow_Any_AVX2(const uint8_t *src_ptr, uint8_t *dst_ptr, int width);

void ARGBToYRow_SSSE3(const uint8_t *src_argb, uint8_t *dst_y, int width);

void ABGRToYRow_SSSE3(const uint8_t *src_abgr, uint8_t *dst_y, int width);

void BGRAToYRow_SSSE3(const uint8_t *src_bgra, uint8_t *dst_y, int width);

void ABGRToYRow_SSSE3(const uint8_t *src_abgr, uint8_t *dst_y, int width);

void ARGBToYRow_C(const uint8_t *src_rgb, uint8_t *dst_y, int width);

void ABGRToYRow_C(const uint8_t *src_rgb, uint8_t *dst_y, int width);

void RGB565ToYRow_C(const uint8_t *src_rgb565, uint8_t *dst_y, int width);

void ARGBToYRow_Any_SSSE3(const uint8_t *src_ptr, uint8_t *dst_ptr, int width);

void BGRAToYRow_Any_SSSE3(const uint8_t *src_ptr, uint8_t *dst_ptr, int width);

void ABGRToYRow_Any_SSSE3(const uint8_t *src_ptr, uint8_t *dst_ptr, int width);

void ARGBToUVRow_AVX2(const uint8_t *src_argb,
                      int src_stride_argb,
                      uint8_t *dst_u,
                      uint8_t *dst_v,
                      int width);

void ABGRToUVRow_AVX2(const uint8_t *src_abgr,
                      int src_stride_abgr,
                      uint8_t *dst_u,
                      uint8_t *dst_v,
                      int width);

void ARGBToUVRow_SSSE3(const uint8_t *src_argb,
                       int src_stride_argb,
                       uint8_t *dst_u,
                       uint8_t *dst_v,
                       int width);

void BGRAToUVRow_SSSE3(const uint8_t *src_bgra,
                       int src_stride_bgra,
                       uint8_t *dst_u,
                       uint8_t *dst_v,
                       int width);

void ABGRToUVRow_SSSE3(const uint8_t *src_abgr,
                       int src_stride_abgr,
                       uint8_t *dst_u,
                       uint8_t *dst_v,
                       int width);

void RGBAToUVRow_SSSE3(const uint8_t *src_rgba,
                       int src_stride_rgba,
                       uint8_t *dst_u,
                       uint8_t *dst_v,
                       int width);

void ARGBToUVRow_Any_AVX2(const uint8_t *src_ptr,
                          int src_stride,
                          uint8_t *dst_u,
                          uint8_t *dst_v,
                          int width);

void ABGRToUVRow_Any_AVX2(const uint8_t *src_ptr,
                          int src_stride,
                          uint8_t *dst_u,
                          uint8_t *dst_v,
                          int width);

void ARGBToUVRow_Any_SSSE3(const uint8_t *src_ptr,
                           int src_stride,
                           uint8_t *dst_u,
                           uint8_t *dst_v,
                           int width);

void BGRAToUVRow_Any_SSSE3(const uint8_t *src_ptr,
                           int src_stride,
                           uint8_t *dst_u,
                           uint8_t *dst_v,
                           int width);

void ABGRToUVRow_Any_SSSE3(const uint8_t *src_ptr,
                           int src_stride,
                           uint8_t *dst_u,
                           uint8_t *dst_v,
                           int width);

void RGBAToUVRow_Any_SSSE3(const uint8_t *src_ptr,
                           int src_stride,
                           uint8_t *dst_u,
                           uint8_t *dst_v,
                           int width);

void ARGBToUVRow_C(const uint8_t *src_rgb,
                   int src_stride_rgb,
                   uint8_t *dst_u,
                   uint8_t *dst_v,
                   int width);

void ARGBToUVRow_C(const uint8_t *src_rgb,
                   int src_stride_rgb,
                   uint8_t *dst_u,
                   uint8_t *dst_v,
                   int width);

void BGRAToUVRow_C(const uint8_t *src_rgb,
                   int src_stride_rgb,
                   uint8_t *dst_u,
                   uint8_t *dst_v,
                   int width);

void ABGRToUVRow_C(const uint8_t *src_rgb,
                   int src_stride_rgb,
                   uint8_t *dst_u,
                   uint8_t *dst_v,
                   int width);

void RGBAToUVRow_C(const uint8_t *src_rgb,
                   int src_stride_rgb,
                   uint8_t *dst_u,
                   uint8_t *dst_v,
                   int width);

void RGB565ToUVRow_C(const uint8_t *src_rgb565,
                     int src_stride_rgb565,
                     uint8_t *dst_u,
                     uint8_t *dst_v,
                     int width);

void MirrorRow_AVX2(const uint8_t *src, uint8_t *dst, int width);

void MirrorRow_SSSE3(const uint8_t *src, uint8_t *dst, int width);

void MirrorRow_C(const uint8_t *src, uint8_t *dst, int width);

void MirrorRow_Any_AVX2(const uint8_t *src_ptr, uint8_t *dst_ptr, int width);

void MirrorRow_Any_SSSE3(const uint8_t *src_ptr, uint8_t *dst_ptr, int width);

void MirrorRow_Any_SSE2(const uint8_t *src, uint8_t *dst, int width);

void MirrorUVRow_AVX2(const uint8_t *src_uv, uint8_t *dst_uv, int width);

void MirrorUVRow_SSSE3(const uint8_t *src_uv, uint8_t *dst_uv, int width);

void MirrorUVRow_Any_AVX2(const uint8_t *src_ptr, uint8_t *dst_ptr, int width);

void MirrorUVRow_Any_SSSE3(const uint8_t *src_ptr, uint8_t *dst_ptr, int width);

void ARGBMirrorRow_AVX2(const uint8_t *src, uint8_t *dst, int width);

void ARGBMirrorRow_SSE2(const uint8_t *src, uint8_t *dst, int width);

void ARGBMirrorRow_C(const uint8_t *src, uint8_t *dst, int width);

void ARGBMirrorRow_Any_AVX2(const uint8_t *src_ptr,
                            uint8_t *dst_ptr,
                            int width);

void ARGBMirrorRow_Any_SSE2(const uint8_t *src_ptr,
                            uint8_t *dst_ptr,
                            int width);

void CopyRow_SSE2(const uint8_t *src, uint8_t *dst, int width);

void CopyRow_AVX(const uint8_t *src, uint8_t *dst, int width);

void CopyRow_ERMS(const uint8_t *src, uint8_t *dst, int width);

void CopyRow_C(const uint8_t *src, uint8_t *dst, int count);

void CopyRow_Any_SSE2(const uint8_t *src_ptr, uint8_t *dst_ptr, int width);

void CopyRow_Any_AVX(const uint8_t *src_ptr, uint8_t *dst_ptr, int width);

void RGB565ToARGBRow_SSE2(const uint8_t *src, uint8_t *dst, int width);

void RGB565ToARGBRow_AVX2(const uint8_t *src_rgb565,
                          uint8_t *dst_argb,
                          int width);

void RGB565ToARGBRow_C(const uint8_t *src_rgb565, uint8_t *dst_argb, int width);

void RGB565ToARGBRow_Any_SSE2(const uint8_t *src_ptr,
                              uint8_t *dst_ptr,
                              int width);

void RGB565ToARGBRow_Any_AVX2(const uint8_t *src_ptr,
                              uint8_t *dst_ptr,
                              int width);

// Used for I420Scale, ARGBScale, and ARGBInterpolate.
void InterpolateRow_C(uint8_t *dst_ptr,
                      const uint8_t *src_ptr,
                      ptrdiff_t src_stride,
                      int width,
                      int source_y_fraction);

void InterpolateRow_SSSE3(uint8_t *dst_ptr,
                          const uint8_t *src_ptr,
                          ptrdiff_t src_stride,
                          int dst_width,
                          int source_y_fraction);

void InterpolateRow_AVX2(uint8_t *dst_ptr,
                         const uint8_t *src_ptr,
                         ptrdiff_t src_stride,
                         int dst_width,
                         int source_y_fraction);

void InterpolateRow_Any_SSSE3(uint8_t *dst_ptr,
                              const uint8_t *src_ptr,
                              ptrdiff_t src_stride_ptr,
                              int width,
                              int source_y_fraction);

void InterpolateRow_Any_AVX2(uint8_t *dst_ptr,
                             const uint8_t *src_ptr,
                             ptrdiff_t src_stride_ptr,
                             int width,
                             int source_y_fraction);

#endif  // INCLUDE_LIBYUV_ROW_H_