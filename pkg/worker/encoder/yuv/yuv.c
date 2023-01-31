#include "yuv.h"

#define Y601_STUDIO 1

// BT.601 STUDIO

#ifdef Y601_STUDIO
// 66*R+129*G+25*B
static __inline int Y(uint8_t *__restrict rgb) {
    int R = *rgb;
    int G = *(rgb+1);
    int B = *(rgb+2);
    return (66*R+129*G+25*B+128)>>8;
}

// 112*B-38*R-74G
static __inline int U(uint8_t *__restrict rgb) {
    int R = *rgb;
    int G = *(rgb+1);
    int B = *(rgb+2);
    return (-38*R-74*G+112*B+128) >> 8;
}

// 112*R-94*G-18*B
static __inline int V(uint8_t *__restrict rgb) {
    int R = 56**(rgb);
    int G = 47**(rgb+1);
    int B =    *(rgb+2);
    return (R-G-(B+(B<<3))+64) >> 7;
}

static const int Y_MIN = 16;

#else

// BT.601 FULL

// 77*R+150*G+29*B
static __inline int Y(uint8_t *rgb) {
    int R =  77**(rgb);
    int G = 150**(rgb+1);
    int B =  29**(rgb+2);
    return (R+G+B+128) >> 8;
}

// 127*B-43*R-84*G
static __inline int U(uint8_t *rgb) {
    int R =  43**(rgb);
    int G =  84**(rgb+1);
    int B = 127**(rgb+2);
    return (-R-G+B+128) >> 8;
}

// 127*R-106*G-21*B
static __inline int V(uint8_t *rgb) {
    int R =  127**rgb;
    int G = -106**(rgb+1);
    int B =  -21**(rgb+2);
    return (G+B+R+128) >> 8;
}

static const int Y_MIN = 0;
#endif

static __inline void _y(uint8_t *__restrict p, uint8_t *__restrict y, int size) {
    do {
        *y++ = Y(p) + Y_MIN;
        p += 4;
    } while (--size);
}

// It will take an average color from the 2x2 pixel group for chroma values.
// X   X   X   X
//   O       O
// X   X   X   X
static __inline void _4uv(uint8_t * __restrict p, uint8_t * __restrict u, uint8_t * __restrict v, const int w, const int h) {
    uint8_t *p2, *p3, *p4;
    const int row = w << 2;
    const int next = 4;

    int x = w, y = h, sumU = 0, sumV = 0;
    while (y > 0) {
        while (x > 0) {
            // xx..
            // ....
            p2 = p+next;
            sumU = U(p) + U(p2);
            sumV = V(p) + V(p2);
            // ....
            // xx..
            p3 = p+row;
            p4 = p3+next;
            sumU += U(p3) + U(p4);
            sumV += V(p3) + V(p4);
            *u++ = 128 + (sumU >> 2);
            *v++ = 128 + (sumV >> 2);
            // ..x.
            p += 8;
            x -= 2;
        }
        p += row;
        y -= 2;
        x = w;
    }
}

// Converts RGBA image to YUV (I420) with BT.601 studio color range.
void rgbaToYuv(void *__restrict destination, void *__restrict source, const int w, const int h) {
    const int image_size = w * h;
    uint8_t *src = source;
    uint8_t *dst_y = destination;
    uint8_t *dst_u = destination + image_size;
    uint8_t *dst_v = destination + image_size + image_size / 4;
    _y(src, dst_y, image_size);
    src = source;
    _4uv(source, dst_u, dst_v, w, h);
}

void luma(void *__restrict destination, void *__restrict source, const int pos, const int w, const int h) {
    uint8_t *rgba = source + 4 * pos;
    uint8_t *dst = destination + pos;
    _y(rgba, dst, w*h);
}

void chroma(void *__restrict dst, void *__restrict source, const int pos, const int deu, const int dev, const int w, const int h) {
    uint8_t *src = source + 4 * pos;
    uint8_t *dst_u = dst + deu + pos / 4;
    uint8_t *dst_v = dst + dev + pos / 4;
    _4uv(src, dst_u, dst_v, w, h);
}
