/*
 *  Copyright 2011 The LibYuv Project Authors. All rights reserved.
 *
 *  Use of this source code is governed by a BSD-style license
 *  that can be found in the LICENSE file in the root of the source
 *  tree. An additional intellectual property rights grant can be found
 *  in the file PATENTS. All contributing project authors may
 *  be found in the AUTHORS file in the root of the source tree.
 */

#include "row.h"

#include <assert.h>
#include <string.h>  // For memcpy and memset.

#define STATIC_CAST(type, expr) (type)(expr)

// This macro controls YUV to RGB using unsigned math to extend range of
// YUV to RGB coefficients to 0 to 4 instead of 0 to 2 for more accuracy on B:
// LIBYUV_UNLIMITED_DATA

// Macros to enable unlimited data for each colorspace
// LIBYUV_UNLIMITED_BT601
// LIBYUV_UNLIMITED_BT709
// LIBYUV_UNLIMITED_BT2020

#if !defined(LIBYUV_BIT_EXACT) && (defined(__x86_64__) || defined(_M_X64) || \
                                   defined(__i386__) || defined(_M_IX86))
#define LIBYUV_ARGBTOUV_PAVGB 1
#define LIBYUV_RGBTOU_TRUNCATE 1
#endif
#if defined(LIBYUV_BIT_EXACT)
#define LIBYUV_UNATTENUATE_DUP 1
#endif

// llvm x86 is poor at ternary operator, so use branchless min/max.

#define USE_BRANCHLESS 1
#if USE_BRANCHLESS

static __inline int32_t clamp0(int32_t v) {
    return -(v >= 0) & v;
}

// TODO(fbarchard): make clamp255 preserve negative values.
static __inline int32_t clamp255(int32_t v) {
    return (-(v >= 255) | v) & 255;
}

static __inline int32_t clamp1023(int32_t v) {
    return (-(v >= 1023) | v) & 1023;
}

// clamp to max
static __inline int32_t ClampMax(int32_t v, int32_t max) {
    return (-(v >= max) | v) & max;
}

static __inline uint32_t Abs(int32_t v) {
    int m = -(v < 0);
    return (v + m) ^ m;
}

#else   // USE_BRANCHLESS
static __inline int32_t clamp0(int32_t v) {
  return (v < 0) ? 0 : v;
}

static __inline int32_t clamp255(int32_t v) {
  return (v > 255) ? 255 : v;
}

static __inline int32_t clamp1023(int32_t v) {
  return (v > 1023) ? 1023 : v;
}

static __inline int32_t ClampMax(int32_t v, int32_t max) {
  return (v > max) ? max : v;
}

static __inline uint32_t Abs(int32_t v) {
  return (v < 0) ? -v : v;
}
#endif  // USE_BRANCHLESS

static __inline uint32_t Clamp(int32_t val) {
    int v = clamp0(val);
    return (uint32_t) (clamp255(v));
}

static __inline uint32_t Clamp10(int32_t val) {
    int v = clamp0(val);
    return (uint32_t) (clamp1023(v));
}

// Little Endian
#if defined(__x86_64__) || defined(_M_X64) || defined(__i386__) || \
    defined(_M_IX86) || defined(__arm__) || defined(_M_ARM) || \
    (defined(__BYTE_ORDER__) && __BYTE_ORDER__ == __ORDER_LITTLE_ENDIAN__)
#define WRITEWORD(p, v) *(uint32_t*)(p) = v
#else
static inline void WRITEWORD(uint8_t* p, uint32_t v) {
  p[0] = (uint8_t)(v & 255);
  p[1] = (uint8_t)((v >> 8) & 255);
  p[2] = (uint8_t)((v >> 16) & 255);
  p[3] = (uint8_t)((v >> 24) & 255);
}
#endif

void RGB565ToARGBRow_C(const uint8_t *src_rgb565,
                       uint8_t *dst_argb,
                       int width) {
    int x;
    for (x = 0; x < width; ++x) {
        uint8_t b = STATIC_CAST(uint8_t, src_rgb565[0] & 0x1f);
        uint8_t g = STATIC_CAST(
                uint8_t, (src_rgb565[0] >> 5) | ((src_rgb565[1] & 0x07) << 3));
        uint8_t r = STATIC_CAST(uint8_t, src_rgb565[1] >> 3);
        dst_argb[0] = STATIC_CAST(uint8_t, (b << 3) | (b >> 2));
        dst_argb[1] = STATIC_CAST(uint8_t, (g << 2) | (g >> 4));
        dst_argb[2] = STATIC_CAST(uint8_t, (r << 3) | (r >> 2));
        dst_argb[3] = 255u;
        dst_argb += 4;
        src_rgb565 += 2;
    }
}

// 8 bit
// Intel SSE/AVX uses the following equivalent formula
// 0x7e80 = (66 + 129 + 25) * -128 + 0x1000 (for +16) and 0x0080 for round.
//  return (66 * ((int)r - 128) + 129 * ((int)g - 128) + 25 * ((int)b - 128) +
//  0x7e80) >> 8;

static __inline uint8_t RGBToY(uint8_t r, uint8_t g, uint8_t b) {
    return STATIC_CAST(uint8_t, (66 * r + 129 * g + 25 * b + 0x1080) >> 8);
}

#define AVGB(a, b) (((a) + (b) + 1) >> 1)

// LIBYUV_RGBTOU_TRUNCATE mimics x86 code that does not round.
#ifdef LIBYUV_RGBTOU_TRUNCATE

