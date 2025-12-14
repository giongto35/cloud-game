package media

import (
	"errors"
	"slices"
)

type ResampleAlgo uint8

const (
	ResampleNearest ResampleAlgo = iota
	ResampleLinear
	ResampleSpeex
)

type buffer struct {
	in, out   samples
	frames    []float32
	resampler *Resampler
	srcHz     int
	dstHz     int
	fi        int
	p         int
	algo      ResampleAlgo
}

func newBuffer(frames []float32, hz int) (*buffer, error) {
	if hz < 2000 || len(frames) == 0 {
		return nil, errors.New("invalid params")
	}

	// frames should be sorted ascending, largest last
	frames = slices.Clone(frames)
	slices.Sort(frames)

	maxSize := stereoSamples(hz, frames[len(frames)-1])

	return &buffer{
		in:     make(samples, maxSize),
		out:    make(samples, maxSize),
		frames: frames,
		srcHz:  hz,
		dstHz:  hz,
		fi:     len(frames) - 1, // start with largest
	}, nil
}

func (b *buffer) close() {
	if b.resampler != nil {
		b.resampler.Destroy()
	}
}

func (b *buffer) resample(targetHz int, algo ResampleAlgo) error {
	b.algo, b.dstHz = algo, targetHz
	b.out = make(samples, stereoSamples(targetHz, b.frames[len(b.frames)-1]))

	if algo == ResampleSpeex {
		var err error
		b.resampler, err = NewResampler(2, b.srcHz, targetHz, QualityMax)
		return err
	}
	return nil
}

func (b *buffer) write(s samples, onFull func(samples, float32)) {
	for len(s) > 0 {
		srcSize := stereoSamples(b.srcHz, b.frames[b.fi])

		n := copy(b.in[b.p:srcSize], s)
		if n == 0 {
			// oof
			break
		}

		s = s[n:]
		b.p += n

		if b.p >= srcSize {
			onFull(b.stretch(srcSize), b.frames[b.fi])
			b.p = 0
			b.choose(len(s))
		}
	}
	// Remaining samples stay in buffer, will be completed on next write
}

func (b *buffer) choose(remaining int) {
	// Find the largest bucket that fits in remaining samples
	for i := len(b.frames) - 1; i >= 0; i-- {
		if remaining >= stereoSamples(b.srcHz, b.frames[i]) {
			b.fi = i
			return
		}
	}
	// Nothing fits - use smallest and wait for more data
	b.fi = 0
}

func (b *buffer) stretch(srcSize int) samples {
	dstSize := stereoSamples(b.dstHz, b.frames[b.fi])
	src, out := b.in[:srcSize], b.out[:dstSize]

	if srcSize == dstSize {
		return src
	}

	switch b.algo {
	case ResampleSpeex:
		if n, _ := b.resampler.Process(src, out); n == dstSize {
			return out
		}
		fallthrough
	case ResampleLinear:
		return linear(src, out)
	case ResampleNearest:
		return nearest(src, out)
	}
	return src
}

func linear(src, out samples) samples {
	sn, dn := len(src)/2, len(out)/2
	if sn < 2 || dn < 2 {
		return out
	}
	ratio := ((sn - 1) << 16) / (dn - 1)
	for i := 0; i < dn; i++ {
		pos := i * ratio
		si, frac := (pos>>16)*2, pos&0xFFFF
		di := i * 2
		if si >= len(src)-2 {
			out[di], out[di+1] = src[len(src)-2], src[len(src)-1]
		} else {
			out[di] = int16(int32(src[si]) + ((int32(src[si+2])-int32(src[si]))*int32(frac))>>16)
			out[di+1] = int16(int32(src[si+1]) + ((int32(src[si+3])-int32(src[si+1]))*int32(frac))>>16)
		}
	}
	return out
}

func nearest(src, out samples) samples {
	sn, dn := len(src)/2, len(out)/2
	if sn < 2 || dn < 2 {
		return out
	}
	for i := 0; i < dn; i++ {
		si, di := (i*sn/dn)*2, i*2
		out[di], out[di+1] = src[si], src[si+1]
	}
	return out
}

func stereoSamples(hz int, ms float32) int {
	return int(float32(hz)*ms/1000+0.5) * 2
}
