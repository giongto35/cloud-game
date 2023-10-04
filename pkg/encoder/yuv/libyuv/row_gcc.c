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

// This module is for GCC x86 and x64.
#if !defined(LIBYUV_DISABLE_X86) && (defined(__x86_64__) || defined(__i386__))

#if defined(HAS_ARGBTOYROW_SSSE3) || defined(HAS_ARGBGRAYROW_SSSE3)

// Constants for ARGB
static const uvec8 kARGBToY = {25u, 129u, 66u, 0u, 25u, 129u, 66u, 0u,
                               25u, 129u, 66u, 0u, 25u, 129u, 66u, 0u};


#endif  // defined(HAS_ARGBTOYROW_SSSE3) || defined(HAS_ARGBGRAYROW_SSSE3)

#if defined(HAS_ARGBTOYROW_SSSE3) || defined(HAS_I422TOARGBROW_SSSE3)

static const vec8 kARGBToU = {112, -74, -38, 0, 112, -74, -38, 0,
                              112, -74, -38, 0, 112, -74, -38, 0};

static const vec8 kARGBToV = {-18, -94, 112, 0, -18, -94, 112, 0,
                              -18, -94, 112, 0, -18, -94, 112, 0};

// Constants for BGRA
static const uvec8 kBGRAToY = {0u, 66u, 129u, 25u, 0u, 66u, 129u, 25u,
                               0u, 66u, 129u, 25u, 0u, 66u, 129u, 25u};

static const vec8 kBGRAToU = {0, -38, -74, 112, 0, -38, -74, 112,
                              0, -38, -74, 112, 0, -38, -74, 112};

static const vec8 kBGRAToV = {0, 112, -94, -18, 0, 112, -94, -18,
                              0, 112, -94, -18, 0, 112, -94, -18};

// Constants for ABGR
static const uvec8 kABGRToY = {66u, 129u, 25u, 0u, 66u, 129u, 25u, 0u,
                               66u, 129u, 25u, 0u, 66u, 129u, 25u, 0u};

static const vec8 kABGRToU = {-38, -74, 112, 0, -38, -74, 112, 0,
                              -38, -74, 112, 0, -38, -74, 112, 0};

static const vec8 kABGRToV = {112, -94, -18, 0, 112, -94, -18, 0,
                              112, -94, -18, 0, 112, -94, -18, 0};

// Constants for RGBA.
//static const uvec8 kRGBAToY = {0u, 25u, 129u, 66u, 0u, 25u, 129u, 66u,
//                               0u, 25u, 129u, 66u, 0u, 25u, 129u, 66u};

static const vec8 kRGBAToU = {0, 112, -74, -38, 0, 112, -74, -38,
                              0, 112, -74, -38, 0, 112, -74, -38};

static const vec8 kRGBAToV = {0, -18, -94, 112, 0, -18, -94, 112,
                              0, -18, -94, 112, 0, -18, -94, 112};

static const uvec16 kAddY16 = {0x7e80u, 0x7e80u, 0x7e80u, 0x7e80u,
                               0x7e80u, 0x7e80u, 0x7e80u, 0x7e80u};

static const uvec8 kAddUV128 = {128u, 128u, 128u, 128u, 128u, 128u, 128u, 128u,
                                128u, 128u, 128u, 128u, 128u, 128u, 128u, 128u};

static const uvec16 kSub128 = {0x8080u, 0x8080u, 0x8080u, 0x8080u,
                               0x8080u, 0x8080u, 0x8080u, 0x8080u};

#endif  // defined(HAS_ARGBTOYROW_SSSE3) || defined(HAS_I422TOARGBROW_SSSE3)

// clang-format off

// TODO(mraptis): Consider passing R, G, B multipliers as parameter.
// round parameter is register containing value to add before shift.
#define RGBTOY(round)                            \
  "1:                                        \n" \
  "movdqu    (%0),%%xmm0                     \n" \
  "movdqu    0x10(%0),%%xmm1                 \n" \
  "movdqu    0x20(%0),%%xmm2                 \n" \
  "movdqu    0x30(%0),%%xmm3                 \n" \
  "psubb     %%xmm5,%%xmm0                   \n" \
  "psubb     %%xmm5,%%xmm1                   \n" \
  "psubb     %%xmm5,%%xmm2                   \n" \
  "psubb     %%xmm5,%%xmm3                   \n" \
  "movdqu    %%xmm4,%%xmm6                   \n" \
  "pmaddubsw %%xmm0,%%xmm6                   \n" \
  "movdqu    %%xmm4,%%xmm0                   \n" \
  "pmaddubsw %%xmm1,%%xmm0                   \n" \
  "movdqu    %%xmm4,%%xmm1                   \n" \
  "pmaddubsw %%xmm2,%%xmm1                   \n" \
  "movdqu    %%xmm4,%%xmm2                   \n" \
  "pmaddubsw %%xmm3,%%xmm2                   \n" \
  "lea       0x40(%0),%0                     \n" \
  "phaddw    %%xmm0,%%xmm6                   \n" \
  "phaddw    %%xmm2,%%xmm1                   \n" \
  "prefetcht0 1280(%0)                       \n" \
  "paddw     %%" #round ",%%xmm6             \n" \
  "paddw     %%" #round ",%%xmm1             \n" \
  "psrlw     $0x8,%%xmm6                     \n" \
  "psrlw     $0x8,%%xmm1                     \n" \
  "packuswb  %%xmm1,%%xmm6                   \n" \
  "movdqu    %%xmm6,(%1)                     \n" \
  "lea       0x10(%1),%1                     \n" \
  "sub       $0x10,%2                        \n" \
  "jg        1b                              \n"

