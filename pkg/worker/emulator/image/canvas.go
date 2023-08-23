package image

import (
	"image"
	"math/bits"
	"sync"
	"unsafe"
)

// Canvas is a stateful drawing surface, i.e. image.RGBA
type Canvas struct {
	enabled  bool
	w, h     int
	vertical bool
	pool     sync.Pool
	wg       sync.WaitGroup
}

type Frame struct {
	*image.RGBA
}

func (f *Frame) Opaque() bool { return true }
func (f *Frame) Copy() Frame {
	return Frame{&image.RGBA{
		Pix:    append([]uint8{}, f.Pix...),
		Stride: f.Stride,
		Rect:   f.Rect,
	}}
}

const (
	BitFormatShort5551  = iota // BIT_FORMAT_SHORT_5_5_5_1 has 5 bits R, 5 bits G, 5 bits B, 1 bit alpha
	BitFormatInt8888Rev        // BIT_FORMAT_INT_8_8_8_8_REV has 8 bits R, 8 bits G, 8 bits B, 8 bit alpha
	BitFormatShort565          // BIT_FORMAT_SHORT_5_6_5 has 5 bits R, 6 bits G, 5 bits
)

func NewCanvas(w, h, size int) *Canvas {
	return &Canvas{
		enabled:  true,
		w:        w,
		h:        h,
		vertical: h > w, // input is inverted
		pool: sync.Pool{New: func() any {
			return &Frame{&image.RGBA{
				Pix:  make([]uint8, size<<2),
				Rect: image.Rectangle{Max: image.Point{X: w, Y: h}},
			}}
		}},
	}
}

func (c *Canvas) Get(w, h int) *Frame {
	i := c.pool.Get().(*Frame)
	if c.vertical {
		w, h = h, w
	}
	i.Stride = w << 2
	i.Pix = i.Pix[:i.Stride*h]
	i.Rect.Max.X = w
	i.Rect.Max.Y = h
	return i
}

func (c *Canvas) Put(i *Frame) {
	if c.enabled && i != nil {
		c.pool.Put(i)
	}
}
func (c *Canvas) Clear()                  { c.wg = sync.WaitGroup{} }
func (c *Canvas) SetEnabled(enabled bool) { c.enabled = enabled }

func (c *Canvas) Draw(encoding uint32, rot *Rotate, w, h, packedW, bpp int, data []byte, th int) *Frame {
	dst := c.Get(w, h)
	if th == 0 {
		frame(encoding, dst, data, 0, h, h, w, packedW, bpp, rot)
	} else {
		hn := h / th
		c.wg.Add(th)
		for i := 0; i < th; i++ {
			xx := hn * i
			go func() {
				frame(encoding, dst, data, xx, hn, h, w, packedW, bpp, rot)
				c.wg.Done()
			}()
		}
		c.wg.Wait()
	}

	// rescale
	if dst.Rect.Dx() != c.w || dst.Rect.Dy() != c.h {
		ww := c.w
		hh := c.h
		// w, h supposedly have been swapped before
		if c.vertical {
			ww, hh = c.h, c.w
		}
		out := c.Get(ww, hh)
		Resize(ScaleNearestNeighbour, dst.RGBA, out.RGBA)
		c.Put(dst)
		return out
	}

	return dst
}

func frame(encoding uint32, dst *Frame, data []byte, yy int, hn int, h int, w int, pwb int, bpp int, rot *Rotate) {
	sPtr := unsafe.Pointer(&data[yy*pwb])
	dPtr := unsafe.Pointer(&dst.Pix[yy*dst.Stride])
	// some cores can zero-right-pad rows to the packed width value
	pad := pwb - w*bpp
	yn := yy + hn

	if rot == nil {
		// LE, BE might not work
		switch encoding {
		case BitFormatShort565:
			for y := yy; y < yn; y++ {
				for x := 0; x < w; x++ {
					i565((*uint32)(dPtr), uint32(*(*uint16)(sPtr)))
					sPtr = unsafe.Add(sPtr, uintptr(bpp))
					dPtr = unsafe.Add(dPtr, uintptr(4))
				}
				if pad > 0 {
					sPtr = unsafe.Add(sPtr, uintptr(pad))
				}
			}
		case BitFormatInt8888Rev:
			for y := yy; y < yn; y++ {
				for x := 0; x < w; x++ {
					ix8888((*uint32)(dPtr), *(*uint32)(sPtr))
					sPtr = unsafe.Add(sPtr, uintptr(bpp))
					dPtr = unsafe.Add(dPtr, uintptr(4))
				}
				if pad > 0 {
					sPtr = unsafe.Add(sPtr, uintptr(pad))
				}
			}
		}
	} else {
		switch encoding {
		case BitFormatShort565:
			for y := yy; y < yn; y++ {
				for x, k := 0, 0; x < w; x++ {
					dx, dy := rot.Call(x, y, w, h)
					k = dx<<2 + dy*dst.Stride
					dPtr = unsafe.Pointer(&dst.Pix[k])
					i565((*uint32)(dPtr), uint32(*(*uint16)(sPtr)))
					sPtr = unsafe.Add(sPtr, uintptr(bpp))
				}
				if pad > 0 {
					sPtr = unsafe.Add(sPtr, uintptr(pad))
				}
			}
		case BitFormatInt8888Rev:
			for y := yy; y < yn; y++ {
				for x, k := 0, 0; x < w; x++ {
					dx, dy := rot.Call(x, y, w, h)
					k = dx<<2 + dy*dst.Stride
					dPtr = unsafe.Pointer(&dst.Pix[k])
					ix8888((*uint32)(dPtr), *(*uint32)(sPtr))
					sPtr = unsafe.Add(sPtr, uintptr(bpp))
				}
				if pad > 0 {
					sPtr = unsafe.Add(sPtr, uintptr(pad))
				}
			}
		}
	}
}

func i565(dst *uint32, px uint32) {
	*dst = (px >> 8 & 0xf8) | ((px >> 3 & 0xfc) << 8) | ((px << 3 & 0xfc) << 16) // | 0xff000000
	// setting the last byte to 255 allows saving RGBA images to PNG not as black squares
}

func ix8888(dst *uint32, px uint32) {
	//*dst = ((px >> 16) & 0xff) | (px & 0xff00) | ((px << 16) & 0xff0000) + 0xff000000
	*dst = bits.ReverseBytes32(px << 8) //| 0xff000000
}
