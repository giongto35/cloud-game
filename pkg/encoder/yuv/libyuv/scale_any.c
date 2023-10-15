/*
 *  Copyright 2015 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#include "scale_row.h"

// Fixed scale down.
// Mask may be non-power of 2, so use MOD
#define SDANY(NAMEANY, SCALEROWDOWN_SIMD, SCALEROWDOWN_C, FACTOR, BPP, MASK)   \
  void NAMEANY(const uint8_t* src_ptr, ptrdiff_t src_stride, uint8_t* dst_ptr, \
               int dst_width) {                                                \
    int r = (int)((unsigned int)dst_width % (MASK + 1)); /* NOLINT */          \
    int n = dst_width - r;                                                     \
    if (n > 0) {                                                               \
      SCALEROWDOWN_SIMD(src_ptr, src_stride, dst_ptr, n);                      \
    }                                                                          \
    SCALEROWDOWN_C(src_ptr + (n * FACTOR) * BPP, src_stride,                   \
                   dst_ptr + n * BPP, r);                                      \
  }

// Fixed scale down for odd source width.  Used by I420Blend subsampling.
// Since dst_width is (width + 1) / 2, this function scales one less pixel
// and copies the last pixel.
#define SDODD(NAMEANY, SCALEROWDOWN_SIMD, SCALEROWDOWN_C, FACTOR, BPP, MASK)   \
  void NAMEANY(const uint8_t* src_ptr, ptrdiff_t src_stride, uint8_t* dst_ptr, \
               int dst_width) {                                                \
    int r = (int)((unsigned int)(dst_width - 1) % (MASK + 1)); /* NOLINT */    \
    int n = (dst_width - 1) - r;                                               \
    if (n > 0) {                                                               \
      SCALEROWDOWN_SIMD(src_ptr, src_stride, dst_ptr, n);                      \
    }                                                                          \
    SCALEROWDOWN_C(src_ptr + (n * FACTOR) * BPP, src_stride,                   \
                   dst_ptr + n * BPP, r + 1);                                  \
  }

#ifdef HAS_SCALEROWDOWN2_SSSE3

SDANY(ScaleRowDown2_Any_SSSE3, ScaleRowDown2_SSSE3, ScaleRowDown2_C, 2, 1, 15)

SDANY(ScaleRowDown2Linear_Any_SSSE3,
      ScaleRowDown2Linear_SSSE3,
      ScaleRowDown2Linear_C,
      2,
      1,
      15)

SDANY(ScaleRowDown2Box_Any_SSSE3,
      ScaleRowDown2Box_SSSE3,
      ScaleRowDown2Box_C,
      2,
      1,
      15)

SDODD(ScaleRowDown2Box_Odd_SSSE3,
      ScaleRowDown2Box_SSSE3,
      ScaleRowDown2Box_Odd_C,
      2,
      1,
      15)

#endif
#ifdef HAS_SCALEUVROWDOWN2BOX_SSSE3

SDANY(ScaleUVRowDown2Box_Any_SSSE3,
      ScaleUVRowDown2Box_SSSE3,
      ScaleUVRowDown2Box_C,
      2,
      2,
      3)

#endif
#ifdef HAS_SCALEUVROWDOWN2BOX_AVX2

SDANY(ScaleUVRowDown2Box_Any_AVX2,
      ScaleUVRowDown2Box_AVX2,
      ScaleUVRowDown2Box_C,
      2,
      2,
      7)

#endif
#ifdef HAS_SCALEROWDOWN2_AVX2

SDANY(ScaleRowDown2_Any_AVX2, ScaleRowDown2_AVX2, ScaleRowDown2_C, 2, 1, 31)

SDANY(ScaleRowDown2Linear_Any_AVX2,
      ScaleRowDown2Linear_AVX2,
      ScaleRowDown2Linear_C,
      2,
      1,
      31)

SDANY(ScaleRowDown2Box_Any_AVX2,
      ScaleRowDown2Box_AVX2,
      ScaleRowDown2Box_C,
      2,
      1,
      31)

