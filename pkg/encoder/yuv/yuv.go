package yuv

import (
	"image"
	"runtime"
	"sync"
	"unsafe"
)

/*
#cgo CFLAGS: -Wall -O3

#include "yuv.h"
*/
import "C"

type Yuv struct {
	Data []byte
	w, h int
	pos  ChromaPos

	// threading
	threads int
	chunk   int
	wg      sync.WaitGroup
}

type ChromaPos uint8

const (
	TopLeft ChromaPos = iota
	BetweenFour
)

func NewYuvBuffer(w, h int) Yuv {
	size := int(float32(w*h) * 1.5)
	threads := runtime.NumCPU()

	// chunks the image evenly
	chunk := h / threads
	if chunk%2 != 0 {
		chunk--
	}

	return Yuv{
		Data: make([]byte, size, size),
		w:    w,
		h:    h,
		pos:  BetweenFour,

		threads: threads,
		chunk:   chunk,
	}
}

// FromRgbaNonThreaded converts RGBA colorspace into YUV I420 format inside the internal buffer.
// Non-threaded version.
func (yuv *Yuv) FromRgbaNonThreaded(rgba *image.RGBA) *Yuv {
	C.rgbaToYuv(unsafe.Pointer(&yuv.Data[0]), unsafe.Pointer(&rgba.Pix[0]), C.int(yuv.w), C.int(yuv.h), C.chromaPos(yuv.pos))
	return yuv
}

// FromRgbaThreaded converts RGBA colorspace into YUV I420 format inside the internal buffer.
// Threaded version.
//
// We divide the input image into chunks by the number of available CPUs.
// Each chunk should contain 2, 4, 6, and etc. rows of the image.
//
//        8x4          CPU (2)
//  x x x x x x x x  | Thread 1
//  x x x x x x x x  | Thread 1
//  x x x x x x x x  | Thread 2
//  x x x x x x x x  | Thread 2
//
func (yuv *Yuv) FromRgbaThreaded(rgba *image.RGBA) *Yuv {
	yuv.wg.Add(2 * yuv.threads)

	src := unsafe.Pointer(&rgba.Pix[0])
	dst := unsafe.Pointer(&yuv.Data[0])
	ww := C.int(yuv.w)
	chroma := C.chromaPos(yuv.pos)
	chromaU := C.int(yuv.w * yuv.h)
	chromaV := C.int(chromaU + chromaU/4)

	for i := 0; i < yuv.threads; i++ {
		pos, hh := C.int(yuv.w*i*yuv.chunk), C.int(yuv.chunk)
		// we need to know how many pixels left
		// if the image can't be divided evenly
		// between all the threads
		if i == yuv.threads-1 {
			hh = C.int(yuv.h - i*yuv.chunk)
		}
		go func() { defer yuv.wg.Done(); C.luma(dst, src, pos, ww, hh) }()
		go func() { defer yuv.wg.Done(); C.chroma(dst, src, pos, chromaU, chromaV, ww, hh, chroma) }()
	}
	yuv.wg.Wait()
	return yuv
}