#define RGBTOY_AVX2(round)                                       \
  "1:                                        \n"                 \
  "vmovdqu    (%0),%%ymm0                    \n"                 \
  "vmovdqu    0x20(%0),%%ymm1                \n"                 \
  "vmovdqu    0x40(%0),%%ymm2                \n"                 \
  "vmovdqu    0x60(%0),%%ymm3                \n"                 \
  "vpsubb     %%ymm5, %%ymm0, %%ymm0         \n"                 \
  "vpsubb     %%ymm5, %%ymm1, %%ymm1         \n"                 \
  "vpsubb     %%ymm5, %%ymm2, %%ymm2         \n"                 \
  "vpsubb     %%ymm5, %%ymm3, %%ymm3         \n"                 \
  "vpmaddubsw %%ymm0,%%ymm4,%%ymm0           \n"                 \
  "vpmaddubsw %%ymm1,%%ymm4,%%ymm1           \n"                 \
  "vpmaddubsw %%ymm2,%%ymm4,%%ymm2           \n"                 \
  "vpmaddubsw %%ymm3,%%ymm4,%%ymm3           \n"                 \
  "lea       0x80(%0),%0                     \n"                 \
  "vphaddw    %%ymm1,%%ymm0,%%ymm0           \n" /* mutates. */  \
  "vphaddw    %%ymm3,%%ymm2,%%ymm2           \n"                 \
  "prefetcht0 1280(%0)                       \n"                 \
  "vpaddw     %%" #round ",%%ymm0,%%ymm0     \n" /* Add .5 for rounding. */             \
  "vpaddw     %%" #round ",%%ymm2,%%ymm2     \n" \
  "vpsrlw     $0x8,%%ymm0,%%ymm0             \n"                 \
  "vpsrlw     $0x8,%%ymm2,%%ymm2             \n"                 \
  "vpackuswb  %%ymm2,%%ymm0,%%ymm0           \n" /* mutates. */  \
  "vpermd     %%ymm0,%%ymm6,%%ymm0           \n" /* unmutate. */ \
  "vmovdqu    %%ymm0,(%1)                    \n"                 \
  "lea       0x20(%1),%1                     \n"                 \
  "sub       $0x20,%2                        \n"                 \
  "jg        1b                              \n"                 \
  "vzeroupper                                \n"

// clang-format on

#ifdef HAS_ARGBTOYROW_SSSE3

// Convert 16 ARGB pixels (64 bytes) to 16 Y values.
void ARGBToYRow_SSSE3(const uint8_t *src_argb, uint8_t *dst_y, int width) {
    asm volatile(
            "movdqa      %3,%%xmm4                     \n"
            "movdqa      %4,%%xmm5                     \n"
            "movdqa      %5,%%xmm7                     \n"

            LABELALIGN RGBTOY(xmm7)
            : "+r"(src_argb),  // %0
    "+r"(dst_y),     // %1
    "+r"(width)      // %2
            : "m"(kARGBToY),   // %3
    "m"(kSub128),    // %4
    "m"(kAddY16)     // %5
            : "memory", "cc", "xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6",
    "xmm7");
}

#endif  // HAS_ARGBTOYROW_SSSE3

#if defined(HAS_ARGBTOYROW_AVX2) || defined(HAS_ABGRTOYROW_AVX2) || \
    defined(HAS_ARGBEXTRACTALPHAROW_AVX2)
// vpermd for vphaddw + vpackuswb vpermd.
static const lvec32 kPermdARGBToY_AVX = {0, 4, 1, 5, 2, 6, 3, 7};
#endif

#ifdef HAS_ARGBTOYROW_AVX2

// Convert 32 ARGB pixels (128 bytes) to 32 Y values.
void ARGBToYRow_AVX2(const uint8_t *src_argb, uint8_t *dst_y, int width) {
    asm volatile(
            "vbroadcastf128 %3,%%ymm4                  \n"
            "vbroadcastf128 %4,%%ymm5                  \n"
            "vbroadcastf128 %5,%%ymm7                  \n"
            "vmovdqu     %6,%%ymm6                     \n" LABELALIGN RGBTOY_AVX2(
                    ymm7) "vzeroupper                                \n"
            : "+r"(src_argb),         // %0
    "+r"(dst_y),            // %1
    "+r"(width)             // %2
            : "m"(kARGBToY),          // %3
    "m"(kSub128),           // %4
    "m"(kAddY16),           // %5
    "m"(kPermdARGBToY_AVX)  // %6
            : "memory", "cc", "xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6",
    "xmm7");
}

#endif  // HAS_ARGBTOYROW_AVX2

#ifdef HAS_ABGRTOYROW_AVX2

// Convert 32 ABGR pixels (128 bytes) to 32 Y values.
void ABGRToYRow_AVX2(const uint8_t *src_abgr, uint8_t *dst_y, int width) {
    asm volatile(
            "vbroadcastf128 %3,%%ymm4                  \n"
            "vbroadcastf128 %4,%%ymm5                  \n"
            "vbroadcastf128 %5,%%ymm7                  \n"
            "vmovdqu     %6,%%ymm6                     \n" LABELALIGN RGBTOY_AVX2(
                    ymm7) "vzeroupper                                \n"
            : "+r"(src_abgr),         // %0
    "+r"(dst_y),            // %1
    "+r"(width)             // %2
            : "m"(kABGRToY),          // %3
    "m"(kSub128),           // %4
    "m"(kAddY16),           // %5
    "m"(kPermdARGBToY_AVX)  // %6
            : "memory", "cc", "xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6",
    "xmm7");
}

#endif  // HAS_ABGRTOYROW_AVX2

#ifdef HAS_ARGBTOUVROW_SSSE3

void ARGBToUVRow_SSSE3(const uint8_t *src_argb,
                       int src_stride_argb,
                       uint8_t *dst_u,
                       uint8_t *dst_v,
                       int width) {
    asm volatile(
            "movdqa      %5,%%xmm3                     \n"
            "movdqa      %6,%%xmm4                     \n"
            "movdqa      %7,%%xmm5                     \n"
            "sub         %1,%2                         \n"

            LABELALIGN
            "1:                                        \n"
            "movdqu      (%0),%%xmm0                   \n"
            "movdqu      0x00(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm0                 \n"
            "movdqu      0x10(%0),%%xmm1               \n"
            "movdqu      0x10(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm1                 \n"
            "movdqu      0x20(%0),%%xmm2               \n"
            "movdqu      0x20(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm2                 \n"
            "movdqu      0x30(%0),%%xmm6               \n"
            "movdqu      0x30(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm6                 \n"

            "lea         0x40(%0),%0                   \n"
            "movdqa      %%xmm0,%%xmm7                 \n"
            "shufps      $0x88,%%xmm1,%%xmm0           \n"
            "shufps      $0xdd,%%xmm1,%%xmm7           \n"
            "pavgb       %%xmm7,%%xmm0                 \n"
            "movdqa      %%xmm2,%%xmm7                 \n"
            "shufps      $0x88,%%xmm6,%%xmm2           \n"
            "shufps      $0xdd,%%xmm6,%%xmm7           \n"
            "pavgb       %%xmm7,%%xmm2                 \n"
            "movdqa      %%xmm0,%%xmm1                 \n"
            "movdqa      %%xmm2,%%xmm6                 \n"
            "pmaddubsw   %%xmm4,%%xmm0                 \n"
            "pmaddubsw   %%xmm4,%%xmm2                 \n"
            "pmaddubsw   %%xmm3,%%xmm1                 \n"
            "pmaddubsw   %%xmm3,%%xmm6                 \n"
            "phaddw      %%xmm2,%%xmm0                 \n"
            "phaddw      %%xmm6,%%xmm1                 \n"
            "psraw       $0x8,%%xmm0                   \n"
            "psraw       $0x8,%%xmm1                   \n"
            "packsswb    %%xmm1,%%xmm0                 \n"
            "paddb       %%xmm5,%%xmm0                 \n"
            "movlps      %%xmm0,(%1)                   \n"
            "movhps      %%xmm0,0x00(%1,%2,1)          \n"
            "lea         0x8(%1),%1                    \n"
            "sub         $0x10,%3                      \n"
            "jg          1b                            \n"
            : "+r"(src_argb),                    // %0
    "+r"(dst_u),                       // %1
    "+r"(dst_v),                       // %2
    "+rm"(width)                       // %3
            : "r"((intptr_t) (src_stride_argb)),  // %4
    "m"(kARGBToV),                     // %5
    "m"(kARGBToU),                     // %6
    "m"(kAddUV128)                     // %7
            : "memory", "cc", "xmm0", "xmm1", "xmm2", "xmm6", "xmm7");
}

