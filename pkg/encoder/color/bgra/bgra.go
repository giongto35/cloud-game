package bgra

import (
	"image"
	"image/color"
)

type BGRA struct {
	image.RGBA
}

var BGRAModel = color.ModelFunc(func(c color.Color) color.Color {
	if _, ok := c.(BGRAColor); ok {
		return c
	}
	r, g, b, a := c.RGBA()
	return BGRAColor{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
})

// BGRAColor represents a BGRA color.
type BGRAColor struct {
	R, G, B, A uint8
}

func (c BGRAColor) RGBA() (r, g, b, a uint32) {
	r = uint32(c.B)
	r |= r << 8
	g = uint32(c.G)
	g |= g << 8
	b = uint32(c.R)
	b |= b << 8
	a = uint32(255) //uint32(c.A)
	a |= a << 8
	return
}

func NewBGRA(r image.Rectangle) *BGRA {
	return &BGRA{*image.NewRGBA(r)}
}

func (p *BGRA) ColorModel() color.Model { return BGRAModel }
func (p *BGRA) At(x, y int) color.Color {
	i := p.PixOffset(x, y)
	s := p.Pix[i : i+4 : i+4]
	return BGRAColor{s[0], s[1], s[2], s[3]}
}

func (p *BGRA) Set(x, y int, c color.Color) {
	i := p.PixOffset(x, y)
	c1 := BGRAModel.Convert(c).(BGRAColor)
	s := p.Pix[i : i+4 : i+4]
	s[0] = c1.R
	s[1] = c1.G
	s[2] = c1.B
	s[3] = 255
}
