package recorder

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

func AddLabel(img *image.RGBA, x, y int, label string) {
	draw.Draw(img, image.Rect(x, y, x+len(label)*7+3, y+12), &image.Uniform{C: color.RGBA{}}, image.Point{}, draw.Src)
	(&font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.Int26_6((x + 2) * 64), Y: fixed.Int26_6((y + 10) * 64)},
	}).DrawString(label)
}

func clone(src image.Image) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}

func TimeFormat(d time.Duration) string {
	mms := int(d.Milliseconds())
	ms := mms % 1000
	s := (mms / 1000) % 60
	m := (mms / (1000 * 60)) % 60
	h := (mms / (1000 * 60 * 60)) % 24
	return fmt.Sprintf("%02d:%02d:%02d.%03d", h, m, s, ms)
}
