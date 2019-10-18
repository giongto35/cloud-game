package image

import (
	"image"
)

func DrawRgbaImage(pixFormat int, scaleType int, w int, h int, packedW int, vw int, vh int, bpp int, data []byte, image *image.RGBA) {
	switch pixFormat {
	case BIT_FORMAT_SHORT_5_6_5:
		Resize(scaleType, rgb565, w, h, packedW, vw, vh, bpp, data, image)
	case BIT_FORMAT_INT_8_8_8_8_REV:
		Resize(scaleType, rgba8888, w, h, packedW, vw, vh, bpp, data, image)
	case BIT_FORMAT_SHORT_5_5_5_1:
		fallthrough
	default:
		image = nil
	}
}