#endif  // HAS_ARGBTOUVROW_SSSE3

#if defined(HAS_ARGBTOUVROW_AVX2) || defined(HAS_ABGRTOUVROW_AVX2) || \
    defined(HAS_ARGBTOUVJROW_AVX2) || defined(HAS_ABGRTOUVJROW_AVX2)
// vpshufb for vphaddw + vpackuswb packed to shorts.
static const lvec8 kShufARGBToUV_AVX = {
        0, 1, 8, 9, 2, 3, 10, 11, 4, 5, 12, 13, 6, 7, 14, 15,
        0, 1, 8, 9, 2, 3, 10, 11, 4, 5, 12, 13, 6, 7, 14, 15};
#endif

#if defined(HAS_ARGBTOUVROW_AVX2)

void ARGBToUVRow_AVX2(const uint8_t *src_argb,
                      int src_stride_argb,
                      uint8_t *dst_u,
                      uint8_t *dst_v,
                      int width) {
    asm volatile(
            "vbroadcastf128 %5,%%ymm5                  \n"
            "vbroadcastf128 %6,%%ymm6                  \n"
            "vbroadcastf128 %7,%%ymm7                  \n"
            "sub         %1,%2                         \n"

            LABELALIGN
            "1:                                        \n"
            "vmovdqu     (%0),%%ymm0                   \n"
            "vmovdqu     0x20(%0),%%ymm1               \n"
            "vmovdqu     0x40(%0),%%ymm2               \n"
            "vmovdqu     0x60(%0),%%ymm3               \n"
            "vpavgb      0x00(%0,%4,1),%%ymm0,%%ymm0   \n"
            "vpavgb      0x20(%0,%4,1),%%ymm1,%%ymm1   \n"
            "vpavgb      0x40(%0,%4,1),%%ymm2,%%ymm2   \n"
            "vpavgb      0x60(%0,%4,1),%%ymm3,%%ymm3   \n"
            "lea         0x80(%0),%0                   \n"
            "vshufps     $0x88,%%ymm1,%%ymm0,%%ymm4    \n"
            "vshufps     $0xdd,%%ymm1,%%ymm0,%%ymm0    \n"
            "vpavgb      %%ymm4,%%ymm0,%%ymm0          \n"
            "vshufps     $0x88,%%ymm3,%%ymm2,%%ymm4    \n"
            "vshufps     $0xdd,%%ymm3,%%ymm2,%%ymm2    \n"
            "vpavgb      %%ymm4,%%ymm2,%%ymm2          \n"

            "vpmaddubsw  %%ymm7,%%ymm0,%%ymm1          \n"
            "vpmaddubsw  %%ymm7,%%ymm2,%%ymm3          \n"
            "vpmaddubsw  %%ymm6,%%ymm0,%%ymm0          \n"
            "vpmaddubsw  %%ymm6,%%ymm2,%%ymm2          \n"
            "vphaddw     %%ymm3,%%ymm1,%%ymm1          \n"
            "vphaddw     %%ymm2,%%ymm0,%%ymm0          \n"
            "vpsraw      $0x8,%%ymm1,%%ymm1            \n"
            "vpsraw      $0x8,%%ymm0,%%ymm0            \n"
            "vpacksswb   %%ymm0,%%ymm1,%%ymm0          \n"
            "vpermq      $0xd8,%%ymm0,%%ymm0           \n"
            "vpshufb     %8,%%ymm0,%%ymm0              \n"
            "vpaddb      %%ymm5,%%ymm0,%%ymm0          \n"

            "vextractf128 $0x0,%%ymm0,(%1)             \n"
            "vextractf128 $0x1,%%ymm0,0x0(%1,%2,1)     \n"
            "lea         0x10(%1),%1                   \n"
            "sub         $0x20,%3                      \n"
            "jg          1b                            \n"
            "vzeroupper                                \n"
            : "+r"(src_argb),                    // %0
    "+r"(dst_u),                       // %1
    "+r"(dst_v),                       // %2
    "+rm"(width)                       // %3
            : "r"((intptr_t) (src_stride_argb)),  // %4
    "m"(kAddUV128),                    // %5
    "m"(kARGBToV),                     // %6
    "m"(kARGBToU),                     // %7
    "m"(kShufARGBToUV_AVX)             // %8
            : "memory", "cc", "xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6",
    "xmm7");
}

#endif  // HAS_ARGBTOUVROW_AVX2

#ifdef HAS_ABGRTOUVROW_AVX2

