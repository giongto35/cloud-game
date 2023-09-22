#ifndef CANVAS_H__
#define CANVAS_H__

#include <stdint.h>

#define BIT_SHORT5551 0
#define BIT_INT_8888REV 1
#define BIT_SHORT565 2

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

int rot_x(int t, int x, int y, int w, int h);
int rot_y(int t, int x, int y, int w, int h);

#define _565(x) ((x >> 8 & 0xf8) | ((x >> 3 & 0xfc) << 8) | ((x << 3 & 0xfc) << 16)); // | 0xff000000
#define _8888rev(px) (((px >> 16) & 0xff) | (px & 0xff00) | ((px << 16) & 0xff0000)); // | 0xff000000)


void RGBA(int pix, void *destination, void *source, int yy, int yn, int xw, int xh, int dw, int pad, int rot);

void i565(void *destination, void *source, int yy, int yn, int xw, int pad);
void i8888(void *destination, void *source, int yy, int yn, int xw, int pad);
void i565r(void *destination, void *source, int yy, int yn, int xw, int xh, int dw, int pad, int rot);
void i8888r(void *destination, void *source, int yy, int yn, int xw, int xh, int dw, int pad, int rot);

uint32_t px8888rev(uint32_t px);

#endif