SDODD(ScaleRowDown2Box_Odd_AVX2,
      ScaleRowDown2Box_AVX2,
      ScaleRowDown2Box_Odd_C,
      2,
      1,
      31)

#endif
#ifdef HAS_SCALEROWDOWN4_SSSE3

SDANY(ScaleRowDown4_Any_SSSE3, ScaleRowDown4_SSSE3, ScaleRowDown4_C, 4, 1, 7)

SDANY(ScaleRowDown4Box_Any_SSSE3,
      ScaleRowDown4Box_SSSE3,
      ScaleRowDown4Box_C,
      4,
      1,
      7)

#endif
#ifdef HAS_SCALEROWDOWN4_AVX2

SDANY(ScaleRowDown4_Any_AVX2, ScaleRowDown4_AVX2, ScaleRowDown4_C, 4, 1, 15)

SDANY(ScaleRowDown4Box_Any_AVX2,
      ScaleRowDown4Box_AVX2,
      ScaleRowDown4Box_C,
      4,
      1,
      15)

#endif
#ifdef HAS_SCALEROWDOWN34_SSSE3

SDANY(ScaleRowDown34_Any_SSSE3,
      ScaleRowDown34_SSSE3,
      ScaleRowDown34_C,
      4 / 3,
      1,
      23)

SDANY(ScaleRowDown34_0_Box_Any_SSSE3,
      ScaleRowDown34_0_Box_SSSE3,
      ScaleRowDown34_0_Box_C,
      4 / 3,
      1,
      23)

SDANY(ScaleRowDown34_1_Box_Any_SSSE3,
      ScaleRowDown34_1_Box_SSSE3,
      ScaleRowDown34_1_Box_C,
      4 / 3,
      1,
      23)

#endif

#ifdef HAS_SCALEROWDOWN38_SSSE3

SDANY(ScaleRowDown38_Any_SSSE3,
      ScaleRowDown38_SSSE3,
      ScaleRowDown38_C,
      8 / 3,
      1,
      11)

SDANY(ScaleRowDown38_3_Box_Any_SSSE3,
      ScaleRowDown38_3_Box_SSSE3,
      ScaleRowDown38_3_Box_C,
      8 / 3,
      1,
      5)

SDANY(ScaleRowDown38_2_Box_Any_SSSE3,
      ScaleRowDown38_2_Box_SSSE3,
      ScaleRowDown38_2_Box_C,
      8 / 3,
      1,
      5)

#endif


#undef SDANY

// Scale down by even scale factor.
#define SDAANY(NAMEANY, SCALEROWDOWN_SIMD, SCALEROWDOWN_C, BPP, MASK)       \
  void NAMEANY(const uint8_t* src_ptr, ptrdiff_t src_stride, int src_stepx, \
               uint8_t* dst_ptr, int dst_width) {                           \
    int r = dst_width & MASK;                                               \
    int n = dst_width & ~MASK;                                              \
    if (n > 0) {                                                            \
      SCALEROWDOWN_SIMD(src_ptr, src_stride, src_stepx, dst_ptr, n);        \
    }                                                                       \
    SCALEROWDOWN_C(src_ptr + (n * src_stepx) * BPP, src_stride, src_stepx,  \
                   dst_ptr + n * BPP, r);                                   \
  }



#ifdef SASIMDONLY
// This also works and uses memcpy and SIMD instead of C, but is slower on ARM

// Add rows box filter scale down.  Using macro from row_any
#define SAROW(NAMEANY, ANY_SIMD, SBPP, BPP, MASK)                      \
  void NAMEANY(const uint8_t* src_ptr, uint16_t* dst_ptr, int width) { \
    SIMD_ALIGNED(uint16_t dst_temp[32]);                               \
    SIMD_ALIGNED(uint8_t src_temp[32]);                                \
    memset(dst_temp, 0, 32 * 2); /* for msan */                        \
    int r = width & MASK;                                              \
    int n = width & ~MASK;                                             \
    if (n > 0) {                                                       \
      ANY_SIMD(src_ptr, dst_ptr, n);                                   \
    }                                                                  \
    memcpy(src_temp, src_ptr + n * SBPP, r * SBPP);                    \
    memcpy(dst_temp, dst_ptr + n * BPP, r * BPP);                      \
    ANY_SIMD(src_temp, dst_temp, MASK + 1);                            \
    memcpy(dst_ptr + n * BPP, dst_temp, r * BPP);                      \
  }

