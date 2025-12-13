package media

import "errors"

type ResampleAlgo uint8

const (
	ResampleNearest ResampleAlgo = iota
	ResampleLinear
)

// preallocated scratch buffer for resampling output
// size for max Opus frame: 60ms at 48kHz stereo = 48000 * 0.06 * 2 = 5760 samples
var stretchBuf = make(samples, 5760)

// buffer is a simple non-concurrent safe buffer for audio samples.
type buffer struct {
	useResample bool
	algo        ResampleAlgo
	srcHz       int

	raw samples

	buckets []bucket
	bi      int
}

type bucket struct {
	mem samples
	ms  float32
	p   int
	dst int
}

func newBuffer(frames []float32, hz int) (*buffer, error) {
	if hz < 2000 {
		return nil, errors.New("hz should be > 2000")
	}
	if len(frames) == 0 {
		return nil, errors.New("frames list is empty")
	}

	buf := buffer{srcHz: hz}

	totalSize := 0
	for _, f := range frames {
		totalSize += frameStereoSamples(hz, f)
	}

	if totalSize == 0 {
		return nil, errors.New("calculated buffer size is 0, check params")
	}

	buf.raw = make(samples, totalSize)

	// map buckets to the raw continuous array
	offset := 0
	for _, f := range frames {
		size := frameStereoSamples(hz, f)
		buf.buckets = append(buf.buckets, bucket{
			mem: buf.raw[offset : offset+size],
			ms:  f,
		})
		offset += size
	}

	// start with the largest bucket (last one, assuming frames are sorted ascending)
	buf.bi = len(buf.buckets) - 1

	return &buf, nil
}

// cur returns the current bucket pointer
func (b *buffer) cur() *bucket { return &b.buckets[b.bi] }

// choose selects the best bucket for the remaining samples.
// It picks the largest bucket that can be completely filled.
// Buckets should be sorted by size ascending for this to work optimally.
func (b *buffer) choose(remaining int) {
	// search from largest to smallest
	for i := len(b.buckets) - 1; i >= 0; i-- {
		if remaining >= len(b.buckets[i].mem) {
			b.bi = i
			return
		}
	}
	// fall back to smallest bucket if remaining < all bucket sizes
	b.bi = 0
}

// resample enables resampling to target Hz with specified algorithm
func (b *buffer) resample(targetHz int, algo ResampleAlgo) {
	b.useResample = true
	b.algo = algo
	for i := range b.buckets {
		b.buckets[i].dst = frameStereoSamples(targetHz, b.buckets[i].ms)
	}
}

// stretch applies the selected resampling algorithm
func (b *buffer) stretch(src samples, dstSize int) samples {
	switch b.algo {
	case ResampleNearest:
		return stretchNearest(src, dstSize)
	case ResampleLinear:
		return stretchLinear(src, dstSize)
	default:
		return stretchLinear(src, dstSize)
	}
}

// write fills the buffer and calls onFull when a complete frame is ready.
// returns the number of samples consumed.
func (b *buffer) write(s samples, onFull func(samples, float32)) int {
	read := 0
	for read < len(s) {
		cur := b.cur()

		// copy all samples into current bucket
		n := copy(cur.mem[cur.p:], s[read:])
		read += n
		cur.p += n

		// bucket is full - emit frame
		if cur.p == len(cur.mem) {
			if b.useResample {
				onFull(b.stretch(cur.mem, cur.dst), cur.ms)
			} else {
				onFull(cur.mem, cur.ms)
			}

			// select next bucket and reset write position
			b.choose(len(s) - read)
			b.cur().p = 0
		}
	}
	return read
}

// frameStereoSamples calculates stereo frame size in samples.
// e.g., 48000 Hz * 20ms = 960 samples/channel * 2 channels = 1920 total samples
func frameStereoSamples(hz int, ms float32) int {
	samplesPerChannel := int(float32(hz)*ms/1000 + 0.5) // round to nearest
	return samplesPerChannel * 2                        // stereo
}

// stretchLinear resamples stereo audio using linear interpolation.
func stretchLinear(src samples, dstSize int) samples {
	srcLen := len(src)
	if srcLen < 2 || dstSize < 2 {
		return stretchBuf[:dstSize]
	}

	out := stretchBuf[:dstSize]

	srcPairs := srcLen / 2
	dstPairs := dstSize / 2

	// Fixed-point ratio for precision (16.16 fixed point)
	ratio := ((srcPairs - 1) << 16) / (dstPairs - 1)

	for i := 0; i < dstPairs; i++ {
		// Calculate source position in fixed-point
		pos := i * ratio
		srcIdx := pos >> 16
		frac := pos & 0xFFFF

		dstIdx := i * 2

		if srcIdx >= srcPairs-1 {
			// Last sample - no interpolation
			out[dstIdx] = src[srcLen-2]
			out[dstIdx+1] = src[srcLen-1]
		} else {
			// Linear interpolation for both channels
			srcBase := srcIdx * 2

			// Left channel
			l0 := int32(src[srcBase])
			l1 := int32(src[srcBase+2])
			out[dstIdx] = int16(l0 + ((l1-l0)*int32(frac))>>16)

			// Right channel
			r0 := int32(src[srcBase+1])
			r1 := int32(src[srcBase+3])
			out[dstIdx+1] = int16(r0 + ((r1-r0)*int32(frac))>>16)
		}
	}

	return out
}

// stretchNearest is a faster nearest-neighbor version if quality isn't critical
func stretchNearest(src samples, dstSize int) samples {
	srcLen := len(src)
	if srcLen < 2 || dstSize < 2 {
		return stretchBuf[:dstSize]
	}

	out := stretchBuf[:dstSize]

	srcPairs := srcLen / 2
	dstPairs := dstSize / 2

	for i := 0; i < dstPairs; i++ {
		srcIdx := (i * srcPairs / dstPairs) * 2
		dstIdx := i * 2
		out[dstIdx] = src[srcIdx]
		out[dstIdx+1] = src[srcIdx+1]
	}

	return out
}
