package image

import (
	"image"
	"image/color"
)

const (
	// BIT_FORMAT_SHORT_5_5_5_1 has 5 bits R, 5 bits G, 5 bits B, 1 bit alpha
	BIT_FORMAT_SHORT_5_5_5_1 = iota
	// BIT_FORMAT_INT_8_8_8_8_REV has 8 bits R, 8 bits G, 8 bits B, 8 bit alpha
	BIT_FORMAT_INT_8_8_8_8_REV
	// BIT_FORMAT_SHORT_5_6_5 has 5 bits R, 6 bits G, 5 bits
	BIT_FORMAT_SHORT_5_6_5
)

type Format func(data []byte, index int, x int, y int, image *image.RGBA)

func rgb565(data []byte, index int, x int, y int, image *image.RGBA) {
	pixel := (int)(data[index]) + ((int)(data[index+1]) << 8)
	b5 := pixel & 0x1F
	g6 := (pixel >> 5) & 0x3F
	r5 := pixel >> 11

	b8 := (b5*255 + 15) / 31
	g8 := (g6*255 + 31) / 63
	r8 := (r5*255 + 15) / 31

	image.Set(x, y, color.RGBA{R: byte(r8), G: byte(g8), B: byte(b8), A: 255})
}

func rgba8888(data []byte, index int, x int, y int, image *image.RGBA) {
	b8 := data[index]
	g8 := data[index+1]
	r8 := data[index+2]
	a8 := data[index+3]

	image.Set(x, y, color.RGBA{R: r8, G: g8, B: b8, A: a8})
}