static __inline uint8_t RGBToU(uint8_t r, uint8_t g, uint8_t b) {
    return STATIC_CAST(uint8_t, (112 * b - 74 * g - 38 * r + 0x8000) >> 8);
}

static __inline uint8_t RGBToV(uint8_t r, uint8_t g, uint8_t b) {
    return STATIC_CAST(uint8_t, (112 * r - 94 * g - 18 * b + 0x8000) >> 8);
}

#else
// TODO(fbarchard): Add rounding to x86 SIMD and use this
static __inline uint8_t RGBToU(uint8_t r, uint8_t g, uint8_t b) {
  return STATIC_CAST(uint8_t, (112 * b - 74 * g - 38 * r + 0x8080) >> 8);
}
static __inline uint8_t RGBToV(uint8_t r, uint8_t g, uint8_t b) {
  return STATIC_CAST(uint8_t, (112 * r - 94 * g - 18 * b + 0x8080) >> 8);
}
#endif

// LIBYUV_ARGBTOUV_PAVGB mimics x86 code that subsamples with 2 pavgb.
#if !defined(LIBYUV_ARGBTOUV_PAVGB)
static __inline int RGB2xToU(uint16_t r, uint16_t g, uint16_t b) {
  return STATIC_CAST(
      uint8_t, ((112 / 2) * b - (74 / 2) * g - (38 / 2) * r + 0x8080) >> 8);
}
static __inline int RGB2xToV(uint16_t r, uint16_t g, uint16_t b) {
  return STATIC_CAST(
      uint8_t, ((112 / 2) * r - (94 / 2) * g - (18 / 2) * b + 0x8080) >> 8);
}
#endif

// ARGBToY_C and ARGBToUV_C
// Intel version mimic SSE/AVX which does 2 pavgb
#if LIBYUV_ARGBTOUV_PAVGB
#define MAKEROWY(NAME, R, G, B, BPP)                                       \
  void NAME##ToYRow_C(const uint8_t* src_rgb, uint8_t* dst_y, int width) { \
    int x;                                                                 \
    for (x = 0; x < width; ++x) {                                          \
      dst_y[0] = RGBToY(src_rgb[R], src_rgb[G], src_rgb[B]);               \
      src_rgb += BPP;                                                      \
      dst_y += 1;                                                          \
    }                                                                      \
  }                                                                        \
  void NAME##ToUVRow_C(const uint8_t* src_rgb, int src_stride_rgb,         \
                       uint8_t* dst_u, uint8_t* dst_v, int width) {        \
    const uint8_t* src_rgb1 = src_rgb + src_stride_rgb;                    \
    int x;                                                                 \
    for (x = 0; x < width - 1; x += 2) {                                   \
      uint8_t ab = AVGB(AVGB(src_rgb[B], src_rgb1[B]),                     \
                        AVGB(src_rgb[B + BPP], src_rgb1[B + BPP]));        \
      uint8_t ag = AVGB(AVGB(src_rgb[G], src_rgb1[G]),                     \
                        AVGB(src_rgb[G + BPP], src_rgb1[G + BPP]));        \
      uint8_t ar = AVGB(AVGB(src_rgb[R], src_rgb1[R]),                     \
                        AVGB(src_rgb[R + BPP], src_rgb1[R + BPP]));        \
      dst_u[0] = RGBToU(ar, ag, ab);                                       \
      dst_v[0] = RGBToV(ar, ag, ab);                                       \
      src_rgb += BPP * 2;                                                  \
      src_rgb1 += BPP * 2;                                                 \
      dst_u += 1;                                                          \
      dst_v += 1;                                                          \
    }                                                                      \
    if (width & 1) {                                                       \
      uint8_t ab = AVGB(src_rgb[B], src_rgb1[B]);                          \
      uint8_t ag = AVGB(src_rgb[G], src_rgb1[G]);                          \
      uint8_t ar = AVGB(src_rgb[R], src_rgb1[R]);                          \
      dst_u[0] = RGBToU(ar, ag, ab);                                       \
      dst_v[0] = RGBToV(ar, ag, ab);                                       \
    }                                                                      \
  }
#else
// ARM version does sum / 2 then multiply by 2x smaller coefficients
#define MAKEROWY(NAME, R, G, B, BPP)                                       \
  void NAME##ToYRow_C(const uint8_t* src_rgb, uint8_t* dst_y, int width) { \
    int x;                                                                 \
    for (x = 0; x < width; ++x) {                                          \
      dst_y[0] = RGBToY(src_rgb[R], src_rgb[G], src_rgb[B]);               \
      src_rgb += BPP;                                                      \
      dst_y += 1;                                                          \
    }                                                                      \
  }                                                                        \
  void NAME##ToUVRow_C(const uint8_t* src_rgb, int src_stride_rgb,         \
                       uint8_t* dst_u, uint8_t* dst_v, int width) {        \
    const uint8_t* src_rgb1 = src_rgb + src_stride_rgb;                    \
    int x;                                                                 \
    for (x = 0; x < width - 1; x += 2) {                                   \
      uint16_t ab = (src_rgb[B] + src_rgb[B + BPP] + src_rgb1[B] +         \
                     src_rgb1[B + BPP] + 1) >>                             \
                    1;                                                     \
      uint16_t ag = (src_rgb[G] + src_rgb[G + BPP] + src_rgb1[G] +         \
                     src_rgb1[G + BPP] + 1) >>                             \
                    1;                                                     \
      uint16_t ar = (src_rgb[R] + src_rgb[R + BPP] + src_rgb1[R] +         \
                     src_rgb1[R + BPP] + 1) >>                             \
                    1;                                                     \
      dst_u[0] = RGB2xToU(ar, ag, ab);                                     \
      dst_v[0] = RGB2xToV(ar, ag, ab);                                     \
      src_rgb += BPP * 2;                                                  \
      src_rgb1 += BPP * 2;                                                 \
      dst_u += 1;                                                          \
      dst_v += 1;                                                          \
    }                                                                      \
    if (width & 1) {                                                       \
      uint16_t ab = src_rgb[B] + src_rgb1[B];                              \
      uint16_t ag = src_rgb[G] + src_rgb1[G];                              \
      uint16_t ar = src_rgb[R] + src_rgb1[R];                              \
      dst_u[0] = RGB2xToU(ar, ag, ab);                                     \
      dst_v[0] = RGB2xToV(ar, ag, ab);                                     \
    }                                                                      \
  }
