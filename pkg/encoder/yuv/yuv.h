#ifndef YUV_H__
#define YUV_H__

#include <stdint.h>

// Converts RGBA image to YUV (I420) with BT.601 studio color range.
void rgbaToYuv(void *destination, void *source, int width, int height);

// Converts RGBA image chunk to YUV (I420) chroma with BT.601 studio color range.
// pos contains a shift value for chunks.
// deu, dev contains constant shifts for U, V planes in the resulting array.
// chroma (0, 1) selects chroma estimation algorithm.
void chroma(void *destination, void *source, int pos, int deu, int dev, int width, int height);

// Converts RGBA image chunk to YUV (I420) luma with BT.601 studio color range.
void luma(void *destination, void *source, int pos, int width, int height);

#endif