#ifdef HAS_SCALEADDROW_SSE2
SAROW(ScaleAddRow_Any_SSE2, ScaleAddRow_SSE2, 1, 2, 15)
#endif
#ifdef HAS_SCALEADDROW_AVX2
SAROW(ScaleAddRow_Any_AVX2, ScaleAddRow_AVX2, 1, 2, 31)
#endif
#undef SAANY

#else

// Add rows box filter scale down.
#define SAANY(NAMEANY, SCALEADDROW_SIMD, SCALEADDROW_C, MASK)              \
  void NAMEANY(const uint8_t* src_ptr, uint16_t* dst_ptr, int src_width) { \
    int n = src_width & ~MASK;                                             \
    if (n > 0) {                                                           \
      SCALEADDROW_SIMD(src_ptr, dst_ptr, n);                               \
    }                                                                      \
    SCALEADDROW_C(src_ptr + n, dst_ptr + n, src_width & MASK);             \
  }

#ifdef HAS_SCALEADDROW_SSE2

SAANY(ScaleAddRow_Any_SSE2, ScaleAddRow_SSE2, ScaleAddRow_C, 15)

#endif
#ifdef HAS_SCALEADDROW_AVX2

SAANY(ScaleAddRow_Any_AVX2, ScaleAddRow_AVX2, ScaleAddRow_C, 31)

#endif
#undef SAANY

#endif  // SASIMDONLY

// Scale up horizontally 2 times using linear filter.
#define SUH2LANY(NAME, SIMD, C, MASK, PTYPE)                       \
  void NAME(const PTYPE* src_ptr, PTYPE* dst_ptr, int dst_width) { \
    int work_width = (dst_width - 1) & ~1;                         \
    int r = work_width & MASK;                                     \
    int n = work_width & ~MASK;                                    \
    dst_ptr[0] = src_ptr[0];                                       \
    if (work_width > 0) {                                          \
      if (n != 0) {                                                \
        SIMD(src_ptr, dst_ptr + 1, n);                             \
      }                                                            \
      C(src_ptr + (n / 2), dst_ptr + n + 1, r);                    \
    }                                                              \
    dst_ptr[dst_width - 1] = src_ptr[(dst_width - 1) / 2];         \
  }

// Even the C versions need to be wrapped, because boundary pixels have to
// be handled differently

SUH2LANY(ScaleRowUp2_Linear_Any_C,
         ScaleRowUp2_Linear_C,
         ScaleRowUp2_Linear_C,
         0,
         uint8_t)

SUH2LANY(ScaleRowUp2_Linear_16_Any_C,
         ScaleRowUp2_Linear_16_C,
         ScaleRowUp2_Linear_16_C,
         0,
         uint16_t)

#ifdef HAS_SCALEROWUP2_LINEAR_SSE2

SUH2LANY(ScaleRowUp2_Linear_Any_SSE2,
         ScaleRowUp2_Linear_SSE2,
         ScaleRowUp2_Linear_C,
         15,
         uint8_t)

#endif

#ifdef HAS_SCALEROWUP2_LINEAR_SSSE3

SUH2LANY(ScaleRowUp2_Linear_Any_SSSE3,
         ScaleRowUp2_Linear_SSSE3,
         ScaleRowUp2_Linear_C,
         15,
         uint8_t)

#endif

#ifdef HAS_SCALEROWUP2_LINEAR_12_SSSE3

SUH2LANY(ScaleRowUp2_Linear_12_Any_SSSE3,
         ScaleRowUp2_Linear_12_SSSE3,
         ScaleRowUp2_Linear_16_C,
         15,
         uint16_t)

#endif

#ifdef HAS_SCALEROWUP2_LINEAR_16_SSE2

SUH2LANY(ScaleRowUp2_Linear_16_Any_SSE2,
         ScaleRowUp2_Linear_16_SSE2,
         ScaleRowUp2_Linear_16_C,
         7,
         uint16_t)