#endif

MAKEROWY(ARGB, 2, 1, 0, 4)

MAKEROWY(BGRA, 1, 2, 3, 4)

MAKEROWY(ABGR, 0, 1, 2, 4)

MAKEROWY(RGBA, 3, 2, 1, 4)

#undef MAKEROWY

// JPeg uses a variation on BT.601-1 full range
// y =  0.29900 * r + 0.58700 * g + 0.11400 * b
// u = -0.16874 * r - 0.33126 * g + 0.50000 * b  + center
// v =  0.50000 * r - 0.41869 * g - 0.08131 * b  + center
// BT.601 Mpeg range uses:
// b 0.1016 * 255 = 25.908 = 25
// g 0.5078 * 255 = 129.489 = 129
// r 0.2578 * 255 = 65.739 = 66
// JPeg 7 bit Y (deprecated)
// b 0.11400 * 128 = 14.592 = 15
// g 0.58700 * 128 = 75.136 = 75
// r 0.29900 * 128 = 38.272 = 38
// JPeg 8 bit Y:
// b 0.11400 * 256 = 29.184 = 29
// g 0.58700 * 256 = 150.272 = 150
// r 0.29900 * 256 = 76.544 = 77
// JPeg 8 bit U:
// b  0.50000 * 255 = 127.5 = 127
// g -0.33126 * 255 = -84.4713 = -84
// r -0.16874 * 255 = -43.0287 = -43
// JPeg 8 bit V:
// b -0.08131 * 255 = -20.73405 = -20
// g -0.41869 * 255 = -106.76595 = -107
// r  0.50000 * 255 = 127.5 = 127

// 8 bit
static __inline uint8_t RGBToYJ(uint8_t r, uint8_t g, uint8_t b) {
    return (77 * r + 150 * g + 29 * b + 128) >> 8;
}

#if defined(LIBYUV_ARGBTOUV_PAVGB)

static __inline uint8_t RGBToUJ(uint8_t r, uint8_t g, uint8_t b) {
    return (127 * b - 84 * g - 43 * r + 0x8080) >> 8;
}

static __inline uint8_t RGBToVJ(uint8_t r, uint8_t g, uint8_t b) {
    return (127 * r - 107 * g - 20 * b + 0x8080) >> 8;
}

#else
static __inline uint8_t RGB2xToUJ(uint16_t r, uint16_t g, uint16_t b) {
  return ((127 / 2) * b - (84 / 2) * g - (43 / 2) * r + 0x8080) >> 8;
}
static __inline uint8_t RGB2xToVJ(uint16_t r, uint16_t g, uint16_t b) {
  return ((127 / 2) * r - (107 / 2) * g - (20 / 2) * b + 0x8080) >> 8;
}
#endif

// ARGBToYJ_C and ARGBToUVJ_C
// Intel version mimic SSE/AVX which does 2 pavgb
#if LIBYUV_ARGBTOUV_PAVGB
#define MAKEROWYJ(NAME, R, G, B, BPP)                                       \
  void NAME##ToYJRow_C(const uint8_t* src_rgb, uint8_t* dst_y, int width) { \
    int x;                                                                  \
    for (x = 0; x < width; ++x) {                                           \
      dst_y[0] = RGBToYJ(src_rgb[R], src_rgb[G], src_rgb[B]);               \
      src_rgb += BPP;                                                       \
      dst_y += 1;                                                           \
    }                                                                       \
  }                                                                         \
  void NAME##ToUVJRow_C(const uint8_t* src_rgb, int src_stride_rgb,         \
                        uint8_t* dst_u, uint8_t* dst_v, int width) {        \
    const uint8_t* src_rgb1 = src_rgb + src_stride_rgb;                     \
    int x;                                                                  \
    for (x = 0; x < width - 1; x += 2) {                                    \
      uint8_t ab = AVGB(AVGB(src_rgb[B], src_rgb1[B]),                      \
                        AVGB(src_rgb[B + BPP], src_rgb1[B + BPP]));         \
      uint8_t ag = AVGB(AVGB(src_rgb[G], src_rgb1[G]),                      \
                        AVGB(src_rgb[G + BPP], src_rgb1[G + BPP]));         \
      uint8_t ar = AVGB(AVGB(src_rgb[R], src_rgb1[R]),                      \
                        AVGB(src_rgb[R + BPP], src_rgb1[R + BPP]));         \
      dst_u[0] = RGBToUJ(ar, ag, ab);                                       \
      dst_v[0] = RGBToVJ(ar, ag, ab);                                       \
      src_rgb += BPP * 2;                                                   \
      src_rgb1 += BPP * 2;                                                  \
      dst_u += 1;                                                           \
      dst_v += 1;                                                           \
    }                                                                       \
    if (width & 1) {                                                        \
      uint8_t ab = AVGB(src_rgb[B], src_rgb1[B]);                           \
      uint8_t ag = AVGB(src_rgb[G], src_rgb1[G]);                           \
      uint8_t ar = AVGB(src_rgb[R], src_rgb1[R]);                           \
      dst_u[0] = RGBToUJ(ar, ag, ab);                                       \
      dst_v[0] = RGBToVJ(ar, ag, ab);                                       \
    }                                                                       \
  }
