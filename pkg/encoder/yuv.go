package encoder

import (
	"image"
	"unsafe"
)

// see: https://stackoverflow.com/questions/9465815/rgb-to-yuv420-algorithm-efficiency
// credit to https://github.com/poi5305/go-yuv2webRTC/blob/master/webrtc/webrtc.go

/*
void rgba2yuv(void * destination, void * source, int width, int height, int stride) {
  const int image_size = width * height;
  unsigned char * rgba = source;
  unsigned char * dst_y = destination;
  unsigned char * dst_u = destination + image_size;
  unsigned char * dst_v = destination + image_size + image_size / 4;

  int i, x, y;
  // Y plane
  for (y = 0; y < height; ++y) {
    for (x = 0; x < width; ++x) {
      i = y * (width + stride) + x;
      * dst_y++ = ((66 * rgba[4 * i] + 129 * rgba[4 * i + 1] + 25 * rgba[4 * i + 2]) >> 8) + 16;
    }
  }

  // U plane
  for (y = 0; y < height; y += 2) {
    for (x = 0; x < width; x += 2) {
      i = y * (width + stride) + x;
      * dst_u++ = ((-38 * rgba[4 * i] + -74 * rgba[4 * i + 1] + 112 * rgba[4 * i + 2]) >> 8) + 128;
    }
  }

  // V plane
  for (y = 0; y < height; y += 2) {
    for (x = 0; x < width; x += 2) {
      i = y * (width + stride) + x;
      * dst_v++ = ((112 * rgba[4 * i] + -94 * rgba[4 * i + 1] + -18 * rgba[4 * i + 2]) >> 8) + 128;
    }
  }
}
*/
import "C"

type Yuv struct {
	data []byte
	w, h int
}

func NewYuvBuffer(w, h int) Yuv {
	size := int(float32(w*h) * 1.5)
	return Yuv{
		data: make([]byte, size, size),
		w:    w,
		h:    h,
	}
}

// FromRGBA converts RGBA colorspace into YUV I420 format inside the internal buffer.
func (yuv *Yuv) FromRGBA(rgba *image.RGBA) *Yuv {
	C.rgba2yuv(unsafe.Pointer(&yuv.data[0]), unsafe.Pointer(&rgba.Pix[0]), C.int(yuv.w), C.int(yuv.h), C.int(0))
	return yuv
}
