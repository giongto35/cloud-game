package media

import (
	"errors"

	"github.com/aam335/speexdsp"
)

type ResampleAlgo uint8

const (
	ResampleNearest ResampleAlgo = iota
	ResampleLinear
	ResampleSpeex
)

type buffer struct {
	raw       samples
	scratch   samples
	buckets   []bucket
	resampler *speexdsp.Resampler
	srcHz     int
	dstHz     int
	bi        int
	algo      ResampleAlgo
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

	var totalSize int
	for _, f := range frames {
		totalSize += stereoSamples(hz, f)
	}
	if totalSize == 0 {
		return nil, errors.New("zero buffer size")
	}

	buf := &buffer{
		raw:     make(samples, totalSize),
		scratch: make(samples, 5760),
		srcHz:   hz,
		dstHz:   hz,
	}

	offset := 0
	for _, f := range frames {
		size := stereoSamples(hz, f)
		buf.buckets = append(buf.buckets, bucket{mem: buf.raw[offset : offset+size], ms: f, dst: size})
		offset += size
	}
	buf.bi = len(buf.buckets) - 1

	return buf, nil
}

func (b *buffer) close() {
	if b.resampler != nil {
		b.resampler.Destroy()
		b.resampler = nil
	}
}

func (b *buffer) resample(targetHz int, algo ResampleAlgo) error {
	b.algo = algo
	b.dstHz = targetHz

	for i := range b.buckets {
		b.buckets[i].dst = stereoSamples(targetHz, b.buckets[i].ms)
	}

	if algo == ResampleSpeex {
		var err error
		if b.resampler, err = speexdsp.ResamplerInit(2, b.srcHz, targetHz, speexdsp.QualityDesktop); err != nil {
			return err
		}
	}
	return nil
}

func (b *buffer) write(s samples, onFull func(samples, float32)) int {
	read := 0
	for read < len(s) {
		cur := &b.buckets[b.bi]
		n := copy(cur.mem[cur.p:], s[read:])
		read += n
		cur.p += n

		if cur.p == len(cur.mem) {
			onFull(b.stretch(cur.mem, cur.dst), cur.ms)
			b.choose(len(s) - read)
			b.buckets[b.bi].p = 0
		}
	}
	return read
}

func (b *buffer) choose(remaining int) {
	for i := len(b.buckets) - 1; i >= 0; i-- {
		if remaining >= len(b.buckets[i].mem) {
			b.bi = i
			return
		}
	}
	b.bi = 0
}

func (b *buffer) stretch(src samples, dstSize int) samples {
	switch b.algo {
	case ResampleSpeex:
		if b.resampler != nil {
			if _, out, err := b.resampler.PocessIntInterleaved(src); err == nil {
				if len(out) == dstSize {
					return out
				}
				src = out // use speex output for linear correction
			}
		}
		fallthrough
	case ResampleLinear:
		return b.linear(src, dstSize)
	case ResampleNearest:
		return b.nearest(src, dstSize)
	default:
		return b.linear(src, dstSize)
	}
}

func (b *buffer) linear(src samples, dstSize int) samples {
	srcLen := len(src)
	if srcLen < 2 || dstSize < 2 {
		return b.scratch[:dstSize]
	}

	out := b.scratch[:dstSize]
	srcPairs, dstPairs := srcLen/2, dstSize/2
	ratio := ((srcPairs - 1) << 16) / (dstPairs - 1)

	for i := 0; i < dstPairs; i++ {
		pos := i * ratio
		idx, frac := (pos>>16)*2, pos&0xFFFF
		di := i * 2

		if idx >= srcLen-2 {
			out[di], out[di+1] = src[srcLen-2], src[srcLen-1]
		} else {
			out[di] = int16(int32(src[idx]) + ((int32(src[idx+2])-int32(src[idx]))*int32(frac))>>16)
			out[di+1] = int16(int32(src[idx+1]) + ((int32(src[idx+3])-int32(src[idx+1]))*int32(frac))>>16)
		}
	}
	return out
}

func (b *buffer) nearest(src samples, dstSize int) samples {
	srcLen := len(src)
	if srcLen < 2 || dstSize < 2 {
		return b.scratch[:dstSize]
	}

	out := b.scratch[:dstSize]
	srcPairs, dstPairs := srcLen/2, dstSize/2

	for i := 0; i < dstPairs; i++ {
		si := (i * srcPairs / dstPairs) * 2
		di := i * 2
		out[di], out[di+1] = src[si], src[si+1]
	}
	return out
}

func stereoSamples(hz int, ms float32) int {
	return int(float32(hz)*ms/1000+0.5) * 2
}