#else
// ARM version does sum / 2 then multiply by 2x smaller coefficients
#define MAKEROWYJ(NAME, R, G, B, BPP)                                       \
  void NAME##ToYJRow_C(const uint8_t* src_rgb, uint8_t* dst_y, int width) { \
    int x;                                                                  \
    for (x = 0; x < width; ++x) {                                           \
      dst_y[0] = RGBToYJ(src_rgb[R], src_rgb[G], src_rgb[B]);               \
      src_rgb += BPP;                                                       \
      dst_y += 1;                                                           \
    }                                                                       \
  }                                                                         \
  void NAME##ToUVJRow_C(const uint8_t* src_rgb, int src_stride_rgb,         \
                        uint8_t* dst_u, uint8_t* dst_v, int width) {        \
    const uint8_t* src_rgb1 = src_rgb + src_stride_rgb;                     \
    int x;                                                                  \
    for (x = 0; x < width - 1; x += 2) {                                    \
      uint16_t ab = (src_rgb[B] + src_rgb[B + BPP] + src_rgb1[B] +          \
                     src_rgb1[B + BPP] + 1) >>                              \
                    1;                                                      \
      uint16_t ag = (src_rgb[G] + src_rgb[G + BPP] + src_rgb1[G] +          \
                     src_rgb1[G + BPP] + 1) >>                              \
                    1;                                                      \
      uint16_t ar = (src_rgb[R] + src_rgb[R + BPP] + src_rgb1[R] +          \
                     src_rgb1[R + BPP] + 1) >>                              \
                    1;                                                      \
      dst_u[0] = RGB2xToUJ(ar, ag, ab);                                     \
      dst_v[0] = RGB2xToVJ(ar, ag, ab);                                     \
      src_rgb += BPP * 2;                                                   \
      src_rgb1 += BPP * 2;                                                  \
      dst_u += 1;                                                           \
      dst_v += 1;                                                           \
    }                                                                       \
    if (width & 1) {                                                        \
      uint16_t ab = (src_rgb[B] + src_rgb1[B]);                             \
      uint16_t ag = (src_rgb[G] + src_rgb1[G]);                             \
      uint16_t ar = (src_rgb[R] + src_rgb1[R]);                             \
      dst_u[0] = RGB2xToUJ(ar, ag, ab);                                     \
      dst_v[0] = RGB2xToVJ(ar, ag, ab);                                     \
    }                                                                       \
  }

#endif

MAKEROWYJ(ARGB, 2, 1, 0, 4)

MAKEROWYJ(ABGR, 0, 1, 2, 4)

MAKEROWYJ(RGBA, 3, 2, 1, 4)

MAKEROWYJ(RGB24, 2, 1, 0, 3)

MAKEROWYJ(RAW, 0, 1, 2, 3)

#undef MAKEROWYJ

void RGB565ToYRow_C(const uint8_t *src_rgb565, uint8_t *dst_y, int width) {
    int x;
    for (x = 0; x < width; ++x) {
        uint8_t b = src_rgb565[0] & 0x1f;
        uint8_t g = STATIC_CAST(
                uint8_t, (src_rgb565[0] >> 5) | ((src_rgb565[1] & 0x07) << 3));
        uint8_t r = src_rgb565[1] >> 3;
        b = STATIC_CAST(uint8_t, (b << 3) | (b >> 2));
        g = STATIC_CAST(uint8_t, (g << 2) | (g >> 4));
        r = STATIC_CAST(uint8_t, (r << 3) | (r >> 2));
        dst_y[0] = RGBToY(r, g, b);
        src_rgb565 += 2;
        dst_y += 1;
    }
}