#endif

#ifdef HAS_SCALEROWUP2_LINEAR_AVX2

SUH2LANY(ScaleRowUp2_Linear_Any_AVX2,
         ScaleRowUp2_Linear_AVX2,
         ScaleRowUp2_Linear_C,
         31,
         uint8_t)

#endif

#ifdef HAS_SCALEROWUP2_LINEAR_12_AVX2

SUH2LANY(ScaleRowUp2_Linear_12_Any_AVX2,
         ScaleRowUp2_Linear_12_AVX2,
         ScaleRowUp2_Linear_16_C,
         31,
         uint16_t)

#endif

#ifdef HAS_SCALEROWUP2_LINEAR_16_AVX2

SUH2LANY(ScaleRowUp2_Linear_16_Any_AVX2,
         ScaleRowUp2_Linear_16_AVX2,
         ScaleRowUp2_Linear_16_C,
         15,
         uint16_t)

#endif
#undef SUH2LANY

// Scale up 2 times using bilinear filter.
// This function produces 2 rows at a time.
#define SU2BLANY(NAME, SIMD, C, MASK, PTYPE)                              \
  void NAME(const PTYPE* src_ptr, ptrdiff_t src_stride, PTYPE* dst_ptr,   \
            ptrdiff_t dst_stride, int dst_width) {                        \
    int work_width = (dst_width - 1) & ~1;                                \
    int r = work_width & MASK;                                            \
    int n = work_width & ~MASK;                                           \
    const PTYPE* sa = src_ptr;                                            \
    const PTYPE* sb = src_ptr + src_stride;                               \
    PTYPE* da = dst_ptr;                                                  \
    PTYPE* db = dst_ptr + dst_stride;                                     \
    da[0] = (3 * sa[0] + sb[0] + 2) >> 2;                                 \
    db[0] = (sa[0] + 3 * sb[0] + 2) >> 2;                                 \
    if (work_width > 0) {                                                 \
      if (n != 0) {                                                       \
        SIMD(sa, sb - sa, da + 1, db - da, n);                            \
      }                                                                   \
      C(sa + (n / 2), sb - sa, da + n + 1, db - da, r);                   \
    }                                                                     \
    da[dst_width - 1] =                                                   \
        (3 * sa[(dst_width - 1) / 2] + sb[(dst_width - 1) / 2] + 2) >> 2; \
    db[dst_width - 1] =                                                   \
        (sa[(dst_width - 1) / 2] + 3 * sb[(dst_width - 1) / 2] + 2) >> 2; \
  }

SU2BLANY(ScaleRowUp2_Bilinear_Any_C,
         ScaleRowUp2_Bilinear_C,
         ScaleRowUp2_Bilinear_C,
         0,
         uint8_t)

SU2BLANY(ScaleRowUp2_Bilinear_16_Any_C,
         ScaleRowUp2_Bilinear_16_C,
         ScaleRowUp2_Bilinear_16_C,
         0,
         uint16_t)

#ifdef HAS_SCALEROWUP2_BILINEAR_SSE2

SU2BLANY(ScaleRowUp2_Bilinear_Any_SSE2,
         ScaleRowUp2_Bilinear_SSE2,
         ScaleRowUp2_Bilinear_C,
         15,
         uint8_t)

#endif

#ifdef HAS_SCALEROWUP2_BILINEAR_12_SSSE3

SU2BLANY(ScaleRowUp2_Bilinear_12_Any_SSSE3,
         ScaleRowUp2_Bilinear_12_SSSE3,
         ScaleRowUp2_Bilinear_16_C,
         15,
         uint16_t)

#endif

#ifdef HAS_SCALEROWUP2_BILINEAR_16_SSE2

SU2BLANY(ScaleRowUp2_Bilinear_16_Any_SSE2,
         ScaleRowUp2_Bilinear_16_SSE2,
         ScaleRowUp2_Bilinear_16_C,
         7,
         uint16_t)

#endif

#ifdef HAS_SCALEROWUP2_BILINEAR_SSSE3

