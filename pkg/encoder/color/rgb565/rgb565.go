package rgb565

import (
	"encoding/binary"
	"image"
	"image/color"
	"math"
)

// RGB565 is an in-memory image whose At method returns RGB565 values.
type RGB565 struct {
	// Pix holds the image's pixels, as RGB565 values in big-endian format. The pixel at
	// (x, y) starts at Pix[(y-p.Rect.Min.Y)*p.Stride + (x-p.Rect.Min.X)*2].
	Pix []uint8
	// Stride is the Pix stride (in bytes) between vertically adjacent pixels.
	Stride int
	// Rect is the image's bounds.
	Rect image.Rectangle
}

// Model is the model for RGB565 colors.
var Model = color.ModelFunc(func(c color.Color) color.Color {
	//if _, ok := c.(Color); ok {
	//	return c
	//}
	r, g, b, _ := c.RGBA()
	return Color(uint16((r<<8)&rMask | (g<<3)&gMask | (b>>3)&bMask))
})

const (
	rMask = 0b1111100000000000
	gMask = 0b0000011111100000
	bMask = 0b0000000000011111
)

// Color represents an RGB565 color.
type Color uint16

func (c Color) RGBA() (r, g, b, a uint32) {
	return uint32(math.Round(float64(c&rMask>>11)*255.0/31.0)) << 8,
		uint32(math.Round(float64(c&gMask>>5)*255.0/63.0)) << 8,
		uint32(math.Round(float64(c&bMask)*255.0/31.0)) << 8,
		0xffff
}

func NewRGB565(r image.Rectangle) *RGB565 {
	return &RGB565{Pix: make([]uint8, r.Dx()*r.Dy()<<1), Stride: r.Dx() << 1, Rect: r}
}

func (p *RGB565) Bounds() image.Rectangle { return p.Rect }
func (p *RGB565) ColorModel() color.Model { return Model }
func (p *RGB565) PixOffset(x, y int) int  { return (x-p.Rect.Min.X)<<1 + (y-p.Rect.Min.Y)*p.Stride }

func (p *RGB565) At(x, y int) color.Color {
	i := p.PixOffset(x, y)
	return Color(binary.LittleEndian.Uint16(p.Pix[i : i+2]))
}

func (p *RGB565) Set(x, y int, c color.Color) {
	i := p.PixOffset(x, y)
	binary.LittleEndian.PutUint16(p.Pix[i:i+2], uint16(Model.Convert(c).(Color)))
}
