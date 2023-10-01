package yuv

import (
	"image"
	"sync"
	"unsafe"
)

/*
#cgo CFLAGS: -Wall
#include "yuv.h"
*/
import "C"

type ImgProcessor interface {
	Process(rgba *image.RGBA) []byte
	Put(*[]byte)
}

type Options struct {
	Threads int
}

type processor struct {
	w, h int

	// cache
	ww   C.int
	pool sync.Pool
}

type threadedProcessor struct {
	*processor

	// threading
	threads int
	chunk   int

	// cache
	chromaU C.int
	chromaV C.int
	wg      sync.WaitGroup
}

// NewYuvImgProcessor creates new YUV image converter from RGBA.
func NewYuvImgProcessor(w, h int, opts *Options) ImgProcessor {
	bufSize := int(float32(w*h) * 1.5)

	processor := processor{
		w:  w,
		h:  h,
		ww: C.int(w),
		pool: sync.Pool{New: func() any {
			b := make([]byte, bufSize)
			return &b
		}},
	}

	if opts != nil && opts.Threads > 0 {
		// chunks the image evenly
		chunk := h / opts.Threads
		if chunk%2 != 0 {
			chunk--
		}

		return &threadedProcessor{
			chromaU:   C.int(w * h),
			chromaV:   C.int(w*h + w*h/4),
			chunk:     chunk,
			processor: &processor,
			threads:   opts.Threads,
			wg:        sync.WaitGroup{},
		}
	}
	return &processor
}

// Process converts RGBA colorspace into YUV I420 format inside the internal buffer.
// Non-threaded version.
func (yuv *processor) Process(rgba *image.RGBA) []byte {
	buf := *yuv.pool.Get().(*[]byte)
	C.rgbaToYuv(unsafe.Pointer(&buf[0]), unsafe.Pointer(&rgba.Pix[0]), yuv.ww, C.int(yuv.h))
	return buf
}

func (yuv *processor) Put(x *[]byte) { yuv.pool.Put(x) }

// Process converts RGBA colorspace into YUV I420 format inside the internal buffer.
// Threaded version.
//
// We divide the input image into chunks by the number of available CPUs.
// Each chunk should contain 2, 4, 6, etc. rows of the image.
//
//	      8x4          CPU (2)
//	x x x x x x x x  | Coroutine 1
//	x x x x x x x x  | Coroutine 1
//	x x x x x x x x  | Coroutine 2
//	x x x x x x x x  | Coroutine 2
func (yuv *threadedProcessor) Process(rgba *image.RGBA) []byte {
	src := unsafe.Pointer(&rgba.Pix[0])
	buf := *yuv.pool.Get().(*[]byte)
	dst := unsafe.Pointer(&buf[0])
	yuv.wg.Add(yuv.threads << 1)
	chunk := yuv.w * yuv.chunk
	for i := 0; i < yuv.threads; i++ {
		pos, hh := C.int(i*chunk), C.int(yuv.chunk)
		if i == yuv.threads-1 {
			hh = C.int(yuv.h - i*yuv.chunk)
		}
		go yuv.chroma_(src, dst, pos, hh)
		go yuv.luma_(src, dst, pos, hh)
	}
	yuv.wg.Wait()
	return buf
}

func (yuv *threadedProcessor) luma_(src unsafe.Pointer, dst unsafe.Pointer, pos C.int, hh C.int) {
	C.luma(dst, src, pos, yuv.ww, hh)
	yuv.wg.Done()
}

func (yuv *threadedProcessor) chroma_(src unsafe.Pointer, dst unsafe.Pointer, pos C.int, hh C.int) {
	C.chroma(dst, src, pos, yuv.chromaU, yuv.chromaV, yuv.ww, hh)
	yuv.wg.Done()
}
