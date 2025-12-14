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
	resampler *C.SpeexResamplerState
	channels  int
}

const (
	QualityMax     = 10
	QualityMin     = 0
	QualityDefault = 4
	QualityDesktop = 5
	QualityVoIP    = 3
)

func NewResampler(channels, inRate, outRate, quality int) (*Resampler, error) {
	var err C.int
	r := &Resampler{channels: channels}

	// Use fractional init for exact ratio
	g := gcd(outRate, inRate)
	r.resampler = C.speex_resampler_init_frac(
		C.spx_uint32_t(channels),
		C.spx_uint32_t(outRate/g),
		C.spx_uint32_t(inRate/g),
		C.spx_uint32_t(inRate),
		C.spx_uint32_t(outRate),
		C.int(quality),
		&err,
	)
	if r.resampler == nil {
		return nil, errors.New(C.GoString(C.speex_resampler_strerror(err)))
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

func (r *Resampler) Process(in, out []int16) (int, error) {
	if r.resampler == nil || len(in) < r.channels || len(out) < r.channels {
		return 0, nil
	}

	inLen := C.spx_uint32_t(len(in) / r.channels)
	outLen := C.spx_uint32_t(len(out) / r.channels)

	res := C.speex_resampler_process_interleaved_int(
		r.resampler,
		(*C.spx_int16_t)(&in[0]),
		&inLen,
		(*C.spx_int16_t)(&out[0]),
		&outLen,
	)
	if res != 0 {
		return 0, errors.New(C.GoString(C.speex_resampler_strerror(res)))
	}

	return int(outLen) * r.channels, nil
}

func gcd(a, b int) int {
	for b != 0 {
		a, b = b, a%b
	}
	return a
}
