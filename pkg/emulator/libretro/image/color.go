package image

import (
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

type Format func(data []byte, index int) color.RGBA

func Rgb565(data []byte, index int) color.RGBA {
	pixel := (int)(data[index]) + ((int)(data[index+1]) << 8)

	return color.RGBA{
		R: byte(((pixel>>11)*255 + 15) / 31),
		G: byte((((pixel>>5)&0x3F)*255 + 31) / 63),
		B: byte(((pixel&0x1F)*255 + 15) / 31),
		A: 255,
	}
}

func Rgba8888(data []byte, index int) color.RGBA {
	return color.RGBA{
		R: data[index+2],
		G: data[index+1],
		B: data[index],
		A: data[index+3],
	}
}