void ABGRToUVRow_AVX2(const uint8_t *src_abgr,
                      int src_stride_abgr,
                      uint8_t *dst_u,
                      uint8_t *dst_v,
                      int width) {
    asm volatile(
            "vbroadcastf128 %5,%%ymm5                  \n"
            "vbroadcastf128 %6,%%ymm6                  \n"
            "vbroadcastf128 %7,%%ymm7                  \n"
            "sub         %1,%2                         \n"

            LABELALIGN
            "1:                                        \n"
            "vmovdqu     (%0),%%ymm0                   \n"
            "vmovdqu     0x20(%0),%%ymm1               \n"
            "vmovdqu     0x40(%0),%%ymm2               \n"
            "vmovdqu     0x60(%0),%%ymm3               \n"
            "vpavgb      0x00(%0,%4,1),%%ymm0,%%ymm0   \n"
            "vpavgb      0x20(%0,%4,1),%%ymm1,%%ymm1   \n"
            "vpavgb      0x40(%0,%4,1),%%ymm2,%%ymm2   \n"
            "vpavgb      0x60(%0,%4,1),%%ymm3,%%ymm3   \n"
            "lea         0x80(%0),%0                   \n"
            "vshufps     $0x88,%%ymm1,%%ymm0,%%ymm4    \n"
            "vshufps     $0xdd,%%ymm1,%%ymm0,%%ymm0    \n"
            "vpavgb      %%ymm4,%%ymm0,%%ymm0          \n"
            "vshufps     $0x88,%%ymm3,%%ymm2,%%ymm4    \n"
            "vshufps     $0xdd,%%ymm3,%%ymm2,%%ymm2    \n"
            "vpavgb      %%ymm4,%%ymm2,%%ymm2          \n"

            "vpmaddubsw  %%ymm7,%%ymm0,%%ymm1          \n"
            "vpmaddubsw  %%ymm7,%%ymm2,%%ymm3          \n"
            "vpmaddubsw  %%ymm6,%%ymm0,%%ymm0          \n"
            "vpmaddubsw  %%ymm6,%%ymm2,%%ymm2          \n"
            "vphaddw     %%ymm3,%%ymm1,%%ymm1          \n"
            "vphaddw     %%ymm2,%%ymm0,%%ymm0          \n"
            "vpsraw      $0x8,%%ymm1,%%ymm1            \n"
            "vpsraw      $0x8,%%ymm0,%%ymm0            \n"
            "vpacksswb   %%ymm0,%%ymm1,%%ymm0          \n"
            "vpermq      $0xd8,%%ymm0,%%ymm0           \n"
            "vpshufb     %8,%%ymm0,%%ymm0              \n"
            "vpaddb      %%ymm5,%%ymm0,%%ymm0          \n"

            "vextractf128 $0x0,%%ymm0,(%1)             \n"
            "vextractf128 $0x1,%%ymm0,0x0(%1,%2,1)     \n"
            "lea         0x10(%1),%1                   \n"
            "sub         $0x20,%3                      \n"
            "jg          1b                            \n"
            "vzeroupper                                \n"
            : "+r"(src_abgr),                    // %0
    "+r"(dst_u),                       // %1
    "+r"(dst_v),                       // %2
    "+rm"(width)                       // %3
            : "r"((intptr_t) (src_stride_abgr)),  // %4
    "m"(kAddUV128),                    // %5
    "m"(kABGRToV),                     // %6
    "m"(kABGRToU),                     // %7
    "m"(kShufARGBToUV_AVX)             // %8
            : "memory", "cc", "xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6",
    "xmm7");
}

#endif  // HAS_ABGRTOUVROW_AVX2

void BGRAToYRow_SSSE3(const uint8_t *src_bgra, uint8_t *dst_y, int width) {
    asm volatile(
            "movdqa      %3,%%xmm4                     \n"
            "movdqa      %4,%%xmm5                     \n"
            "movdqa      %5,%%xmm7                     \n"

            LABELALIGN RGBTOY(xmm7)
            : "+r"(src_bgra),  // %0
    "+r"(dst_y),     // %1
    "+r"(width)      // %2
            : "m"(kBGRAToY),   // %3
    "m"(kSub128),    // %4
    "m"(kAddY16)     // %5
            : "memory", "cc", "xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6",
    "xmm7");
}

void BGRAToUVRow_SSSE3(const uint8_t *src_bgra,
                       int src_stride_bgra,
                       uint8_t *dst_u,
                       uint8_t *dst_v,
                       int width) {
    asm volatile(
            "movdqa      %5,%%xmm3                     \n"
            "movdqa      %6,%%xmm4                     \n"
            "movdqa      %7,%%xmm5                     \n"
            "sub         %1,%2                         \n"

            LABELALIGN
            "1:                                        \n"
            "movdqu      (%0),%%xmm0                   \n"
            "movdqu      0x00(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm0                 \n"
            "movdqu      0x10(%0),%%xmm1               \n"
            "movdqu      0x10(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm1                 \n"
            "movdqu      0x20(%0),%%xmm2               \n"
            "movdqu      0x20(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm2                 \n"
            "movdqu      0x30(%0),%%xmm6               \n"
            "movdqu      0x30(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm6                 \n"

            "lea         0x40(%0),%0                   \n"
            "movdqa      %%xmm0,%%xmm7                 \n"
            "shufps      $0x88,%%xmm1,%%xmm0           \n"
            "shufps      $0xdd,%%xmm1,%%xmm7           \n"
            "pavgb       %%xmm7,%%xmm0                 \n"
            "movdqa      %%xmm2,%%xmm7                 \n"
            "shufps      $0x88,%%xmm6,%%xmm2           \n"
            "shufps      $0xdd,%%xmm6,%%xmm7           \n"
            "pavgb       %%xmm7,%%xmm2                 \n"
            "movdqa      %%xmm0,%%xmm1                 \n"
            "movdqa      %%xmm2,%%xmm6                 \n"
            "pmaddubsw   %%xmm4,%%xmm0                 \n"
            "pmaddubsw   %%xmm4,%%xmm2                 \n"
            "pmaddubsw   %%xmm3,%%xmm1                 \n"
            "pmaddubsw   %%xmm3,%%xmm6                 \n"
            "phaddw      %%xmm2,%%xmm0                 \n"
            "phaddw      %%xmm6,%%xmm1                 \n"
            "psraw       $0x8,%%xmm0                   \n"
            "psraw       $0x8,%%xmm1                   \n"
            "packsswb    %%xmm1,%%xmm0                 \n"
            "paddb       %%xmm5,%%xmm0                 \n"
            "movlps      %%xmm0,(%1)                   \n"
            "movhps      %%xmm0,0x00(%1,%2,1)          \n"
            "lea         0x8(%1),%1                    \n"
            "sub         $0x10,%3                      \n"
            "jg          1b                            \n"
            : "+r"(src_bgra),                    // %0
    "+r"(dst_u),                       // %1
    "+r"(dst_v),                       // %2
    "+rm"(width)                       // %3
            : "r"((intptr_t) (src_stride_bgra)),  // %4
    "m"(kBGRAToV),                     // %5
    "m"(kBGRAToU),                     // %6
    "m"(kAddUV128)                     // %7
            : "memory", "cc", "xmm0", "xmm1", "xmm2", "xmm6", "xmm7");
}

