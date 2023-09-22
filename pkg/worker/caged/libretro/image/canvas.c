#include "canvas.h"

__inline int rot_x(int t, int x, int y, int w, int h) {
    switch (t) {
        case 1:
            return r90_x(x,y,w,h);
            break;
        case 2:
            return r180_x(x,y,w,h);
            break;
        case 3:
            return r270_x(x,y,w,h);
            break;
        case 4:
            return fy180_x(x,y,w,h);
            break;
    }
    return x;
}

__inline int rot_y(int t, int x, int y, int w, int h) {
    switch (t) {
        case 1:
            return r90_y(x,y,w,h);
            break;
        case 2:
            return r180_y(x,y,w,h);
            break;
        case 3:
            return r270_y(x,y,w,h);
            break;
        case 4:
            return fy180_y(x,y,w,h);
            break;
    }
  return y;
 }


void RGBA(int pix, void *destination, void *source, int yy, int yn, int xw, int xh, int dw, int pad, int rot) {
    switch (pix) {
        case BIT_SHORT5551:
            break;
        case BIT_INT_8888REV:
            if (rot == 0) {
                i8888(destination, source, yy, yn, xw, pad);
            } else {
                i8888r(destination, source, yy, yn, xw, xh, dw, pad, rot);
            }
            break;
        case BIT_SHORT565:
            if (rot == 0) {
                i565(destination, source, yy, yn, xw, pad);
            } else {
                i565r(destination, source, yy, yn, xw, xh, dw, pad, rot);
            }
            break;
    }
}

void i565(void *destination, void *source, int yy, int yn, int xw, int pad) {
    uint8_t *src = source; // must be in bytes because of possible padding in bytes
    uint32_t *dst = destination;

    int y, x;
    uint32_t px;

    for (y = yy; y < yn; ++y) {
        for (x = 0; x < xw; ++x) {
            px = *(uint16_t *)src;
            src += 2;
            *dst++ = _565(px);
        }
        src += pad;
    }
}

void i565r(void *destination, void *source, int yy, int yn, int xw, int xh, int dw, int pad, int rot) {
    uint8_t *src = source;
    uint32_t *dst = destination;

    uint32_t px;

    int x, y, dx, dy;

    for (y = yy; y < yn; ++y) {
        for (x = 0; x < xw; ++x) {
            px = *(uint16_t *)src;
            src += 2;

            dx = rot_x(rot, x, y, xw, xh);
            dy = rot_y(rot, x, y, xw, xh);

            dst[dx+dy*dw] = _565(px);
        }
        src += pad;
    }
}

void i8888r(void *destination, void *source, int yy, int yn, int xw, int xh, int dw, int pad, int rot) {
    uint8_t *src = source;
    uint32_t *dst = destination;

    int y, x;
    uint32_t px;

    int dx, dy;

    for (y = yy; y < yn; ++y) {
        for (x = 0; x < xw; ++x) {
            px = *(uint32_t *)src;

            dx = rot_x(rot, x, y, xw, xh);
            dy = rot_y(rot, x, y, xw, xh);

            dst[dx+dy*dw] = _8888rev(px);
            src += 4;
        }
        src += pad;
    }
}

void i8888(void *destination, void *source, int yy, int yn, int xw, int pad) {
    uint8_t *src = source; // must be in bytes because of possible padding in bytes
    uint32_t *dst = destination;

    int y, x;
    uint32_t px;

    for (y = yy; y < yn; ++y) {
        for (x = 0; x < xw; ++x) {
            px = *(uint32_t *)src;
            src += 4;
            *dst++ = _8888rev(px);
        }
        src += pad;
    }
}

uint32_t px8888rev(uint32_t px) {
    return _8888rev(px);
}
