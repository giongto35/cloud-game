// credit to https://github.com/poi5305/go-yuv2webRTC/blob/master/webrtc/webrtc.go
package util

import (
	"image"
	"log"
	"os/user"
	"unsafe"

	"github.com/giongto35/cloud-game/pkg/config"
)

// https://stackoverflow.com/questions/9465815/rgb-to-yuv420-algorithm-efficiency

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

// RgbaToYuv convert to yuv from rgba
func RgbaToYuv(rgba *image.RGBA) []byte {
	w := rgba.Rect.Max.X
	h := rgba.Rect.Max.Y
	size := int(float32(w*h) * 1.5)
	stride := rgba.Stride - w*4
	yuv := make([]byte, size, size)
	C.rgba2yuv(unsafe.Pointer(&yuv[0]), unsafe.Pointer(&rgba.Pix[0]), C.int(w), C.int(h), C.int(stride))
	return yuv
}

// RgbaToYuvInplace convert to yuv from rgba inplace to yuv. Avoid reallocation
func RgbaToYuvInplace(rgba *image.RGBA, yuv []byte, width, height int) {
	stride := rgba.Stride - width*4
	C.rgba2yuv(unsafe.Pointer(&yuv[0]), unsafe.Pointer(&rgba.Pix[0]), C.int(width), C.int(height), C.int(stride))
}

var homeDir string

func init() {
	u, err := user.Current()
	if err != nil {
		log.Fatalln(err)
	}
	homeDir = u.HomeDir
}

// GetSavePath returns save location of game based on roomID
func GetSavePath(roomID string) string {
	return savePath(roomID)
}

func savePath(hash string) string {
	return homeDir + "/.cr/save/" + hash + ".dat"
}

// GetVideoEncoder returns video encoder based on some qualification.
// Actually Android is only supporting VP8 but H264 has better encoding performance
// TODO: Better use useragent attribute from frontend
func GetVideoEncoder(isMobile bool) string {
	return config.CODEC_VP8
}