void ABGRToYRow_SSSE3(const uint8_t *src_abgr, uint8_t *dst_y, int width) {
    asm volatile(
            "movdqa      %3,%%xmm4                     \n"
            "movdqa      %4,%%xmm5                     \n"
            "movdqa      %5,%%xmm7                     \n"

            LABELALIGN RGBTOY(xmm7)
            : "+r"(src_abgr),  // %0
    "+r"(dst_y),     // %1
    "+r"(width)      // %2
            : "m"(kABGRToY),   // %3
    "m"(kSub128),    // %4
    "m"(kAddY16)     // %5
            : "memory", "cc", "xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5", "xmm6",
    "xmm7");
}

void ABGRToUVRow_SSSE3(const uint8_t *src_abgr,
                       int src_stride_abgr,
                       uint8_t *dst_u,
                       uint8_t *dst_v,
                       int width) {
    asm volatile(
            "movdqa      %5,%%xmm3                     \n"
            "movdqa      %6,%%xmm4                     \n"
            "movdqa      %7,%%xmm5                     \n"
            "sub         %1,%2                         \n"

            LABELALIGN
            "1:                                        \n"
            "movdqu      (%0),%%xmm0                   \n"
            "movdqu      0x00(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm0                 \n"
            "movdqu      0x10(%0),%%xmm1               \n"
            "movdqu      0x10(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm1                 \n"
            "movdqu      0x20(%0),%%xmm2               \n"
            "movdqu      0x20(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm2                 \n"
            "movdqu      0x30(%0),%%xmm6               \n"
            "movdqu      0x30(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm6                 \n"

            "lea         0x40(%0),%0                   \n"
            "movdqa      %%xmm0,%%xmm7                 \n"
            "shufps      $0x88,%%xmm1,%%xmm0           \n"
            "shufps      $0xdd,%%xmm1,%%xmm7           \n"
            "pavgb       %%xmm7,%%xmm0                 \n"
            "movdqa      %%xmm2,%%xmm7                 \n"
            "shufps      $0x88,%%xmm6,%%xmm2           \n"
            "shufps      $0xdd,%%xmm6,%%xmm7           \n"
            "pavgb       %%xmm7,%%xmm2                 \n"
            "movdqa      %%xmm0,%%xmm1                 \n"
            "movdqa      %%xmm2,%%xmm6                 \n"
            "pmaddubsw   %%xmm4,%%xmm0                 \n"
            "pmaddubsw   %%xmm4,%%xmm2                 \n"
            "pmaddubsw   %%xmm3,%%xmm1                 \n"
            "pmaddubsw   %%xmm3,%%xmm6                 \n"
            "phaddw      %%xmm2,%%xmm0                 \n"
            "phaddw      %%xmm6,%%xmm1                 \n"
            "psraw       $0x8,%%xmm0                   \n"
            "psraw       $0x8,%%xmm1                   \n"
            "packsswb    %%xmm1,%%xmm0                 \n"
            "paddb       %%xmm5,%%xmm0                 \n"
            "movlps      %%xmm0,(%1)                   \n"
            "movhps      %%xmm0,0x00(%1,%2,1)          \n"
            "lea         0x8(%1),%1                    \n"
            "sub         $0x10,%3                      \n"
            "jg          1b                            \n"
            : "+r"(src_abgr),                    // %0
    "+r"(dst_u),                       // %1
    "+r"(dst_v),                       // %2
    "+rm"(width)                       // %3
            : "r"((intptr_t) (src_stride_abgr)),  // %4
    "m"(kABGRToV),                     // %5
    "m"(kABGRToU),                     // %6
    "m"(kAddUV128)                     // %7
            : "memory", "cc", "xmm0", "xmm1", "xmm2", "xmm6", "xmm7");
}

void RGBAToUVRow_SSSE3(const uint8_t *src_rgba,
                       int src_stride_rgba,
                       uint8_t *dst_u,
                       uint8_t *dst_v,
                       int width) {
    asm volatile(
            "movdqa      %5,%%xmm3                     \n"
            "movdqa      %6,%%xmm4                     \n"
            "movdqa      %7,%%xmm5                     \n"
            "sub         %1,%2                         \n"

            LABELALIGN
            "1:                                        \n"
            "movdqu      (%0),%%xmm0                   \n"
            "movdqu      0x00(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm0                 \n"
            "movdqu      0x10(%0),%%xmm1               \n"
            "movdqu      0x10(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm1                 \n"
            "movdqu      0x20(%0),%%xmm2               \n"
            "movdqu      0x20(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm2                 \n"
            "movdqu      0x30(%0),%%xmm6               \n"
            "movdqu      0x30(%0,%4,1),%%xmm7          \n"
            "pavgb       %%xmm7,%%xmm6                 \n"

            "lea         0x40(%0),%0                   \n"
            "movdqa      %%xmm0,%%xmm7                 \n"
            "shufps      $0x88,%%xmm1,%%xmm0           \n"
            "shufps      $0xdd,%%xmm1,%%xmm7           \n"
            "pavgb       %%xmm7,%%xmm0                 \n"
            "movdqa      %%xmm2,%%xmm7                 \n"
            "shufps      $0x88,%%xmm6,%%xmm2           \n"
            "shufps      $0xdd,%%xmm6,%%xmm7           \n"
            "pavgb       %%xmm7,%%xmm2                 \n"
            "movdqa      %%xmm0,%%xmm1                 \n"
            "movdqa      %%xmm2,%%xmm6                 \n"
            "pmaddubsw   %%xmm4,%%xmm0                 \n"
            "pmaddubsw   %%xmm4,%%xmm2                 \n"
            "pmaddubsw   %%xmm3,%%xmm1                 \n"
            "pmaddubsw   %%xmm3,%%xmm6                 \n"
            "phaddw      %%xmm2,%%xmm0                 \n"
            "phaddw      %%xmm6,%%xmm1                 \n"
            "psraw       $0x8,%%xmm0                   \n"
            "psraw       $0x8,%%xmm1                   \n"
            "packsswb    %%xmm1,%%xmm0                 \n"
            "paddb       %%xmm5,%%xmm0                 \n"
            "movlps      %%xmm0,(%1)                   \n"
            "movhps      %%xmm0,0x00(%1,%2,1)          \n"
            "lea         0x8(%1),%1                    \n"
            "sub         $0x10,%3                      \n"
            "jg          1b                            \n"
            : "+r"(src_rgba),                    // %0
    "+r"(dst_u),                       // %1
    "+r"(dst_v),                       // %2
    "+rm"(width)                       // %3
            : "r"((intptr_t) (src_stride_rgba)),  // %4
    "m"(kRGBAToV),                     // %5
    "m"(kRGBAToU),                     // %6
    "m"(kAddUV128)                     // %7
            : "memory", "cc", "xmm0", "xmm1", "xmm2", "xmm6", "xmm7");
}

