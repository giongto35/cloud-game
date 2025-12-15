package media

import (
	"errors"

	"github.com/giongto35/cloud-game/v3/pkg/resampler"
)

type ResampleAlgo uint8

const (
	ResampleNearest ResampleAlgo = iota
	ResampleLinear
	ResampleSpeex
)

type buffer struct {
	raw     samples
	scratch samples
	buckets []bucket
	srcHz   int
	dstHz   int
	bi      int
	algo    ResampleAlgo

	resampler *resampler.Resampler
}

type bucket struct {
	mem samples
	ms  float32
	p   int
	dst int
}

func newBuffer(frames []float32, hz int) (*buffer, error) {
	if hz < 2000 || len(frames) == 0 {
		return nil, errors.New("invalid params")
	}

	buckets := make([]bucket, len(frames))
	var total int
	for i, ms := range frames {
		n := stereoSamples(hz, ms)
		buckets[i] = bucket{ms: ms, dst: n}
		total += n
	}
	if total == 0 {
		return nil, errors.New("zero buffer size")
	}

	raw := make(samples, total)
	for i, off := 0, 0; i < len(buckets); i++ {
		buckets[i].mem = raw[off : off+buckets[i].dst]
		off += buckets[i].dst
	}

	return &buffer{
		raw:     raw,
		scratch: make(samples, 5760),
		buckets: buckets,
		srcHz:   hz,
		dstHz:   hz,
		bi:      len(buckets) - 1,
	}, nil
}

func (b *buffer) close() {
	if b.resampler != nil {
		b.resampler.Destroy()
		b.resampler = nil
	}
}

func (b *buffer) resample(hz int, algo ResampleAlgo) error {
	b.algo, b.dstHz = algo, hz
	for i := range b.buckets {
		b.buckets[i].dst = stereoSamples(hz, b.buckets[i].ms)
	}
	if algo == ResampleSpeex {
		var err error
		b.resampler, err = resampler.Init(2, b.srcHz, hz, resampler.QualityMax)
		return err
	}
	return nil
}

func (b *buffer) write(s samples, onFull func(samples, float32)) int {
	n := len(s)
	for i := 0; i < n; {
		cur := &b.buckets[b.bi]
		c := copy(cur.mem[cur.p:], s[i:])
		i += c
		cur.p += c
		if cur.p == len(cur.mem) {
			onFull(b.stretch(cur.mem, cur.dst), cur.ms)
			b.choose(n - i)
			b.buckets[b.bi].p = 0
		}
	}
	return n
}

func (b *buffer) choose(rem int) {
	for i := len(b.buckets) - 1; i >= 0; i-- {
		if rem >= len(b.buckets[i].mem) {
			b.bi = i
			return
		}
	}
	b.bi = 0
}

func (b *buffer) stretch(src samples, size int) samples {
	if len(src) == size {
		return src
	}

	if cap(b.scratch) < size {
		b.scratch = make(samples, size)
	}
	out := b.scratch[:size]

	if b.algo == ResampleSpeex && b.resampler != nil {
		if n, _ := b.resampler.Process(out, src); n > 0 {
			for i := n; i < size; i += 2 {
				out[i], out[i+1] = out[n-2], out[n-1]
			}
			return out
		}
	}

	if b.algo == ResampleNearest {
		resampler.Nearest(out, src)
	} else {
		resampler.Linear(out, src)
	}
	return out
}

func stereoSamples(hz int, ms float32) int {
	return int(float32(hz)*ms/1000+0.5) * 2
}
