package rgba

import (
	"image"
	"image/color"
)

func ToRGBA(img image.Image, flipped bool) *image.RGBA {
	bounds := img.Bounds()
	sw, sh := bounds.Dx(), bounds.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, sw, sh))
	for y := range sh {
		yy := y
		if flipped {
			yy = sh - y
		}
		for x := range sw {
			px := img.At(x, y)
			rgba := color.RGBAModel.Convert(px).(color.RGBA)
			dst.Set(x, yy, rgba)
		}
	}
	return dst
}