#ifdef HAS_MIRRORROW_SSSE3
// Shuffle table for reversing the bytes.
static const uvec8 kShuffleMirror = {15u, 14u, 13u, 12u, 11u, 10u, 9u, 8u,
                                     7u, 6u, 5u, 4u, 3u, 2u, 1u, 0u};

void MirrorRow_SSSE3(const uint8_t *src, uint8_t *dst, int width) {
    intptr_t temp_width = (intptr_t) (width);
    asm volatile(

            "movdqa      %3,%%xmm5                     \n"

            LABELALIGN
            "1:                                        \n"
            "movdqu      -0x10(%0,%2,1),%%xmm0         \n"
            "pshufb      %%xmm5,%%xmm0                 \n"
            "movdqu      %%xmm0,(%1)                   \n"
            "lea         0x10(%1),%1                   \n"
            "sub         $0x10,%2                      \n"
            "jg          1b                            \n"
            : "+r"(src),           // %0
    "+r"(dst),           // %1
    "+r"(temp_width)     // %2
            : "m"(kShuffleMirror)  // %3
            : "memory", "cc", "xmm0", "xmm5");
}

#endif  // HAS_MIRRORROW_SSSE3

#ifdef HAS_MIRRORROW_AVX2

void MirrorRow_AVX2(const uint8_t *src, uint8_t *dst, int width) {
    intptr_t temp_width = (intptr_t) (width);
    asm volatile(

            "vbroadcastf128 %3,%%ymm5                  \n"

            LABELALIGN
            "1:                                        \n"
            "vmovdqu     -0x20(%0,%2,1),%%ymm0         \n"
            "vpshufb     %%ymm5,%%ymm0,%%ymm0          \n"
            "vpermq      $0x4e,%%ymm0,%%ymm0           \n"
            "vmovdqu     %%ymm0,(%1)                   \n"
            "lea         0x20(%1),%1                   \n"
            "sub         $0x20,%2                      \n"
            "jg          1b                            \n"
            "vzeroupper                                \n"
            : "+r"(src),           // %0
    "+r"(dst),           // %1
    "+r"(temp_width)     // %2
            : "m"(kShuffleMirror)  // %3
            : "memory", "cc", "xmm0", "xmm5");
}

#endif  // HAS_MIRRORROW_AVX2

#ifdef HAS_MIRRORUVROW_SSSE3
// Shuffle table for reversing the UV.
static const uvec8 kShuffleMirrorUV = {14u, 15u, 12u, 13u, 10u, 11u, 8u, 9u,
                                       6u, 7u, 4u, 5u, 2u, 3u, 0u, 1u};

void MirrorUVRow_SSSE3(const uint8_t *src_uv, uint8_t *dst_uv, int width) {
    intptr_t temp_width = (intptr_t) (width);
    asm volatile(

            "movdqa      %3,%%xmm5                     \n"

            LABELALIGN
            "1:                                        \n"
            "movdqu      -0x10(%0,%2,2),%%xmm0         \n"
            "pshufb      %%xmm5,%%xmm0                 \n"
            "movdqu      %%xmm0,(%1)                   \n"
            "lea         0x10(%1),%1                   \n"
            "sub         $0x8,%2                       \n"
            "jg          1b                            \n"
            : "+r"(src_uv),          // %0
    "+r"(dst_uv),          // %1
    "+r"(temp_width)       // %2
            : "m"(kShuffleMirrorUV)  // %3
            : "memory", "cc", "xmm0", "xmm5");
}

#endif  // HAS_MIRRORUVROW_SSSE3

#ifdef HAS_MIRRORUVROW_AVX2

void MirrorUVRow_AVX2(const uint8_t *src_uv, uint8_t *dst_uv, int width) {
    intptr_t temp_width = (intptr_t) (width);
    asm volatile(

            "vbroadcastf128 %3,%%ymm5                  \n"

            LABELALIGN
            "1:                                        \n"
            "vmovdqu     -0x20(%0,%2,2),%%ymm0         \n"
            "vpshufb     %%ymm5,%%ymm0,%%ymm0          \n"
            "vpermq      $0x4e,%%ymm0,%%ymm0           \n"
            "vmovdqu     %%ymm0,(%1)                   \n"
            "lea         0x20(%1),%1                   \n"
            "sub         $0x10,%2                      \n"
            "jg          1b                            \n"
            "vzeroupper                                \n"
            : "+r"(src_uv),          // %0
    "+r"(dst_uv),          // %1
    "+r"(temp_width)       // %2
            : "m"(kShuffleMirrorUV)  // %3
            : "memory", "cc", "xmm0", "xmm5");
}

#endif  // HAS_MIRRORUVROW_AVX2

#ifdef HAS_MIRRORSPLITUVROW_SSSE3
// Shuffle table for reversing the bytes of UV channels.
static const uvec8 kShuffleMirrorSplitUV = {14u, 12u, 10u, 8u, 6u, 4u, 2u, 0u,
                                            15u, 13u, 11u, 9u, 7u, 5u, 3u, 1u};

void MirrorSplitUVRow_SSSE3(const uint8_t *src,
                            uint8_t *dst_u,
                            uint8_t *dst_v,
                            int width) {
    intptr_t temp_width = (intptr_t) (width);
    asm volatile(
            "movdqa      %4,%%xmm1                     \n"
            "lea         -0x10(%0,%3,2),%0             \n"
            "sub         %1,%2                         \n"

            LABELALIGN
            "1:                                        \n"
            "movdqu      (%0),%%xmm0                   \n"
            "lea         -0x10(%0),%0                  \n"
            "pshufb      %%xmm1,%%xmm0                 \n"
            "movlpd      %%xmm0,(%1)                   \n"
            "movhpd      %%xmm0,0x00(%1,%2,1)          \n"
            "lea         0x8(%1),%1                    \n"
            "sub         $8,%3                         \n"
            "jg          1b                            \n"
            : "+r"(src),                  // %0
    "+r"(dst_u),                // %1
    "+r"(dst_v),                // %2
    "+r"(temp_width)            // %3
            : "m"(kShuffleMirrorSplitUV)  // %4
            : "memory", "cc", "xmm0", "xmm1");
}

