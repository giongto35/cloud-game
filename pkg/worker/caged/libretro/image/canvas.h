#ifndef CANVAS_H__
#define CANVAS_H__

#include <stdint.h>

#define BIT_SHORT5551 0
#define BIT_INT_8888REV 1
#define BIT_SHORT565 2

#define NO_ROT 0
#define	A90 1
#define	A180 2
#define	A270 3
#define	F180 4

// Rotate90 is 90° CCW or 270° CW.
#define r90_x(x, y, w, h) ( y )
#define r90_y(x, y, w, h) ( (w - 1) - x )

// Rotate180 is 180° CCW.
#define r180_x(x, y, w, h) ( (w - 1) - x )
#define r180_y(x, y, w, h) ( (h - 1) - y )

// Rotate270 is 270° CCW or 90° CW.
#define r270_x(x, y, w, h) ( (h - 1) - y )
#define r270_y(x, y, w, h) ( x )

// Flip Y
#define fy180_x(x, y, w, h) ( x )
#define fy180_y(x, y, w, h) ( (h - 1) - y )

typedef struct XY {
    int x, y;
} xy;

xy rotate(int t, int x, int y, int w, int h);

void RGBA(int pix, uint32_t *dst, void *source, int y, int h, int w, int hh, int dw, int pad, int rot);

uint32_t _565(uint32_t x);
uint32_t _8888rev(uint32_t px);

#endif
