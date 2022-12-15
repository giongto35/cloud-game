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

type imageCache struct {
	image *image.RGBA
	w     int
	h     int
}

func (i *imageCache) get(w, h int) *image.RGBA {
	if i.w == w && i.h == h {
		return i.image
	}
	i.w, i.h = w, h
	i.image = image.NewRGBA(image.Rect(0, 0, w, h))
	return i.image
}

var (
	canvas1 = imageCache{image.NewRGBA(image.Rectangle{}), 0, 0}
	canvas2 = imageCache{image.NewRGBA(image.Rectangle{}), 0, 0}
	wg      sync.WaitGroup
)

func DrawRgbaImage(encoding uint32, rot *Rotate, scaleType int, flipV bool, w, h, packedW, bpp int,
	data []byte, dw, dh, th int) *image.RGBA {
	// !to implement own image interfaces img.Pix = bytes[]
	ww, hh := w, h
	if rot != nil && rot.IsEven {
		ww, hh = hh, ww
	}
	src := canvas1.get(ww, hh)

	normY := !flipV
	hn := h / th
	pwb := packedW * bpp
	wg.Add(th)
	for i := 0; i < th; i++ {
		xx := hn * i
		go func() {
			var px uint32
			var dst *uint32
			for y, yy, l, lx, row := xx, 0, xx+hn, 0, 0; y < l; y++ {
				if normY {
					yy = y
				} else {
					yy = (h - 1) - y
				}
				row = yy * src.Stride
				lx = y * pwb
				for x, k := 0, 0; x < w; x++ {
					if rot == nil {
						k = x<<2 + row
					} else {
						dx, dy := rot.Call(x, yy, w, h)
						k = dx<<2 + dy*src.Stride
					}
					px = *(*uint32)(unsafe.Pointer(&data[x*bpp+lx]))
					dst = (*uint32)(unsafe.Pointer(&src.Pix[k]))
					// LE, BE might not work
					switch encoding {
					case BitFormatShort565:
						i565(dst, px)
					case BitFormatInt8888Rev:
						ix8888(dst, px)
					}
				}
			}
			wg.Done()
		}()
	}
	wg.Wait()

	if ww == dw && hh == dh {
		return src
	} else {
		out := canvas2.get(dw, dh)
		Resize(scaleType, src, out)
		return out
	}
}

func i565(dst *uint32, px uint32) {
	*dst = ((px >> 8) & 0xf8) | (((px >> 3) & 0xfc) << 8) | (((px << 3) & 0xfc) << 16) // | 0xff000000
}

func ix8888(dst *uint32, px uint32) {
	*dst = ((px >> 16) & 0xff) | (px & 0xff00) | ((px << 16) & 0xff0000) // | 0xff000000
}

func Clear() {
	wg = sync.WaitGroup{}
	canvas1.get(0, 0)
	canvas2.get(0, 0)
}