SU2BLANY(ScaleRowUp2_Bilinear_Any_SSSE3,
         ScaleRowUp2_Bilinear_SSSE3,
         ScaleRowUp2_Bilinear_C,
         15,
         uint8_t)

#endif

#ifdef HAS_SCALEROWUP2_BILINEAR_AVX2

SU2BLANY(ScaleRowUp2_Bilinear_Any_AVX2,
         ScaleRowUp2_Bilinear_AVX2,
         ScaleRowUp2_Bilinear_C,
         31,
         uint8_t)

#endif

#ifdef HAS_SCALEROWUP2_BILINEAR_12_AVX2

SU2BLANY(ScaleRowUp2_Bilinear_12_Any_AVX2,
         ScaleRowUp2_Bilinear_12_AVX2,
         ScaleRowUp2_Bilinear_16_C,
         15,
         uint16_t)

#endif

#ifdef HAS_SCALEROWUP2_BILINEAR_16_AVX2

SU2BLANY(ScaleRowUp2_Bilinear_16_Any_AVX2,
         ScaleRowUp2_Bilinear_16_AVX2,
         ScaleRowUp2_Bilinear_16_C,
         15,
         uint16_t)

#endif

#undef SU2BLANY

// Scale bi-planar plane up horizontally 2 times using linear filter.
#define SBUH2LANY(NAME, SIMD, C, MASK, PTYPE)                         \
  void NAME(const PTYPE* src_ptr, PTYPE* dst_ptr, int dst_width) {    \
    int work_width = (dst_width - 1) & ~1;                            \
    int r = work_width & MASK;                                        \
    int n = work_width & ~MASK;                                       \
    dst_ptr[0] = src_ptr[0];                                          \
    dst_ptr[1] = src_ptr[1];                                          \
    if (work_width > 0) {                                             \
      if (n != 0) {                                                   \
        SIMD(src_ptr, dst_ptr + 2, n);                                \
      }                                                               \
      C(src_ptr + n, dst_ptr + 2 * n + 2, r);                         \
    }                                                                 \
    dst_ptr[2 * dst_width - 2] = src_ptr[((dst_width + 1) & ~1) - 2]; \
    dst_ptr[2 * dst_width - 1] = src_ptr[((dst_width + 1) & ~1) - 1]; \
  }

SBUH2LANY(ScaleUVRowUp2_Linear_Any_C,
          ScaleUVRowUp2_Linear_C,
          ScaleUVRowUp2_Linear_C,
          0,
          uint8_t)

SBUH2LANY(ScaleUVRowUp2_Linear_16_Any_C,
          ScaleUVRowUp2_Linear_16_C,
          ScaleUVRowUp2_Linear_16_C,
          0,
          uint16_t)

#ifdef HAS_SCALEUVROWUP2_LINEAR_SSSE3

SBUH2LANY(ScaleUVRowUp2_Linear_Any_SSSE3,
          ScaleUVRowUp2_Linear_SSSE3,
          ScaleUVRowUp2_Linear_C,
          7,
          uint8_t)

#endif

#ifdef HAS_SCALEUVROWUP2_LINEAR_AVX2

SBUH2LANY(ScaleUVRowUp2_Linear_Any_AVX2,
          ScaleUVRowUp2_Linear_AVX2,
          ScaleUVRowUp2_Linear_C,
          15,
          uint8_t)

#endif

#ifdef HAS_SCALEUVROWUP2_LINEAR_16_SSE41

SBUH2LANY(ScaleUVRowUp2_Linear_16_Any_SSE41,
          ScaleUVRowUp2_Linear_16_SSE41,
          ScaleUVRowUp2_Linear_16_C,
          3,
          uint16_t)

#endif

#ifdef HAS_SCALEUVROWUP2_LINEAR_16_AVX2

SBUH2LANY(ScaleUVRowUp2_Linear_16_Any_AVX2,
          ScaleUVRowUp2_Linear_16_AVX2,
          ScaleUVRowUp2_Linear_16_C,
          7,
          uint16_t)

#endif

#undef SBUH2LANY

