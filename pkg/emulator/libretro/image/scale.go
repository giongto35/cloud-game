package image

import (
	"image"
)

const (
	// skips image interpolation
	ScaleSkip = -1
	// initial image interpolation algorithm
	ScaleOld = 0
	// nearest neighbour interpolation
	ScaleNearestNeighbour = 1
)

func Resize(scaleType int, fn Format, w int, h int, packedW int, vw int, vh int, bpp int, data []byte, image *image.RGBA) {
	// !to do set it once instead switching on each iteration
	// !to do skip resize if w=vw h=vh
	switch scaleType {
	case ScaleSkip:
		skip(fn, w, h, packedW, vw, vh, bpp, data, image)
	case ScaleNearestNeighbour:
		nearest(fn, w, h, packedW, vw, vh, bpp, data, image)
	case ScaleOld:
		fallthrough
	default:
		old(fn, w, h, packedW, vw, vh, bpp, data, image)
	}
}

func old(fn Format, w int, h int, packedW int, vw int, vh int, bpp int, data []byte, image *image.RGBA) {
	seek := 0

	scaleWidth := float64(vw) / float64(w)
	scaleHeight := float64(vh) / float64(h)

	for y := 0; y < h; y++ {
		for x := 0; x < packedW; x++ {
			xx := int(float64(x) * scaleWidth)
			yy := int(float64(y) * scaleHeight)
			if xx < vw {
				fn(data, seek, xx, yy, image)
			}

			seek += bpp
		}
	}
}

func skip(fn Format, w int, h int, packedW int, _ int, _ int, bpp int, data []byte, image *image.RGBA) {
	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			index := (i * packedW) + j
			index *= bpp

			fn(data, index, j, i, image)
		}
	}
}

func nearest(fn Format, w int, h int, packedW int, vw int, vh int, bpp int, data []byte, image *image.RGBA) {
	xRatio := ((w << 16) / vw) + 1
	yRatio := ((h << 16) / vh) + 1

	for i := 0; i < vh; i++ {
		y2 := (i * yRatio) >> 16
		for j := 0; j < vw; j++ {
			x2 := (j * xRatio) >> 16

			index := (y2 * packedW) + x2
			index *= bpp

			fn(data, index, j, i, image)
		}
	}
}

//func bilinear(fn Format, w int, h int, packedW int, vw int, vh int, bpp int, data []byte, image *image.RGBA) {
//	// !to implement
//}
