package image

import (
	"image"
	"sync"
	"unsafe"

	"golang.org/x/image/draw"
)

/*
#cgo CFLAGS: -Wall
#include "canvas.h"
*/
import "C"

// Canvas is a stateful drawing surface, i.e. image.RGBA
type Canvas struct {
	enabled  bool
	w, h     int
	vertical bool
	pool     sync.Pool
	wg       sync.WaitGroup
}

type Frame struct{ image.RGBA }

func (f *Frame) Unwrap() image.RGBA { return f.RGBA }
func (f *Frame) Opaque() bool       { return true }
func (f *Frame) Copy() Frame {
	return Frame{image.RGBA{Pix: append([]uint8{}, f.Pix...), Stride: f.Stride, Rect: f.Rect}}
}

const (
	BitFormatShort5551  = iota // BIT_FORMAT_SHORT_5_5_5_1 has 5 bits R, 5 bits G, 5 bits B, 1 bit alpha
	BitFormatInt8888Rev        // BIT_FORMAT_INT_8_8_8_8_REV has 8 bits R, 8 bits G, 8 bits B, 8 bit alpha
	BitFormatShort565          // BIT_FORMAT_SHORT_5_6_5 has 5 bits R, 6 bits G, 5 bits
)

const (
	ScaleNot              = iota // skips image interpolation
	ScaleNearestNeighbour        // nearest neighbour interpolation
	ScaleBilinear                // bilinear interpolation
)

func Resize(scaleType int, src *image.RGBA, out *image.RGBA) {
	// !to do set it once instead switching on each iteration
	switch scaleType {
	case ScaleBilinear:
		draw.ApproxBiLinear.Scale(out, out.Bounds(), src, src.Bounds(), draw.Src, nil)
	case ScaleNot:
		fallthrough
	case ScaleNearestNeighbour:
		fallthrough
	default:
		draw.NearestNeighbor.Scale(out, out.Bounds(), src, src.Bounds(), draw.Src, nil)
	}
}

type Rotation uint

const (
	A90 Rotation = iota + 1
	A180
	A270
	F180 // F180 is flipped Y
)

func NewCanvas(w, h, size int) *Canvas {
	return &Canvas{
		enabled:  true,
		w:        w,
		h:        h,
		vertical: h > w, // input is inverted
		pool: sync.Pool{New: func() any {
			i := Frame{image.RGBA{
				Pix:  make([]uint8, size<<2),
				Rect: image.Rectangle{Max: image.Point{X: w, Y: h}},
			}}
			return &i
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
	if c.enabled {
		c.pool.Put(i)
	}
}
func (c *Canvas) Clear()                  { c.wg = sync.WaitGroup{} }
func (c *Canvas) SetEnabled(enabled bool) { c.enabled = enabled }

func (c *Canvas) Draw(encoding uint32, rot Rotation, w, h, packedW, bpp int, data []byte, th int) *Frame {
	dst := c.Get(w, h)
	if th == 0 {
		frame(encoding, dst, data, 0, h, w, h, packedW, bpp, rot)
	} else {
		hn := h / th
		c.wg.Add(th)
		for i := 0; i < th; i++ {
			xx := hn * i
			go func() {
				frame(encoding, dst, data, xx, hn, w, h, packedW, bpp, rot)
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
		Resize(ScaleNearestNeighbour, &dst.RGBA, &out.RGBA)
		c.Put(dst)
		return out
	}

	return dst
}

func frame(encoding uint32, dst *Frame, data []byte, yy int, hn int, w int, h int, pwb int, bpp int, rot Rotation) {
	sPtr := unsafe.Pointer(&data[yy*pwb])
	dPtr := unsafe.Pointer(&dst.Pix[yy*dst.Stride])
	// some cores can zero-right-pad rows to the packed width value
	pad := pwb - w*bpp
	if pad < 0 {
		pad = 0
	}
	if rot != 0 {
		dPtr = unsafe.Pointer(&dst.Pix[0])
	}
	C.RGBA(C.int(encoding), dPtr, sPtr, C.int(yy), C.int(yy+hn), C.int(w), C.int(h), C.int(dst.Stride>>2), C.int(pad), C.int(rot))
}

func _8888rev(px uint32) uint32 { return uint32(C.px8888rev(C.uint32_t(px))) }

func rotate(t int, x int, y int, w int, h int) (int, int) {
	return int(C.rot_x(C.int(t), C.int(x), C.int(y), C.int(w), C.int(h))),
		int(C.rot_y(C.int(t), C.int(x), C.int(y), C.int(w), C.int(h)))
}
