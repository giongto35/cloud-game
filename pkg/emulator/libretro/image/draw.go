package image

import (
	"image"
)

type imageCache struct {
	image *image.RGBA
	w int
	h int
}

var canvas = imageCache{
	image.NewRGBA(image.Rectangle{}),
	0,
	0,
}

func DrawRgbaImage(pixFormat Format, rotationFn Rotate, scaleType, w, h, packedW, bpp int, data []byte, dest *image.RGBA) {
	if pixFormat == nil {
		dest = nil
	}

	// !to implement own image interfaces img.Pix = bytes[]
	ww, hh := w, h
	if rotationFn.IsEven {
		ww, hh = hh, ww
	}
	src := getCanvas(ww, hh)

	drawImage(pixFormat, w, h, packedW, bpp, rotationFn, data, src)
	Resize(scaleType, src, dest)
}

func drawImage(toRGBA Format, w, h, packedW, bpp int, rotationFn Rotate, data []byte, image *image.RGBA) {
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			src := toRGBA(data, (x+y*packedW)*bpp)
			dx, dy := rotationFn.Call(x, y, w, h)
			i := dx*4 + dy*image.Stride
			dst := image.Pix[i : i+4 : i+4]
			dst[0] = src.R
			dst[1] = src.G
			dst[2] = src.B
			dst[3] = src.A
		}
	}
}

func getCanvas(w, h int) *image.RGBA {
	if canvas.w == w && canvas.h == h {
		return canvas.image
	}

	canvas.w, canvas.h = w, h
	canvas.image = image.NewRGBA(image.Rect(0, 0, w, h))

	return canvas.image
}

