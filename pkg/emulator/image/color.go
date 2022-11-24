package image

import "unsafe"

const (
	BitFormatShort5551  = iota // BIT_FORMAT_SHORT_5_5_5_1 has 5 bits R, 5 bits G, 5 bits B, 1 bit alpha
	BitFormatInt8888Rev        // BIT_FORMAT_INT_8_8_8_8_REV has 8 bits R, 8 bits G, 8 bits B, 8 bit alpha
	BitFormatShort565          // BIT_FORMAT_SHORT_5_6_5 has 5 bits R, 6 bits G, 5 bits
)

type RGB struct {
	R, G, B uint8
}

type Format func(data []byte, index int) RGB

func Rgb565(data []byte, index int) RGB {
	pixel := *(*uint16)(unsafe.Pointer(&data[index]))
	return RGB{R: uint8((pixel >> 8) & 0xf8), G: uint8((pixel >> 3) & 0xfc), B: uint8((pixel << 3) & 0xfc)}
}

func Rgba8888(data []byte, index int) RGB {
	return RGB{R: data[index+2], G: data[index+1], B: data[index]}
}
