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

type ImgProcessor interface {
	Process(rgba *image.RGBA) ImgProcessor
	Get() []byte
}

type processor struct {
	Data []byte
	w, h int
	pos  ChromaPos

	// cache
	dst    unsafe.Pointer
	ww     C.int
	chroma C.chromaPos
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

type ChromaPos uint8

const (
	TopLeft ChromaPos = iota
	BetweenFour
)

// NewYuvImgProcessor creates new YUV image converter from RGBA.
func NewYuvImgProcessor(w, h int, options ...Option) ImgProcessor {
	opts := &Options{
		ChromaP: BetweenFour,
		Threads: runtime.NumCPU(),
	}
	opts.override(options...)

	bufSize := int(float32(w*h) * 1.5)
	buf := make([]byte, bufSize)

	processor := processor{
		Data:   buf,
		chroma: C.chromaPos(opts.ChromaP),
		dst:    unsafe.Pointer(&buf[0]),
		h:      h,
		pos:    opts.ChromaP,
		w:      w,
		ww:     C.int(w),
	}

	if opts.Threaded {
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

func (yuv *processor) Get() []byte { return yuv.Data }

// Process converts RGBA colorspace into YUV I420 format inside the internal buffer.
// Non-threaded version.
func (yuv *processor) Process(rgba *image.RGBA) ImgProcessor {
	C.rgbaToYuv(yuv.dst, unsafe.Pointer(&rgba.Pix[0]), yuv.ww, C.int(yuv.h), yuv.chroma)
	return yuv
}

func (yuv *threadedProcessor) Get() []byte { return yuv.Data }

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
func (yuv *threadedProcessor) Process(rgba *image.RGBA) ImgProcessor {
	src := &rgba.Pix[0]
	yuv.wg.Add(yuv.threads << 1)
	for i := 0; i < yuv.threads; i++ {
		pos, hh := C.int(yuv.w*i*yuv.chunk), C.int(yuv.chunk)
		if i == yuv.threads-1 {
			hh = C.int(yuv.h - i*yuv.chunk)
		}
		go yuv.luma_(src, pos, hh)
		go yuv.chroma_(src, pos, hh)
	}
	yuv.wg.Wait()
	return yuv
}

func (yuv *threadedProcessor) luma_(src *uint8, pos C.int, hh C.int) {
	C.luma(yuv.dst, unsafe.Pointer(src), pos, yuv.ww, hh)
	yuv.wg.Done()
}

func (yuv *threadedProcessor) chroma_(src *uint8, pos C.int, hh C.int) {
	C.chroma(yuv.dst, unsafe.Pointer(src), pos, yuv.chromaU, yuv.chromaV, yuv.ww, hh, yuv.chroma)
	yuv.wg.Done()
}