#endif  // HAS_MIRRORSPLITUVROW_SSSE3

#ifdef HAS_ARGBMIRRORROW_SSE2

void ARGBMirrorRow_SSE2(const uint8_t *src, uint8_t *dst, int width) {
    intptr_t temp_width = (intptr_t) (width);
    asm volatile(

            "lea         -0x10(%0,%2,4),%0             \n"

            LABELALIGN
            "1:                                        \n"
            "movdqu      (%0),%%xmm0                   \n"
            "pshufd      $0x1b,%%xmm0,%%xmm0           \n"
            "lea         -0x10(%0),%0                  \n"
            "movdqu      %%xmm0,(%1)                   \n"
            "lea         0x10(%1),%1                   \n"
            "sub         $0x4,%2                       \n"
            "jg          1b                            \n"
            : "+r"(src),        // %0
    "+r"(dst),        // %1
    "+r"(temp_width)  // %2
            :
            : "memory", "cc", "xmm0");
}

#endif  // HAS_ARGBMIRRORROW_SSE2

#ifdef HAS_ARGBMIRRORROW_AVX2
// Shuffle table for reversing the bytes.
static const ulvec32 kARGBShuffleMirror_AVX2 = {7u, 6u, 5u, 4u, 3u, 2u, 1u, 0u};

void ARGBMirrorRow_AVX2(const uint8_t *src, uint8_t *dst, int width) {
    intptr_t temp_width = (intptr_t) (width);
    asm volatile(

            "vmovdqu     %3,%%ymm5                     \n"

            LABELALIGN
            "1:                                        \n"
            "vpermd      -0x20(%0,%2,4),%%ymm5,%%ymm0  \n"
            "vmovdqu     %%ymm0,(%1)                   \n"
            "lea         0x20(%1),%1                   \n"
            "sub         $0x8,%2                       \n"
            "jg          1b                            \n"
            "vzeroupper                                \n"
            : "+r"(src),                    // %0
    "+r"(dst),                    // %1
    "+r"(temp_width)              // %2
            : "m"(kARGBShuffleMirror_AVX2)  // %3
            : "memory", "cc", "xmm0", "xmm5");
}

#endif  // HAS_ARGBMIRRORROW_AVX2


#ifdef HAS_COPYROW_SSE2

void CopyRow_SSE2(const uint8_t *src, uint8_t *dst, int width) {
    asm volatile(
            "test        $0xf,%0                       \n"
            "jne         2f                            \n"
            "test        $0xf,%1                       \n"
            "jne         2f                            \n"

            LABELALIGN
            "1:                                        \n"
            "movdqa      (%0),%%xmm0                   \n"
            "movdqa      0x10(%0),%%xmm1               \n"
            "lea         0x20(%0),%0                   \n"
            "movdqa      %%xmm0,(%1)                   \n"
            "movdqa      %%xmm1,0x10(%1)               \n"
            "lea         0x20(%1),%1                   \n"
            "sub         $0x20,%2                      \n"
            "jg          1b                            \n"
            "jmp         9f                            \n"

            LABELALIGN
            "2:                                        \n"
            "movdqu      (%0),%%xmm0                   \n"
            "movdqu      0x10(%0),%%xmm1               \n"
            "lea         0x20(%0),%0                   \n"
            "movdqu      %%xmm0,(%1)                   \n"
            "movdqu      %%xmm1,0x10(%1)               \n"
            "lea         0x20(%1),%1                   \n"
            "sub         $0x20,%2                      \n"
            "jg          2b                            \n"

            LABELALIGN "9:                                        \n"
            : "+r"(src),   // %0
    "+r"(dst),   // %1
    "+r"(width)  // %2
            :
            : "memory", "cc", "xmm0", "xmm1");
}

#endif  // HAS_COPYROW_SSE2

#ifdef HAS_COPYROW_AVX

void CopyRow_AVX(const uint8_t *src, uint8_t *dst, int width) {
    asm volatile(

            LABELALIGN
            "1:                                        \n"
            "vmovdqu     (%0),%%ymm0                   \n"
            "vmovdqu     0x20(%0),%%ymm1               \n"
            "lea         0x40(%0),%0                   \n"
            "vmovdqu     %%ymm0,(%1)                   \n"
            "vmovdqu     %%ymm1,0x20(%1)               \n"
            "lea         0x40(%1),%1                   \n"
            "sub         $0x40,%2                      \n"
            "jg          1b                            \n"
            "vzeroupper                                \n"
            : "+r"(src),   // %0
    "+r"(dst),   // %1
    "+r"(width)  // %2
            :
            : "memory", "cc", "xmm0", "xmm1");
}

#endif  // HAS_COPYROW_AVX

#ifdef HAS_COPYROW_ERMS

// Multiple of 1.
void CopyRow_ERMS(const uint8_t *src, uint8_t *dst, int width) {
    size_t width_tmp = (size_t) (width);
    asm volatile(

            "rep         movsb                         \n"
            : "+S"(src),       // %0
    "+D"(dst),       // %1
    "+c"(width_tmp)  // %2
            :
            : "memory", "cc");
}

#endif  // HAS_COPYROW_ERMS

#ifdef HAS_INTERPOLATEROW_SSSE3

