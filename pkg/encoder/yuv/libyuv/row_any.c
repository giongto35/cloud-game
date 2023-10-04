/*
 *  Copyright 2012 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#include "row.h"

#include <string.h>  // For memset.

// Subsampled source needs to be increase by 1 of not even.
#define SS(width, shift) (((width) + (1 << (shift)) - 1) >> (shift))

// Any 1 to 1.
#define ANY11(NAMEANY, ANY_SIMD, UVSHIFT, SBPP, BPP, MASK)               \
  void NAMEANY(const uint8_t* src_ptr, uint8_t* dst_ptr, int width) {    \
    SIMD_ALIGNED(uint8_t vin[128]);                                      \
    SIMD_ALIGNED(uint8_t vout[128]);                                     \
    memset(vin, 0, sizeof(vin)); /* for YUY2 and msan */                 \
    int r = width & MASK;                                                \
    int n = width & ~MASK;                                               \
    if (n > 0) {                                                         \
      ANY_SIMD(src_ptr, dst_ptr, n);                                     \
    }                                                                    \
    memcpy(vin, src_ptr + (n >> UVSHIFT) * SBPP, SS(r, UVSHIFT) * SBPP); \
    ANY_SIMD(vin, vout, MASK + 1);                                       \
    memcpy(dst_ptr + n * BPP, vout, r * BPP);                            \
  }

#ifdef HAS_COPYROW_AVX

ANY11(CopyRow_Any_AVX, CopyRow_AVX, 0, 1, 1, 63)

#endif
#ifdef HAS_COPYROW_SSE2

ANY11(CopyRow_Any_SSE2, CopyRow_SSE2, 0, 1, 1, 31)

#endif

#ifdef HAS_ARGBTOYROW_AVX2

ANY11(ARGBToYRow_Any_AVX2, ARGBToYRow_AVX2, 0, 4, 1, 31)

#endif
#ifdef HAS_ABGRTOYROW_AVX2

ANY11(ABGRToYRow_Any_AVX2, ABGRToYRow_AVX2, 0, 4, 1, 31)

#endif
#ifdef HAS_ARGBTOYROW_SSSE3

ANY11(ARGBToYRow_Any_SSSE3, ARGBToYRow_SSSE3, 0, 4, 1, 15)

#endif
#ifdef HAS_BGRATOYROW_SSSE3

ANY11(BGRAToYRow_Any_SSSE3, BGRAToYRow_SSSE3, 0, 4, 1, 15)

ANY11(ABGRToYRow_Any_SSSE3, ABGRToYRow_SSSE3, 0, 4, 1, 15)

#endif

#undef ANY11

// Any 1 to 1 interpolate.  Takes 2 rows of source via stride.
#define ANY11I(NAMEANY, ANY_SIMD, TD, TS, SBPP, BPP, MASK)           \
  void NAMEANY(TD* dst_ptr, const TS* src_ptr, ptrdiff_t src_stride, \
               int width, int source_y_fraction) {                   \
    SIMD_ALIGNED(TS vin[64 * 2]);                                    \
    SIMD_ALIGNED(TD vout[64]);                                       \
    memset(vin, 0, sizeof(vin)); /* for msan */                      \
    int r = width & MASK;                                            \
    int n = width & ~MASK;                                           \
    if (n > 0) {                                                     \
      ANY_SIMD(dst_ptr, src_ptr, src_stride, n, source_y_fraction);  \
    }                                                                \
    memcpy(vin, src_ptr + n * SBPP, r * SBPP * sizeof(TS));          \
    if (source_y_fraction) {                                         \
      memcpy(vin + 64, src_ptr + src_stride + n * SBPP,              \
             r * SBPP * sizeof(TS));                                 \
    }                                                                \
    ANY_SIMD(vout, vin, 64, MASK + 1, source_y_fraction);            \
    memcpy(dst_ptr + n * BPP, vout, r * BPP * sizeof(TD));           \
  }

#ifdef HAS_INTERPOLATEROW_AVX2

ANY11I(InterpolateRow_Any_AVX2, InterpolateRow_AVX2, uint8_t, uint8_t, 1, 1, 31)

#endif
#ifdef HAS_INTERPOLATEROW_SSSE3

ANY11I(InterpolateRow_Any_SSSE3,
       InterpolateRow_SSSE3,
       uint8_t,
       uint8_t,
       1,
       1,
       15)

#endif

#undef ANY11I

