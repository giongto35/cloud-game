#include "yuv.h"

// BT.601 STUDIO

// 66*R+129*G+25*B
static __inline uint8_t Y601_STUDIO(rgb *rgb) {
    int R = rgb->r;
    R += R + (R << 6);
    int G = rgb->g;
    G += G << 7;
    int B = 25*rgb->b;
    return (R+G+B+128) >> 8;
}

// 112*B-38*R-74G
static __inline int U601_STUDIO(rgb *rgb) {
    int R = 19*rgb->r;
    int G = 39*rgb->g;
    int B = 56*rgb->b;
    return (-R-G+B+64) >> 7;
}

// 112*R-94*G-18*B
static __inline int V601_STUDIO(rgb *rgb) {
    int R = 56*rgb->r;
    int G = 47*rgb->g;
    int B = rgb->b;
    B += B << 3;
    return (R-G-B+64) >> 7;
}

// BT.601 FULL

// 77*R+150*G+29*B
static __inline uint8_t Y601_FULL(rgb *rgb) {
    int R =  77*rgb->r;
    int G = 150*rgb->g;
    int B =  29*rgb->b;
    return (R+G+B+128) >> 8;
}

// 127*B-43*R-84*G
static __inline int U601_FULL(rgb *rgb) {
    int R =  43*rgb->r;
    int G =  84*rgb->g;
    int B = 127*rgb->b;
    return (-R-G+B+128) >> 8;
}

// 127*R-106*G-21*B
static __inline int V601_FULL(rgb *rgb) {
    int R = 127*rgb->r;
    int G = 106*rgb->g;
    int B = 21*rgb->b;
    return (R-G-B+128) >> 8;
}

//static const int Y601_STUDIO_MIN = 16;
static const int Y601_FULL_MIN = 0;

#define Y      Y601_FULL//Y601_studio
#define U      U601_FULL//U601_studio
#define V      V601_FULL//V601_studio
#define Y_MIN  Y601_FULL_MIN//Y_STUDIO_MIN

static __inline void _y(uint8_t *p, uint8_t *y, int size) {
    rgb *px;
    do {
        px = (rgb *)(p);
        *y++ = Y_MIN + Y(px);
        p += 4;
    } while (--size);
}

static __inline void _1uv(uint8_t *p, uint8_t *u, uint8_t *v, int w, int h) {
    rgb *p1;
    int x, y;

    const int row = w << 2;
    for (y = 0; y < h; y += 2) {
        for (x = 0; x < w; x += 2) {
            p1 = (rgb *)(p);
            *u++ = 128 + U(p1);
            *v++ = 128 + V(p1);
            p += 8;
        }
        p += row;
    }
}

static __inline void _4uv(uint8_t *p, uint8_t *u, uint8_t *v, int w, int h) {
    rgb *p1, *p2, *p3, *p4;
    int x, y;
    const int row = w << 2;

    int sumU = 0;
    int sumV = 0;
    int pt = 0;

    for (y = 0; y < h; y += 2) {
        for (x = 0; x < w; x += 2) {
            pt = 4;
            // xx..
            // ....
            p1 = (rgb *)(p);
            p2 = (rgb *)(p+pt);
            sumU = U(p1) + U(p2);
            sumV = V(p1) + V(p2);
            // ....
            // xx..
            pt += row;
            p4 = (rgb *)(p+pt);
            pt -= 4;
            p3 = (rgb *)(p+pt);
            sumU += U(p3) + U(p4);
            sumV += V(p3) + V(p4);
            // ..x.
            p += 8;
            *u++ = 128 + (sumU >> 2);
            *v++ = 128 + (sumV >> 2);
        }
        p += row;
    }
}

// Converts RGBA image to YUV (I420) with BT.601 studio color range.
void rgbaToYuv(void *destination, void *source, int w, int h, chromaPos chroma) {
    const int image_size = w * h;
    uint8_t *src = source;
    uint8_t *dst_y = destination;
    uint8_t *dst_u = destination + image_size;
    uint8_t *dst_v = destination + image_size + image_size / 4;

    _y(src, dst_y, image_size);
    src = source;
    if (chroma == BETWEEN_FOUR) {
        _4uv(src, dst_u, dst_v, w, h);
    } else {
        _1uv(src, dst_u, dst_v, w, h);
    }
}

void luma(void *destination, void *source, int pos, int w, int h) {
    uint8_t *rgba = source + 4 * pos;
    uint8_t *dst = destination + pos;
    _y(rgba, dst, w*h);
}

void chroma(void *dst, void *source, int pos, int deu, int dev, int w, int h, chromaPos chroma) {
    uint8_t *src = source + 4 * pos;
    uint8_t *dst_u = dst + deu + pos / 4;
    uint8_t *dst_v = dst + dev + pos / 4;

    if (chroma == BETWEEN_FOUR) {
        _4uv(src, dst_u, dst_v, w, h);
    } else {
        _1uv(src, dst_u, dst_v, w, h);
    }
}
