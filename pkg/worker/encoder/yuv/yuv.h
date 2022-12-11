
typedef enum {
    // It will take each TL pixel for chroma values.
    // XO  X   XO  X
    // X   X   X   X
    TOP_LEFT = 0,
    // It will take an average color from the 2x2 pixel group for chroma values.
    // X   X   X   X
    //   O       O
    // X   X   X   X
    BETWEEN_FOUR = 1
} chromaPos;

// Converts RGBA image to YUV (I420) with BT.601 studio color range.
void rgbaToYuv(void *destination, void *source, int width, int height, chromaPos chroma);

// Converts RGBA image chunk to YUV (I420) chroma with BT.601 studio color range.
// pos contains a shift value for chunks.
// deu, dev contains constant shifts for U, V planes in the resulting array.
// chroma (0, 1) selects chroma estimation algorithm.
void chroma(void *destination, void *source, int pos, int deu, int dev, int width, int height, chromaPos chroma);

// Converts RGBA image chunk to YUV (I420) luma with BT.601 studio color range.
void luma(void *destination, void *source, int pos, int width, int height);