// Any 1 to 1 mirror.
#define ANY11M(NAMEANY, ANY_SIMD, BPP, MASK)                          \
  void NAMEANY(const uint8_t* src_ptr, uint8_t* dst_ptr, int width) { \
    SIMD_ALIGNED(uint8_t vin[64]);                                    \
    SIMD_ALIGNED(uint8_t vout[64]);                                   \
    memset(vin, 0, sizeof(vin)); /* for msan */                       \
    int r = width & MASK;                                             \
    int n = width & ~MASK;                                            \
    if (n > 0) {                                                      \
      ANY_SIMD(src_ptr + r * BPP, dst_ptr, n);                        \
    }                                                                 \
    memcpy(vin, src_ptr, r* BPP);                                     \
    ANY_SIMD(vin, vout, MASK + 1);                                    \
    memcpy(dst_ptr + n * BPP, vout + (MASK + 1 - r) * BPP, r * BPP);  \
  }

#ifdef HAS_MIRRORROW_AVX2

ANY11M(MirrorRow_Any_AVX2, MirrorRow_AVX2, 1, 31)

#endif
#ifdef HAS_MIRRORROW_SSSE3

ANY11M(MirrorRow_Any_SSSE3, MirrorRow_SSSE3, 1, 15)

#endif
#ifdef HAS_MIRRORUVROW_AVX2

ANY11M(MirrorUVRow_Any_AVX2, MirrorUVRow_AVX2, 2, 15)

#endif
#ifdef HAS_MIRRORUVROW_SSSE3

ANY11M(MirrorUVRow_Any_SSSE3, MirrorUVRow_SSSE3, 2, 7)

#endif
#ifdef HAS_ARGBMIRRORROW_AVX2

ANY11M(ARGBMirrorRow_Any_AVX2, ARGBMirrorRow_AVX2, 4, 7)

#endif
#ifdef HAS_ARGBMIRRORROW_SSE2

ANY11M(ARGBMirrorRow_Any_SSE2, ARGBMirrorRow_SSE2, 4, 3)

#endif
#undef ANY11M

// Any 1 to 2 with source stride (2 rows of source).  Outputs UV planes.
// 128 byte row allows for 32 avx ARGB pixels.
#define ANY12S(NAMEANY, ANY_SIMD, UVSHIFT, BPP, MASK)                        \
  void NAMEANY(const uint8_t* src_ptr, int src_stride, uint8_t* dst_u,       \
               uint8_t* dst_v, int width) {                                  \
    SIMD_ALIGNED(uint8_t vin[128 * 2]);                                      \
    SIMD_ALIGNED(uint8_t vout[128 * 2]);                                     \
    memset(vin, 0, sizeof(vin)); /* for msan */                              \
    int r = width & MASK;                                                    \
    int n = width & ~MASK;                                                   \
    if (n > 0) {                                                             \
      ANY_SIMD(src_ptr, src_stride, dst_u, dst_v, n);                        \
    }                                                                        \
    memcpy(vin, src_ptr + (n >> UVSHIFT) * BPP, SS(r, UVSHIFT) * BPP);       \
    memcpy(vin + 128, src_ptr + src_stride + (n >> UVSHIFT) * BPP,           \
           SS(r, UVSHIFT) * BPP);                                            \
    if ((width & 1) && UVSHIFT == 0) { /* repeat last pixel for subsample */ \
      memcpy(vin + SS(r, UVSHIFT) * BPP, vin + SS(r, UVSHIFT) * BPP - BPP,   \
             BPP);                                                           \
      memcpy(vin + 128 + SS(r, UVSHIFT) * BPP,                               \
             vin + 128 + SS(r, UVSHIFT) * BPP - BPP, BPP);                   \
    }                                                                        \
    ANY_SIMD(vin, 128, vout, vout + 128, MASK + 1);                          \
    memcpy(dst_u + (n >> 1), vout, SS(r, 1));                                \
    memcpy(dst_v + (n >> 1), vout + 128, SS(r, 1));                          \
  }

#ifdef HAS_ARGBTOUVROW_AVX2

ANY12S(ARGBToUVRow_Any_AVX2, ARGBToUVRow_AVX2, 0, 4, 31)

#endif
#ifdef HAS_ABGRTOUVROW_AVX2

ANY12S(ABGRToUVRow_Any_AVX2, ABGRToUVRow_AVX2, 0, 4, 31)

#endif
#ifdef HAS_ARGBTOUVROW_SSSE3

ANY12S(ARGBToUVRow_Any_SSSE3, ARGBToUVRow_SSSE3, 0, 4, 15)

ANY12S(BGRAToUVRow_Any_SSSE3, BGRAToUVRow_SSSE3, 0, 4, 15)

ANY12S(ABGRToUVRow_Any_SSSE3, ABGRToUVRow_SSSE3, 0, 4, 15)

ANY12S(RGBAToUVRow_Any_SSSE3, RGBAToUVRow_SSSE3, 0, 4, 15)

#endif
#undef ANY12S
