package image

import (
	"image"
	"sync"
	"unsafe"
)

const (
	BitFormatShort5551  = iota // BIT_FORMAT_SHORT_5_5_5_1 has 5 bits R, 5 bits G, 5 bits B, 1 bit alpha
	BitFormatInt8888Rev        // BIT_FORMAT_INT_8_8_8_8_REV has 8 bits R, 8 bits G, 8 bits B, 8 bit alpha
	BitFormatShort565          // BIT_FORMAT_SHORT_5_6_5 has 5 bits R, 6 bits G, 5 bits
)

var wg sync.WaitGroup

func NewRGBA(w, h int) *image.RGBA { return image.NewRGBA(image.Rect(0, 0, w, h)) }

func Draw(dst *image.RGBA, encoding uint32, rot *Rotate, w, h, packedW, bpp int, data []byte, th int) {
	pwb := packedW * bpp
	if th == 0 {
		frame(encoding, dst, data, 0, h, h, w, pwb, bpp, rot)
	} else {
		hn := h / th
		wg.Add(th)
		for i := 0; i < th; i++ {
			xx := hn * i
			go func() {
				frame(encoding, dst, data, xx, hn, h, w, pwb, bpp, rot)
				wg.Done()
			}()
		}
		wg.Wait()
	}
}

func ReScale(scaleType, w, h int, src *image.RGBA) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	Resize(scaleType, src, out)
	return out
}

func frame(encoding uint32, src *image.RGBA, data []byte, xx int, hn int, h int, w int, pwb int, bpp int, rot *Rotate) {
	var px uint32
	var dst *uint32
	for y, l, lx, row := xx, xx+hn, 0, 0; y < l; y++ {
		row = y * src.Stride
		lx = y * pwb
		for x, k := 0, 0; x < w; x++ {
			if rot == nil {
				k = x<<2 + row
			} else {
				dx, dy := rot.Call(x, y, w, h)
				k = dx<<2 + dy*src.Stride
			}
			dst = (*uint32)(unsafe.Pointer(&src.Pix[k]))
			px = *(*uint32)(unsafe.Pointer(&data[x*bpp+lx]))
			// LE, BE might not work
			switch encoding {
			case BitFormatShort565:
				i565(dst, px)
			case BitFormatInt8888Rev:
				ix8888(dst, px)
			}
		}
	}
}

func i565(dst *uint32, px uint32) {
	*dst = ((px >> 8) & 0xf8) | (((px >> 3) & 0xfc) << 8) | (((px << 3) & 0xfc) << 16) + 0xff000000
	// setting the last byte to 255 allows saving RGBA images to PNG not as black squares
}

func ix8888(dst *uint32, px uint32) {
	*dst = ((px >> 16) & 0xff) | (px & 0xff00) | ((px << 16) & 0xff0000) + 0xff000000
}

func Clear() {
	wg = sync.WaitGroup{}
}