void RGB565ToUVRow_C(const uint8_t *src_rgb565,
                     int src_stride_rgb565,
                     uint8_t *dst_u,
                     uint8_t *dst_v,
                     int width) {
    const uint8_t *next_rgb565 = src_rgb565 + src_stride_rgb565;
    int x;
    for (x = 0; x < width - 1; x += 2) {
        uint8_t b0 = STATIC_CAST(uint8_t, src_rgb565[0] & 0x1f);
        uint8_t g0 = STATIC_CAST(
                uint8_t, (src_rgb565[0] >> 5) | ((src_rgb565[1] & 0x07) << 3));
        uint8_t r0 = STATIC_CAST(uint8_t, src_rgb565[1] >> 3);
        uint8_t b1 = STATIC_CAST(uint8_t, src_rgb565[2] & 0x1f);
        uint8_t g1 = STATIC_CAST(
                uint8_t, (src_rgb565[2] >> 5) | ((src_rgb565[3] & 0x07) << 3));
        uint8_t r1 = STATIC_CAST(uint8_t, src_rgb565[3] >> 3);
        uint8_t b2 = STATIC_CAST(uint8_t, next_rgb565[0] & 0x1f);
        uint8_t g2 = STATIC_CAST(
                uint8_t, (next_rgb565[0] >> 5) | ((next_rgb565[1] & 0x07) << 3));
        uint8_t r2 = STATIC_CAST(uint8_t, next_rgb565[1] >> 3);
        uint8_t b3 = STATIC_CAST(uint8_t, next_rgb565[2] & 0x1f);
        uint8_t g3 = STATIC_CAST(
                uint8_t, (next_rgb565[2] >> 5) | ((next_rgb565[3] & 0x07) << 3));
        uint8_t r3 = STATIC_CAST(uint8_t, next_rgb565[3] >> 3);

        b0 = STATIC_CAST(uint8_t, (b0 << 3) | (b0 >> 2));
        g0 = STATIC_CAST(uint8_t, (g0 << 2) | (g0 >> 4));
        r0 = STATIC_CAST(uint8_t, (r0 << 3) | (r0 >> 2));
        b1 = STATIC_CAST(uint8_t, (b1 << 3) | (b1 >> 2));
        g1 = STATIC_CAST(uint8_t, (g1 << 2) | (g1 >> 4));
        r1 = STATIC_CAST(uint8_t, (r1 << 3) | (r1 >> 2));
        b2 = STATIC_CAST(uint8_t, (b2 << 3) | (b2 >> 2));
        g2 = STATIC_CAST(uint8_t, (g2 << 2) | (g2 >> 4));
        r2 = STATIC_CAST(uint8_t, (r2 << 3) | (r2 >> 2));
        b3 = STATIC_CAST(uint8_t, (b3 << 3) | (b3 >> 2));
        g3 = STATIC_CAST(uint8_t, (g3 << 2) | (g3 >> 4));
        r3 = STATIC_CAST(uint8_t, (r3 << 3) | (r3 >> 2));

#if LIBYUV_ARGBTOUV_PAVGB
        uint8_t ab = AVGB(AVGB(b0, b2), AVGB(b1, b3));
        uint8_t ag = AVGB(AVGB(g0, g2), AVGB(g1, g3));
        uint8_t ar = AVGB(AVGB(r0, r2), AVGB(r1, r3));
        dst_u[0] = RGBToU(ar, ag, ab);
        dst_v[0] = RGBToV(ar, ag, ab);
#else
        uint16_t b = (b0 + b1 + b2 + b3 + 1) >> 1;
        uint16_t g = (g0 + g1 + g2 + g3 + 1) >> 1;
        uint16_t r = (r0 + r1 + r2 + r3 + 1) >> 1;
        dst_u[0] = RGB2xToU(r, g, b);
        dst_v[0] = RGB2xToV(r, g, b);
#endif

        src_rgb565 += 4;
        next_rgb565 += 4;
        dst_u += 1;
        dst_v += 1;
    }
    if (width & 1) {
        uint8_t b0 = STATIC_CAST(uint8_t, src_rgb565[0] & 0x1f);
        uint8_t g0 = STATIC_CAST(
                uint8_t, (src_rgb565[0] >> 5) | ((src_rgb565[1] & 0x07) << 3));
        uint8_t r0 = STATIC_CAST(uint8_t, src_rgb565[1] >> 3);
        uint8_t b2 = STATIC_CAST(uint8_t, next_rgb565[0] & 0x1f);
        uint8_t g2 = STATIC_CAST(
                uint8_t, (next_rgb565[0] >> 5) | ((next_rgb565[1] & 0x07) << 3));
        uint8_t r2 = STATIC_CAST(uint8_t, next_rgb565[1] >> 3);
        b0 = STATIC_CAST(uint8_t, (b0 << 3) | (b0 >> 2));
        g0 = STATIC_CAST(uint8_t, (g0 << 2) | (g0 >> 4));
        r0 = STATIC_CAST(uint8_t, (r0 << 3) | (r0 >> 2));
        b2 = STATIC_CAST(uint8_t, (b2 << 3) | (b2 >> 2));
        g2 = STATIC_CAST(uint8_t, (g2 << 2) | (g2 >> 4));
        r2 = STATIC_CAST(uint8_t, (r2 << 3) | (r2 >> 2));

#if LIBYUV_ARGBTOUV_PAVGB
        uint8_t ab = AVGB(b0, b2);
        uint8_t ag = AVGB(g0, g2);
        uint8_t ar = AVGB(r0, r2);
        dst_u[0] = RGBToU(ar, ag, ab);
        dst_v[0] = RGBToV(ar, ag, ab);
#else
        uint16_t b = b0 + b2;
        uint16_t g = g0 + g2;
        uint16_t r = r0 + r2;
        dst_u[0] = RGB2xToU(r, g, b);
        dst_v[0] = RGB2xToV(r, g, b);
#endif
    }
}

#define REPEAT8(v) (v) | ((v) << 8)
#define SHADE(f, v) v* f >> 24

#undef REPEAT8
#undef SHADE

#define REPEAT8(v) (v) | ((v) << 8)
#define SHADE(f, v) v* f >> 16

#undef REPEAT8
#undef SHADE

#define SHADE(f, v) clamp255(v + f)

#undef SHADE

#define SHADE(f, v) clamp0(f - v)

#undef SHADE

// Macros to create SIMD specific yuv to rgb conversion constants.

// clang-format off

#if defined(__aarch64__) || defined(__arm__)
// Bias values include subtract 128 from U and V, bias from Y and rounding.
// For B and R bias is negative. For G bias is positive.
#define YUVCONSTANTSBODY(YG, YB, UB, UG, VG, VR)                             \
  {{UB, VR, UG, VG, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},                     \
   {YG, (UB * 128 - YB), (UG * 128 + VG * 128 + YB), (VR * 128 - YB), YB, 0, \
    0, 0}}