// Bilinear filter 16x2 -> 16x1
void InterpolateRow_SSSE3(uint8_t *dst_ptr,
                          const uint8_t *src_ptr,
                          ptrdiff_t src_stride,
                          int width,
                          int source_y_fraction) {
    asm volatile(
            "sub         %1,%0                         \n"
            "cmp         $0x0,%3                       \n"
            "je          100f                          \n"
            "cmp         $0x80,%3                      \n"
            "je          50f                           \n"

            "movd        %3,%%xmm0                     \n"
            "neg         %3                            \n"
            "add         $0x100,%3                     \n"
            "movd        %3,%%xmm5                     \n"
            "punpcklbw   %%xmm0,%%xmm5                 \n"
            "punpcklwd   %%xmm5,%%xmm5                 \n"
            "pshufd      $0x0,%%xmm5,%%xmm5            \n"
            "mov         $0x80808080,%%eax             \n"
            "movd        %%eax,%%xmm4                  \n"
            "pshufd      $0x0,%%xmm4,%%xmm4            \n"

            // General purpose row blend.
            LABELALIGN
            "1:                                        \n"
            "movdqu      (%1),%%xmm0                   \n"
            "movdqu      0x00(%1,%4,1),%%xmm2          \n"
            "movdqa      %%xmm0,%%xmm1                 \n"
            "punpcklbw   %%xmm2,%%xmm0                 \n"
            "punpckhbw   %%xmm2,%%xmm1                 \n"
            "psubb       %%xmm4,%%xmm0                 \n"
            "psubb       %%xmm4,%%xmm1                 \n"
            "movdqa      %%xmm5,%%xmm2                 \n"
            "movdqa      %%xmm5,%%xmm3                 \n"
            "pmaddubsw   %%xmm0,%%xmm2                 \n"
            "pmaddubsw   %%xmm1,%%xmm3                 \n"
            "paddw       %%xmm4,%%xmm2                 \n"
            "paddw       %%xmm4,%%xmm3                 \n"
            "psrlw       $0x8,%%xmm2                   \n"
            "psrlw       $0x8,%%xmm3                   \n"
            "packuswb    %%xmm3,%%xmm2                 \n"
            "movdqu      %%xmm2,0x00(%1,%0,1)          \n"
            "lea         0x10(%1),%1                   \n"
            "sub         $0x10,%2                      \n"
            "jg          1b                            \n"
            "jmp         99f                           \n"

            // Blend 50 / 50.
            LABELALIGN
            "50:                                       \n"
            "movdqu      (%1),%%xmm0                   \n"
            "movdqu      0x00(%1,%4,1),%%xmm1          \n"
            "pavgb       %%xmm1,%%xmm0                 \n"
            "movdqu      %%xmm0,0x00(%1,%0,1)          \n"
            "lea         0x10(%1),%1                   \n"
            "sub         $0x10,%2                      \n"
            "jg          50b                           \n"
            "jmp         99f                           \n"

            // Blend 100 / 0 - Copy row unchanged.
            LABELALIGN
            "100:                                      \n"
            "movdqu      (%1),%%xmm0                   \n"
            "movdqu      %%xmm0,0x00(%1,%0,1)          \n"
            "lea         0x10(%1),%1                   \n"
            "sub         $0x10,%2                      \n"
            "jg          100b                          \n"

            "99:                                       \n"
            : "+r"(dst_ptr),               // %0
    "+r"(src_ptr),               // %1
    "+rm"(width),                // %2
    "+r"(source_y_fraction)      // %3
            : "r"((intptr_t) (src_stride))  // %4
            : "memory", "cc", "eax", "xmm0", "xmm1", "xmm2", "xmm3", "xmm4", "xmm5");
}

#endif  // HAS_INTERPOLATEROW_SSSE3

#ifdef HAS_INTERPOLATEROW_AVX2

// Bilinear filter 32x2 -> 32x1
void InterpolateRow_AVX2(uint8_t *dst_ptr,
                         const uint8_t *src_ptr,
                         ptrdiff_t src_stride,
                         int width,
                         int source_y_fraction) {
    asm volatile(
            "sub         %1,%0                         \n"
            "cmp         $0x0,%3                       \n"
            "je          100f                          \n"
            "cmp         $0x80,%3                      \n"
            "je          50f                           \n"

            "vmovd       %3,%%xmm0                     \n"
            "neg         %3                            \n"
            "add         $0x100,%3                     \n"
            "vmovd       %3,%%xmm5                     \n"
            "vpunpcklbw  %%xmm0,%%xmm5,%%xmm5          \n"
            "vpunpcklwd  %%xmm5,%%xmm5,%%xmm5          \n"
            "vbroadcastss %%xmm5,%%ymm5                \n"
            "mov         $0x80808080,%%eax             \n"
            "vmovd       %%eax,%%xmm4                  \n"
            "vbroadcastss %%xmm4,%%ymm4                \n"

            // General purpose row blend.
            LABELALIGN
            "1:                                        \n"
            "vmovdqu     (%1),%%ymm0                   \n"
            "vmovdqu     0x00(%1,%4,1),%%ymm2          \n"
            "vpunpckhbw  %%ymm2,%%ymm0,%%ymm1          \n"
            "vpunpcklbw  %%ymm2,%%ymm0,%%ymm0          \n"
            "vpsubb      %%ymm4,%%ymm1,%%ymm1          \n"
            "vpsubb      %%ymm4,%%ymm0,%%ymm0          \n"
            "vpmaddubsw  %%ymm1,%%ymm5,%%ymm1          \n"
            "vpmaddubsw  %%ymm0,%%ymm5,%%ymm0          \n"
            "vpaddw      %%ymm4,%%ymm1,%%ymm1          \n"
            "vpaddw      %%ymm4,%%ymm0,%%ymm0          \n"
            "vpsrlw      $0x8,%%ymm1,%%ymm1            \n"
            "vpsrlw      $0x8,%%ymm0,%%ymm0            \n"
            "vpackuswb   %%ymm1,%%ymm0,%%ymm0          \n"
            "vmovdqu     %%ymm0,0x00(%1,%0,1)          \n"
            "lea         0x20(%1),%1                   \n"
            "sub         $0x20,%2                      \n"
            "jg          1b                            \n"
            "jmp         99f                           \n"

            // Blend 50 / 50.
            LABELALIGN
            "50:                                       \n"
            "vmovdqu     (%1),%%ymm0                   \n"
            "vpavgb      0x00(%1,%4,1),%%ymm0,%%ymm0   \n"
            "vmovdqu     %%ymm0,0x00(%1,%0,1)          \n"
            "lea         0x20(%1),%1                   \n"
            "sub         $0x20,%2                      \n"
            "jg          50b                           \n"
            "jmp         99f                           \n"

            // Blend 100 / 0 - Copy row unchanged.
            LABELALIGN
            "100:                                      \n"
            "vmovdqu     (%1),%%ymm0                   \n"
            "vmovdqu     %%ymm0,0x00(%1,%0,1)          \n"
            "lea         0x20(%1),%1                   \n"
            "sub         $0x20,%2                      \n"
            "jg          100b                          \n"

            "99:                                       \n"
            "vzeroupper                                \n"
            : "+r"(dst_ptr),               // %0
    "+r"(src_ptr),               // %1
    "+r"(width),                 // %2
    "+r"(source_y_fraction)      // %3
            : "r"((intptr_t) (src_stride))  // %4
            : "memory", "cc", "eax", "xmm0", "xmm1", "xmm2", "xmm4", "xmm5");
}

#endif  // HAS_INTERPOLATEROW_AVX2

#endif  // defined(__x86_64__) || defined(__i386__)
