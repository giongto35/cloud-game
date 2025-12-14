package media

/*
   #cgo pkg-config: speexdsp
   #cgo st LDFLAGS: -l:libspeexdsp.a

   #include <stdint.h>
   #include "speex_resampler.h"
*/
import "C"

import "errors"

type Resampler struct {
	resampler    *C.SpeexResamplerState
	outBuff      []int16 // one of these buffers used when typed data read
	outBuffFloat []float32
	channels     int
	multiplier   float32
}

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

const (
	reserve = 1.1
)

// ResamplerInit Create a new resampler with integer input and output rates
// Resampling quality between 0 and 10, where 0 has poor quality
// and 10 has very high quality
func ResamplerInit(channels, inRate, outRate, quality int) (*Resampler, error) {
	err := C.int(0)
	r := &Resampler{channels: channels}
	r.multiplier = float32(outRate) / float32(inRate) * 1.1
	r.resampler = C.speex_resampler_init(C.spx_uint32_t(channels),
		C.spx_uint32_t(inRate), C.spx_uint32_t(outRate), C.int(quality), &err)
	if r.resampler == nil {
		return nil, StrError(int(err))
	}
	return r, nil
}

// Destroy a resampler
func (r *Resampler) Destroy() error {
	if r.resampler != nil {
		C.speex_resampler_destroy((*C.SpeexResamplerState)(r.resampler))
		return nil
	}
	return StrError(ErrorInvalidArg)
}

// ProcessIntInterleaved Resample an int slice interleaved
func (r *Resampler) ProcessIntInterleaved(in []int16) (int, []int16, error) {
	outBuffCap := int(float32(len(in)) * r.multiplier)
	if outBuffCap > cap(r.outBuff) {
		r.outBuff = make([]int16, int(float32(outBuffCap)*reserve)*4)
	}
	inLen := C.spx_uint32_t(len(in) / r.channels)
	outLen := C.spx_uint32_t(len(r.outBuff) / r.channels)
	res := C.speex_resampler_process_interleaved_int(
		r.resampler,
		(*C.spx_int16_t)(&in[0]),
		&inLen,
		(*C.spx_int16_t)(&r.outBuff[0]),
		&outLen,
	)
	if res != ErrorSuccess {
		return 0, nil, StrError(ErrorInvalidArg)
	}
	return int(inLen) * r.channels, r.outBuff[:outLen*2], nil
}

// StrError returns error message
func StrError(errorCode int) error {
	cS := C.speex_resampler_strerror(C.int(errorCode))
	if cS == nil {
		return nil
	}
	return errors.New(C.GoString(cS))
}
