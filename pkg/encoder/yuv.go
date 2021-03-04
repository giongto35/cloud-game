package encoder

import (
	"image"
	"unsafe"
)

// see: https://stackoverflow.com/questions/9465815/rgb-to-yuv420-algorithm-efficiency
// credit to https://github.com/poi5305/go-yuv2webRTC/blob/master/webrtc/webrtc.go

/*
#cgo CFLAGS: -Wall -O3

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

// Converts RGBA image to YUV (I420).
void rgbaToYuv(void *destination, void *source, int width, int height, chromaPos chroma) {
    const int image_size = width * height;
    unsigned char *rgba = source;
    unsigned char *dst_y = destination;
    unsigned char *dst_u = destination + image_size;
    unsigned char *dst_v = destination + image_size + image_size / 4;

    int x, y, r1, g1, b1;

    // Y plane
    for (y = 0; y < height; ++y) {
        for (x = 0; x < width; ++x) {
            r1 = 4 * (y * width + x);
            g1 = r1 + 1;
            b1 = g1 + 1;
            *dst_y++ = ((66 * rgba[r1] + 129 * rgba[g1] + 25 * rgba[b1]) >> 8) + 16;
        }
    }

    // U+V plane
    if (chroma == TOP_LEFT) {
        for (y = 0; y < height; y += 2) {
            for (x = 0; x < width; x += 2) {
                r1 = 4 * (y * width + x);
                g1 = r1 + 1;
                b1 = g1 + 1;
                *dst_u++ = ((-38 * rgba[r1] + -74 * rgba[g1] + 112 * rgba[b1]) >> 8) + 128;
                *dst_v++ = ((112 * rgba[r1] + -94 * rgba[g1] + -18 * rgba[b1]) >> 8) + 128;
            }
        }
    } else if (chroma == BETWEEN_FOUR) {
        int r2, g2, b2,
			r3, g3, b3,
            r4, g4, b4;

        for (y = 0; y < height; y += 2) {
            for (x = 0; x < width; x += 2) {
                r1 = 4 * (y * width + x);
                g1 = r1 + 1;
                b1 = g1 + 1;
                r2 = 4 * (y * width + x + 1);
                g2 = r2 + 1;
                b2 = g2 + 1;
                r3 = 4 * ((y + 1) * width + x);
                g3 = r3 + 1;
                b3 = g3 + 1;
                r4 = 4 * ((y + 1) * width + x + 1);
                g4 = r4 + 1;
                b4 = g4 + 1;

                *dst_u++ = (((-38 * rgba[r1] + -74 * rgba[g1] + 112 * rgba[b1]) >> 8) +
                            ((-38 * rgba[r2] + -74 * rgba[g2] + 112 * rgba[b2]) >> 8) +
                            ((-38 * rgba[r3] + -74 * rgba[g3] + 112 * rgba[b3]) >> 8) +
                            ((-38 * rgba[r4] + -74 * rgba[g4] + 112 * rgba[b4]) >> 8) + 512) >> 2;
                *dst_v++ = (((112 * rgba[r1] + -94 * rgba[g1] + -18 * rgba[b1]) >> 8) +
                            ((112 * rgba[r2] + -94 * rgba[g2] + -18 * rgba[b2]) >> 8) +
                            ((112 * rgba[r3] + -94 * rgba[g3] + -18 * rgba[b3]) >> 8) +
                            ((112 * rgba[r4] + -94 * rgba[g4] + -18 * rgba[b4]) >> 8) + 512) >> 2;
            }
        }
    }
}
*/
import "C"

type Yuv struct {
	data []byte
	w, h int
	pos  ChromaPos
}

type ChromaPos uint8

const (
	TopLeft ChromaPos = iota
	BetweenFour
)

func NewYuvBuffer(w, h int) Yuv {
	size := int(float32(w*h) * 1.5)
	return Yuv{
		data: make([]byte, size, size),
		w:    w,
		h:    h,
		pos:  BetweenFour,
	}
}

// FromRGBA converts RGBA colorspace into YUV I420 format inside the internal buffer.
func (yuv *Yuv) FromRGBA(rgba *image.RGBA) *Yuv {
	C.rgbaToYuv(unsafe.Pointer(&yuv.data[0]), unsafe.Pointer(&rgba.Pix[0]), C.int(yuv.w), C.int(yuv.h), C.chromaPos(yuv.pos))
	return yuv
}