#else
#define YUVCONSTANTSBODY(YG, YB, UB, UG, VG, VR)                     \
  {{UB, 0, UB, 0, UB, 0, UB, 0, UB, 0, UB, 0, UB, 0, UB, 0,          \
    UB, 0, UB, 0, UB, 0, UB, 0, UB, 0, UB, 0, UB, 0, UB, 0},         \
   {UG, VG, UG, VG, UG, VG, UG, VG, UG, VG, UG, VG, UG, VG, UG, VG,  \
    UG, VG, UG, VG, UG, VG, UG, VG, UG, VG, UG, VG, UG, VG, UG, VG}, \
   {0, VR, 0, VR, 0, VR, 0, VR, 0, VR, 0, VR, 0, VR, 0, VR,          \
    0, VR, 0, VR, 0, VR, 0, VR, 0, VR, 0, VR, 0, VR, 0, VR},         \
   {YG, YG, YG, YG, YG, YG, YG, YG, YG, YG, YG, YG, YG, YG, YG, YG}, \
   {YB, YB, YB, YB, YB, YB, YB, YB, YB, YB, YB, YB, YB, YB, YB, YB}}
#endif

// clang-format on

#define MAKEYUVCONSTANTS(name, YG, YB, UB, UG, VG, VR)            \
  const struct YuvConstants SIMD_ALIGNED(kYuv##name##Constants) = \
      YUVCONSTANTSBODY(YG, YB, UB, UG, VG, VR);                   \
  const struct YuvConstants SIMD_ALIGNED(kYvu##name##Constants) = \
      YUVCONSTANTSBODY(YG, YB, VR, VG, UG, UB);

// TODO(fbarchard): Generate SIMD structures from float matrix.

// BT.601 limited range YUV to RGB reference
//  R = (Y - 16) * 1.164             + V * 1.596
//  G = (Y - 16) * 1.164 - U * 0.391 - V * 0.813
//  B = (Y - 16) * 1.164 + U * 2.018
// KR = 0.299; KB = 0.114

// U and V contributions to R,G,B.
#if defined(LIBYUV_UNLIMITED_DATA) || defined(LIBYUV_UNLIMITED_BT601)
#define UB 129 /* round(2.018 * 64) */
#else
#define UB 128 /* max(128, round(2.018 * 64)) */
#endif
#define UG 25  /* round(0.391 * 64) */
#define VG 52  /* round(0.813 * 64) */
#define VR 102 /* round(1.596 * 64) */

// Y contribution to R,G,B.  Scale and bias.
#define YG 18997 /* round(1.164 * 64 * 256 * 256 / 257) */
#define YB -1160 /* 1.164 * 64 * -16 + 64 / 2 */

MAKEYUVCONSTANTS(I601, YG, YB, UB, UG, VG, VR)

#undef YG
#undef YB
#undef UB
#undef UG
#undef VG
#undef VR

// BT.601 full range YUV to RGB reference (aka JPEG)
// *  R = Y               + V * 1.40200
// *  G = Y - U * 0.34414 - V * 0.71414
// *  B = Y + U * 1.77200
// KR = 0.299; KB = 0.114

// U and V contributions to R,G,B.
#define UB 113 /* round(1.77200 * 64) */
#define UG 22  /* round(0.34414 * 64) */
#define VG 46  /* round(0.71414 * 64) */
#define VR 90  /* round(1.40200 * 64) */

// Y contribution to R,G,B.  Scale and bias.
#define YG 16320 /* round(1.000 * 64 * 256 * 256 / 257) */
#define YB 32    /* 64 / 2 */

MAKEYUVCONSTANTS(JPEG, YG, YB, UB, UG, VG, VR)

#undef YG
#undef YB
#undef UB
#undef UG
#undef VG
#undef VR

// BT.709 limited range YUV to RGB reference
//  R = (Y - 16) * 1.164             + V * 1.793
//  G = (Y - 16) * 1.164 - U * 0.213 - V * 0.533
//  B = (Y - 16) * 1.164 + U * 2.112
//  KR = 0.2126, KB = 0.0722

// U and V contributions to R,G,B.
#if defined(LIBYUV_UNLIMITED_DATA) || defined(LIBYUV_UNLIMITED_BT709)
#define UB 135 /* round(2.112 * 64) */
#else
#define UB 128 /* max(128, round(2.112 * 64)) */
#endif
#define UG 14  /* round(0.213 * 64) */
#define VG 34  /* round(0.533 * 64) */
#define VR 115 /* round(1.793 * 64) */

// Y contribution to R,G,B.  Scale and bias.
#define YG 18997 /* round(1.164 * 64 * 256 * 256 / 257) */
#define YB -1160 /* 1.164 * 64 * -16 + 64 / 2 */

MAKEYUVCONSTANTS(H709, YG, YB, UB, UG, VG, VR)

#undef YG
#undef YB
#undef UB
#undef UG
#undef VG
#undef VR

// BT.709 full range YUV to RGB reference
//  R = Y               + V * 1.5748
//  G = Y - U * 0.18732 - V * 0.46812
//  B = Y + U * 1.8556
//  KR = 0.2126, KB = 0.0722

// U and V contributions to R,G,B.
#define UB 119 /* round(1.8556 * 64) */
#define UG 12  /* round(0.18732 * 64) */
#define VG 30  /* round(0.46812 * 64) */
#define VR 101 /* round(1.5748 * 64) */

// Y contribution to R,G,B.  Scale and bias.  (same as jpeg)
#define YG 16320 /* round(1 * 64 * 256 * 256 / 257) */
#define YB 32    /* 64 / 2 */

MAKEYUVCONSTANTS(F709, YG, YB, UB, UG, VG, VR)

#undef YG
#undef YB
#undef UB
#undef UG
#undef VG
#undef VR

// BT.2020 limited range YUV to RGB reference
//  R = (Y - 16) * 1.164384                + V * 1.67867
//  G = (Y - 16) * 1.164384 - U * 0.187326 - V * 0.65042
//  B = (Y - 16) * 1.164384 + U * 2.14177
// KR = 0.2627; KB = 0.0593

// U and V contributions to R,G,B.
#if defined(LIBYUV_UNLIMITED_DATA) || defined(LIBYUV_UNLIMITED_BT2020)
#define UB 137 /* round(2.142 * 64) */
#else
#define UB 128 /* max(128, round(2.142 * 64)) */
#endif
#define UG 12  /* round(0.187326 * 64) */
#define VG 42  /* round(0.65042 * 64) */
#define VR 107 /* round(1.67867 * 64) */

// Y contribution to R,G,B.  Scale and bias.
#define YG 19003 /* round(1.164384 * 64 * 256 * 256 / 257) */
#define YB -1160 /* 1.164384 * 64 * -16 + 64 / 2 */

MAKEYUVCONSTANTS(2020, YG, YB, UB, UG, VG, VR)

#undef YG
#undef YB
#undef UB
#undef UG
#undef VG
#undef VR

// BT.2020 full range YUV to RGB reference
//  R = Y                + V * 1.474600
//  G = Y - U * 0.164553 - V * 0.571353
//  B = Y + U * 1.881400
// KR = 0.2627; KB = 0.0593

#define UB 120 /* round(1.881400 * 64) */
#define UG 11  /* round(0.164553 * 64) */
#define VG 37  /* round(0.571353 * 64) */
#define VR 94  /* round(1.474600 * 64) */

// Y contribution to R,G,B.  Scale and bias.  (same as jpeg)
#define YG 16320 /* round(1 * 64 * 256 * 256 / 257) */
#define YB 32    /* 64 / 2 */

MAKEYUVCONSTANTS(V2020, YG, YB, UB, UG, VG, VR)

#undef YG
#undef YB
#undef UB
#undef UG
#undef VG
#undef VR

#undef BB
#undef BG
#undef BR

#undef MAKEYUVCONSTANTS

#if defined(__aarch64__) || defined(__arm__)
#define LOAD_YUV_CONSTANTS                 \
  int ub = yuvconstants->kUVCoeff[0];      \
  int vr = yuvconstants->kUVCoeff[1];      \
  int ug = yuvconstants->kUVCoeff[2];      \
  int vg = yuvconstants->kUVCoeff[3];      \
  int yg = yuvconstants->kRGBCoeffBias[0]; \
  int bb = yuvconstants->kRGBCoeffBias[1]; \
  int bg = yuvconstants->kRGBCoeffBias[2]; \
  int br = yuvconstants->kRGBCoeffBias[3]

#define CALC_RGB16                         \
  int32_t y1 = (uint32_t)(y32 * yg) >> 16; \
  int b16 = y1 + (u * ub) - bb;            \
  int g16 = y1 + bg - (u * ug + v * vg);   \
  int r16 = y1 + (v * vr) - br
#else
#define LOAD_YUV_CONSTANTS           \
  int ub = yuvconstants->kUVToB[0];  \
  int ug = yuvconstants->kUVToG[0];  \
  int vg = yuvconstants->kUVToG[1];  \
  int vr = yuvconstants->kUVToR[1];  \
  int yg = yuvconstants->kYToRgb[0]; \
  int yb = yuvconstants->kYBiasToRgb[0]

#define CALC_RGB16                                \
  int32_t y1 = ((uint32_t)(y32 * yg) >> 16) + yb; \
  int8_t ui = (int8_t)u;                          \
  int8_t vi = (int8_t)v;                          \
  ui -= 0x80;                                     \
  vi -= 0x80;                                     \
  int b16 = y1 + (ui * ub);                       \
  int g16 = y1 - (ui * ug + vi * vg);             \
  int r16 = y1 + (vi * vr)
#endif

void MirrorRow_C(const uint8_t *src, uint8_t *dst, int width) {
    int x;
    src += width - 1;
    for (x = 0; x < width - 1; x += 2) {
        dst[x] = src[0];
        dst[x + 1] = src[-1];
        src -= 2;
    }
    if (width & 1) {
        dst[width - 1] = src[0];
    }
}

// Use scale to convert lsb formats to msb, depending how many bits there are:
// 32768 = 9 bits
// 16384 = 10 bits
// 4096 = 12 bits
// 256 = 16 bits
// TODO(fbarchard): change scale to bits
#define C16TO8(v, scale) clamp255(((v) * (scale)) >> 16)

void CopyRow_C(const uint8_t *src, uint8_t *dst, int count) {
    memcpy(dst, src, count);
}

// Divide source RGB by alpha and store to destination.
// b = (b * 255 + (a / 2)) / a;
// g = (g * 255 + (a / 2)) / a;
// r = (r * 255 + (a / 2)) / a;
// Reciprocal method is off by 1 on some values. ie 125
// 8.8 fixed point inverse table with 1.0 in upper short and 1 / a in lower.
#define T(a) 0x01000000 + (0x10000 / a)
const uint32_t fixed_invtbl8[256] = {
        0x01000000, 0x0100ffff, T(0x02), T(0x03), T(0x04), T(0x05), T(0x06),
        T(0x07), T(0x08), T(0x09), T(0x0a), T(0x0b), T(0x0c), T(0x0d),
        T(0x0e), T(0x0f), T(0x10), T(0x11), T(0x12), T(0x13), T(0x14),
        T(0x15), T(0x16), T(0x17), T(0x18), T(0x19), T(0x1a), T(0x1b),
        T(0x1c), T(0x1d), T(0x1e), T(0x1f), T(0x20), T(0x21), T(0x22),
        T(0x23), T(0x24), T(0x25), T(0x26), T(0x27), T(0x28), T(0x29),
        T(0x2a), T(0x2b), T(0x2c), T(0x2d), T(0x2e), T(0x2f), T(0x30),
        T(0x31), T(0x32), T(0x33), T(0x34), T(0x35), T(0x36), T(0x37),
        T(0x38), T(0x39), T(0x3a), T(0x3b), T(0x3c), T(0x3d), T(0x3e),
        T(0x3f), T(0x40), T(0x41), T(0x42), T(0x43), T(0x44), T(0x45),
        T(0x46), T(0x47), T(0x48), T(0x49), T(0x4a), T(0x4b), T(0x4c),
        T(0x4d), T(0x4e), T(0x4f), T(0x50), T(0x51), T(0x52), T(0x53),
        T(0x54), T(0x55), T(0x56), T(0x57), T(0x58), T(0x59), T(0x5a),
        T(0x5b), T(0x5c), T(0x5d), T(0x5e), T(0x5f), T(0x60), T(0x61),
        T(0x62), T(0x63), T(0x64), T(0x65), T(0x66), T(0x67), T(0x68),
        T(0x69), T(0x6a), T(0x6b), T(0x6c), T(0x6d), T(0x6e), T(0x6f),
        T(0x70), T(0x71), T(0x72), T(0x73), T(0x74), T(0x75), T(0x76),
        T(0x77), T(0x78), T(0x79), T(0x7a), T(0x7b), T(0x7c), T(0x7d),
        T(0x7e), T(0x7f), T(0x80), T(0x81), T(0x82), T(0x83), T(0x84),
        T(0x85), T(0x86), T(0x87), T(0x88), T(0x89), T(0x8a), T(0x8b),
        T(0x8c), T(0x8d), T(0x8e), T(0x8f), T(0x90), T(0x91), T(0x92),
        T(0x93), T(0x94), T(0x95), T(0x96), T(0x97), T(0x98), T(0x99),
        T(0x9a), T(0x9b), T(0x9c), T(0x9d), T(0x9e), T(0x9f), T(0xa0),
        T(0xa1), T(0xa2), T(0xa3), T(0xa4), T(0xa5), T(0xa6), T(0xa7),
        T(0xa8), T(0xa9), T(0xaa), T(0xab), T(0xac), T(0xad), T(0xae),
        T(0xaf), T(0xb0), T(0xb1), T(0xb2), T(0xb3), T(0xb4), T(0xb5),
        T(0xb6), T(0xb7), T(0xb8), T(0xb9), T(0xba), T(0xbb), T(0xbc),
        T(0xbd), T(0xbe), T(0xbf), T(0xc0), T(0xc1), T(0xc2), T(0xc3),
        T(0xc4), T(0xc5), T(0xc6), T(0xc7), T(0xc8), T(0xc9), T(0xca),
        T(0xcb), T(0xcc), T(0xcd), T(0xce), T(0xcf), T(0xd0), T(0xd1),
        T(0xd2), T(0xd3), T(0xd4), T(0xd5), T(0xd6), T(0xd7), T(0xd8),
        T(0xd9), T(0xda), T(0xdb), T(0xdc), T(0xdd), T(0xde), T(0xdf),
        T(0xe0), T(0xe1), T(0xe2), T(0xe3), T(0xe4), T(0xe5), T(0xe6),
        T(0xe7), T(0xe8), T(0xe9), T(0xea), T(0xeb), T(0xec), T(0xed),
        T(0xee), T(0xef), T(0xf0), T(0xf1), T(0xf2), T(0xf3), T(0xf4),
        T(0xf5), T(0xf6), T(0xf7), T(0xf8), T(0xf9), T(0xfa), T(0xfb),
        T(0xfc), T(0xfd), T(0xfe), 0x01000100};
#undef T

// Blend 2 rows into 1.
static void HalfRow_C(const uint8_t *src_uv,
                      ptrdiff_t src_uv_stride,
                      uint8_t *dst_uv,
                      int width) {
    int x;
    for (x = 0; x < width; ++x) {
        dst_uv[x] = (src_uv[x] + src_uv[src_uv_stride + x] + 1) >> 1;
    }
}

// C version 2x2 -> 2x1.
void InterpolateRow_C(uint8_t *dst_ptr,
                      const uint8_t *src_ptr,
                      ptrdiff_t src_stride,
                      int width,
                      int source_y_fraction) {
    int y1_fraction = source_y_fraction;
    int y0_fraction = 256 - y1_fraction;
    const uint8_t *src_ptr1 = src_ptr + src_stride;
    int x;
    assert(source_y_fraction >= 0);
    assert(source_y_fraction < 256);

    if (y1_fraction == 0) {
        memcpy(dst_ptr, src_ptr, width);
        return;
    }
    if (y1_fraction == 128) {
        HalfRow_C(src_ptr, src_stride, dst_ptr, width);
        return;
    }
    for (x = 0; x < width; ++x) {
        dst_ptr[0] = STATIC_CAST(
                uint8_t,
                (src_ptr[0] * y0_fraction + src_ptr1[0] * y1_fraction + 128) >> 8);
        ++src_ptr;
        ++src_ptr1;
        ++dst_ptr;
    }
}

// Work around GCC 7 punning warning -Wstrict-aliasing
#if defined(__GNUC__)
typedef uint32_t __attribute__((__may_alias__)) uint32_alias_t;
#else
typedef uint32_t uint32_alias_t;
#endif

#undef STATIC_CAST
