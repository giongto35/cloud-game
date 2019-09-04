package x264

import (
	"image"
	"image/color"
	"image/draw"
)

// YCbCr is an in-memory image of Y'CbCr colors.
type YCbCr struct {
	*image.YCbCr
}

// NewYCbCr returns a new YCbCr image with the given bounds and subsample ratio.
func NewYCbCr(r image.Rectangle) *YCbCr {
	return &YCbCr{image.NewYCbCr(r, image.YCbCrSubsampleRatio420)}
}

// Set sets pixel color.
func (p *YCbCr) Set(x, y int, c color.Color) {
	p.setYCbCr(x, y, p.ColorModel().Convert(c).(color.YCbCr))
}

func (p *YCbCr) setYCbCr(x, y int, c color.YCbCr) {
	if !image.Pt(x, y).In(p.Rect) {
		return
	}

	yi := p.YOffset(x, y)
	ci := p.COffset(x, y)

	p.Y[yi] = c.Y
	p.Cb[ci] = c.Cb
	p.Cr[ci] = c.Cr
}

// ToYCbCr converts image.Image to YCbCr.
func (p *YCbCr) ToYCbCr(src image.Image) {
	bounds := src.Bounds()
	draw.Draw(p, bounds, src, bounds.Min, draw.Src)
}
