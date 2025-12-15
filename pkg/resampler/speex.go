package resampler

/*
   #cgo pkg-config: speexdsp
   #cgo st LDFLAGS: -l:libspeexdsp.a

   #include <stdint.h>
   #include "speex_resampler.h"
*/
import "C"

import (
	"errors"
	"unsafe"
)

// Quality
const (
	QualityMax     = 10
	QualityMin     = 0
	QualityDefault = 4
	QualityDesktop = 5
	QualityVoid    = 3
)

// Errors
const (
	ErrorSuccess = iota
	ErrorAllocFailed
	ErrorBadState
	ErrorInvalidArg
	ErrorPtrOverlap
	ErrorMaxError
)

type Resampler struct {
	resampler *C.SpeexResamplerState
	channels  int
	inRate    int
	outRate   int
}

func Init(channels, inRate, outRate, quality int) (*Resampler, error) {
	var err C.int
	r := &Resampler{
		channels: channels,
		inRate:   inRate,
		outRate:  outRate,
	}

	r.resampler = C.speex_resampler_init(
		C.spx_uint32_t(channels),
		C.spx_uint32_t(inRate),
		C.spx_uint32_t(outRate),
		C.int(quality),
		&err,
	)

	if r.resampler == nil {
		return nil, StrError(int(err))
	}

	C.speex_resampler_skip_zeros(r.resampler)

	return r, nil
}

func (r *Resampler) Destroy() {
	if r.resampler != nil {
		C.speex_resampler_destroy(r.resampler)
		r.resampler = nil
	}
}

// Process performs resampling.
// Returns written samples count and error if any.
func (r *Resampler) Process(out, in []int16) (int, error) {
	if len(in) == 0 || len(out) == 0 {
		return 0, nil
	}

	inLen := C.spx_uint32_t(len(in) / r.channels)
	outLen := C.spx_uint32_t(len(out) / r.channels)

	res := C.speex_resampler_process_interleaved_int(
		r.resampler,
		(*C.spx_int16_t)(unsafe.Pointer(&in[0])),
		&inLen,
		(*C.spx_int16_t)(unsafe.Pointer(&out[0])),
		&outLen,
	)

	if res != ErrorSuccess {
		return 0, StrError(int(res))
	}

	return int(outLen) * r.channels, nil
}

func StrError(errorCode int) error {
	cS := C.speex_resampler_strerror(C.int(errorCode))
	if cS == nil {
		return nil
	}
	return errors.New(C.GoString(cS))
}
