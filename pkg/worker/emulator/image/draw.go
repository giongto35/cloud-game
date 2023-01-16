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

func NewRGBA(w, h int) *image.RGBA {
	return &image.RGBA{
		Pix:    make([]uint8, (w*h)<<2),
		Stride: w << 2,
		Rect:   image.Rectangle{Max: image.Point{X: w, Y: h}},
	}
}

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
	out := NewRGBA(w, h)
	Resize(scaleType, src, out)
	return out
}

func frame(encoding uint32, dst *image.RGBA, data []byte, yy int, hn int, h int, w int, pwb int, bpp int, rot *Rotate) {
	srcPtr := unsafe.Pointer(&data[yy*pwb])
	dstPtr := unsafe.Pointer(&dst.Pix[yy*dst.Stride])
	// some cores can zero-right-pad rows to the packed width value
	pad := pwb - w*bpp
	yn := yy + hn

	if rot == nil {
		for y := yy; y < yn; y++ {
			for x := 0; x < w; x++ {
				// LE, BE might not work
				switch encoding {
				case BitFormatShort565:
					i565((*uint32)(dstPtr), *(*uint16)(srcPtr))
				case BitFormatInt8888Rev:
					ix8888((*uint32)(dstPtr), *(*uint32)(srcPtr))
				}
				srcPtr = unsafe.Add(srcPtr, uintptr(bpp))
				dstPtr = unsafe.Add(dstPtr, uintptr(4))
			}
			if pad > 0 {
				srcPtr = unsafe.Add(srcPtr, uintptr(pad))
			}
		}
	} else {
		for y := yy; y < yn; y++ {
			for x, k := 0, 0; x < w; x++ {
				dx, dy := rot.Call(x, y, w, h)
				k = dx<<2 + dy*dst.Stride
				dstPtr = unsafe.Pointer(&dst.Pix[k])
				switch encoding {
				case BitFormatShort565:
					i565((*uint32)(dstPtr), *(*uint16)(srcPtr))
				case BitFormatInt8888Rev:
					ix8888((*uint32)(dstPtr), *(*uint32)(srcPtr))
				}
				srcPtr = unsafe.Add(srcPtr, uintptr(bpp))
			}
			if pad > 0 {
				srcPtr = unsafe.Add(srcPtr, uintptr(pad))
			}
		}
	}
}

func i565(dst *uint32, px uint16) {
	*dst = (uint32(px>>8) & 0xf8) | ((uint32(px>>3) & 0xfc) << 8) | ((uint32(px<<3) & 0xfc) << 16) + 0xff000000
	// setting the last byte to 255 allows saving RGBA images to PNG not as black squares
}

func ix8888(dst *uint32, px uint32) {
	*dst = ((px >> 16) & 0xff) | (px & 0xff00) | ((px << 16) & 0xff0000) + 0xff000000
}

func Clear() {
	wg = sync.WaitGroup{}
}