// Scale bi-planar plane up 2 times using bilinear filter.
// This function produces 2 rows at a time.
#define SBU2BLANY(NAME, SIMD, C, MASK, PTYPE)                           \
  void NAME(const PTYPE* src_ptr, ptrdiff_t src_stride, PTYPE* dst_ptr, \
            ptrdiff_t dst_stride, int dst_width) {                      \
    int work_width = (dst_width - 1) & ~1;                              \
    int r = work_width & MASK;                                          \
    int n = work_width & ~MASK;                                         \
    const PTYPE* sa = src_ptr;                                          \
    const PTYPE* sb = src_ptr + src_stride;                             \
    PTYPE* da = dst_ptr;                                                \
    PTYPE* db = dst_ptr + dst_stride;                                   \
    da[0] = (3 * sa[0] + sb[0] + 2) >> 2;                               \
    db[0] = (sa[0] + 3 * sb[0] + 2) >> 2;                               \
    da[1] = (3 * sa[1] + sb[1] + 2) >> 2;                               \
    db[1] = (sa[1] + 3 * sb[1] + 2) >> 2;                               \
    if (work_width > 0) {                                               \
      if (n != 0) {                                                     \
        SIMD(sa, sb - sa, da + 2, db - da, n);                          \
      }                                                                 \
      C(sa + n, sb - sa, da + 2 * n + 2, db - da, r);                   \
    }                                                                   \
    da[2 * dst_width - 2] = (3 * sa[((dst_width + 1) & ~1) - 2] +       \
                             sb[((dst_width + 1) & ~1) - 2] + 2) >>     \
                            2;                                          \
    db[2 * dst_width - 2] = (sa[((dst_width + 1) & ~1) - 2] +           \
                             3 * sb[((dst_width + 1) & ~1) - 2] + 2) >> \
                            2;                                          \
    da[2 * dst_width - 1] = (3 * sa[((dst_width + 1) & ~1) - 1] +       \
                             sb[((dst_width + 1) & ~1) - 1] + 2) >>     \
                            2;                                          \
    db[2 * dst_width - 1] = (sa[((dst_width + 1) & ~1) - 1] +           \
                             3 * sb[((dst_width + 1) & ~1) - 1] + 2) >> \
                            2;                                          \
  }

SBU2BLANY(ScaleUVRowUp2_Bilinear_Any_C,
          ScaleUVRowUp2_Bilinear_C,
          ScaleUVRowUp2_Bilinear_C,
          0,
          uint8_t)

SBU2BLANY(ScaleUVRowUp2_Bilinear_16_Any_C,
          ScaleUVRowUp2_Bilinear_16_C,
          ScaleUVRowUp2_Bilinear_16_C,
          0,
          uint16_t)

#ifdef HAS_SCALEUVROWUP2_BILINEAR_SSSE3

SBU2BLANY(ScaleUVRowUp2_Bilinear_Any_SSSE3,
          ScaleUVRowUp2_Bilinear_SSSE3,
          ScaleUVRowUp2_Bilinear_C,
          7,
          uint8_t)

#endif

#ifdef HAS_SCALEUVROWUP2_BILINEAR_AVX2

SBU2BLANY(ScaleUVRowUp2_Bilinear_Any_AVX2,
          ScaleUVRowUp2_Bilinear_AVX2,
          ScaleUVRowUp2_Bilinear_C,
          15,
          uint8_t)

#endif

#ifdef HAS_SCALEUVROWUP2_BILINEAR_16_SSE41

SBU2BLANY(ScaleUVRowUp2_Bilinear_16_Any_SSE41,
          ScaleUVRowUp2_Bilinear_16_SSE41,
          ScaleUVRowUp2_Bilinear_16_C,
          7,
          uint16_t)

#endif

#ifdef HAS_SCALEUVROWUP2_BILINEAR_16_AVX2

SBU2BLANY(ScaleUVRowUp2_Bilinear_16_Any_AVX2,
          ScaleUVRowUp2_Bilinear_16_AVX2,
          ScaleUVRowUp2_Bilinear_16_C,
          7,
          uint16_t)

#endif

#undef SBU2BLANY
