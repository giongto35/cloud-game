#include "canvas.h"

__inline xy rotate(int t, int x, int y, int w, int h) {
    xy p = {x, y};
    switch (t) {
        // 90° CCW or 270° CW
        case A90:
            p.x = y;
            p.y = w - 1 - x;
            break;
        // 180° CCW
        case A180:
            p.x = w - 1 - x;
            p.y = h - 1 - y;
            break;
        // 270° CCW or 90° CW
        case A270:
            p.x = h - 1 - y;
            p.y = x;
            break;
        // flip Y
        case F180:
            //p.x = x;
            p.y = h - 1 - y;
            break;
    }
    return p;
}

__inline uint32_t _565(uint32_t x) {
    return ((x >> 8 & 0xf8) | ((x >> 3 & 0xfc) << 8) | ((x << 3 & 0xfc) << 16)); // | 0xff000000
}

__inline uint32_t _8888rev(uint32_t px) {
    return (((px >> 16) & 0xff) | (px & 0xff00) | ((px << 16) & 0xff0000)); // | 0xff000000
}

void RGBA(int pix, uint32_t *__restrict dst, const void *__restrict source, int y, int h, int w, int hh, int dw, int pad, int rot) {
    int x;
    xy rxy;
    const uint16_t *src16;
    const uint32_t *src32;

    switch (pix) {
        //case BIT_SHORT5551:
        //    break;
        case BIT_INT_8888REV:
            src32 = (const uint32_t *)source;
            int pad32 = pad >> 2;
            if (rot == NO_ROT) {
                for (; y < h; ++y) {
                    for (x = 0; x < w; ++x) {
                        *dst++ = _8888rev(*src32++);
                    }
                    src32 += pad32;
                }
            } else {
                for (; y < h; ++y) {
                    for (x = 0; x < w; ++x) {
                        rxy = rotate(rot, x, y, w, hh);
                        dst[rxy.x+rxy.y*dw] = _8888rev(*src32++);
                    }
                    src32 += pad32;
                }
            }
            break;
        case BIT_SHORT565:
            src16 = (const uint16_t *)source;
            int pad16 = pad >> 1;
            if (rot == NO_ROT) {
                for (; y < h; ++y) {
                    for (x = 0; x < w; ++x) {
                        *dst++ = _565(*src16++);
                    }
                    src16 += pad16;
                }
            } else {
                for (; y < h; ++y) {
                    for (x = 0; x < w; ++x) {
                        rxy = rotate(rot, x, y, w, hh);
                        dst[rxy.x+rxy.y*dw] = _565(*src16++);
                    }
                    src16 += pad16;
                }
            }
            break;
    }
}
