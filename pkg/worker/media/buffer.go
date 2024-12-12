package media

import (
	"errors"
	"math"
	"unsafe"
)

// buffer is a simple non-concurrent safe buffer for audio samples.
type buffer struct {
	stretch bool
	frameHz []int

	raw     samples
	buckets []Bucket
	cur     *Bucket
}

type Bucket struct {
	mem samples
	ms  float32
	lv  int
	dst int
}

func newBuffer(frames []float32, hz int) (*buffer, error) {
	if hz < 2000 {
		return nil, errors.New("hz should be > than 2000")
	}

	buf := buffer{}

	// preallocate continuous array
	s := 0
	for _, f := range frames {
		s += frame(hz, f)
	}
	buf.raw = make(samples, s)

	next := 0
	for _, f := range frames {
		s := frame(hz, f)
		buf.buckets = append(buf.buckets, Bucket{
			mem: buf.raw[next : next+s],
			ms:  f,
		})
		next += s
	}
	buf.cur = &buf.buckets[len(buf.buckets)-1]
	return &buf, nil
}

func (b *buffer) choose(l int) {
	for _, bb := range b.buckets {
		if l >= len(bb.mem) {
			b.cur = &bb
			break
		}
	}
}

func (b *buffer) resample(hz int) {
	b.stretch = true
	for i := range b.buckets {
		b.buckets[i].dst = frame(hz, float32(b.buckets[i].ms))
	}
}

// write fills the buffer until it's full and then passes the gathered data into a callback.
//
// There are two cases to consider:
// 1. Underflow, when the length of the written data is less than the buffer's available space.
// 2. Overflow, when the length exceeds the current available buffer space.
//
// We overwrite any previous values in the buffer and move the internal write pointer
// by the length of the written data.
// In the first case, we won't call the callback, but it will be called every time
// when the internal buffer overflows until all samples are read.
func (b *buffer) write(s samples, onFull func(samples, float32)) (r int) {
	for r < len(s) {
		buf := b.cur
		w := copy(buf.mem[buf.lv:], s[r:])
		r += w
		buf.lv += w
		if buf.lv == len(buf.mem) {
			if b.stretch {
				onFull(buf.mem.stretch(buf.dst), buf.ms)
			} else {
				onFull(buf.mem, buf.ms)
			}
			b.choose(len(s) - r)
			b.cur.lv = 0
		}
	}
	return
}

// frame calculates an audio stereo frame size, i.e. 48k*frame/1000*2
// with round(x / 2) * 2 for the closest even number
func frame(hz int, frame float32) int {
	return int(math.Round(float64(hz)*float64(frame)/1000/2) * 2 * 2)
}

// stretch does a simple stretching of audio samples.
// something like: [1,2,3,4,5,6] -> [1,2,x,x,3,4,x,x,5,6,x,x] -> [1,2,1,2,3,4,3,4,5,6,5,6]
func (s samples) stretch(size int) []int16 {
	out := buf[:size]
	n := len(s)
	ratio := float32(size) / float32(n)
	sPtr := unsafe.Pointer(&s[0])
	for i, l, r := 0, 0, 0; i < n; i += 2 {
		l, r = r, int(float32((i+2)>>1)*ratio)<<1 // index in src * ratio -> approximated index in dst *2 due to int16
		for j := l; j < r; j += 2 {
			*(*int32)(unsafe.Pointer(&out[j])) = *(*int32)(sPtr) // out[j] = s[i]; out[j+1] = s[i+1]
		}
		sPtr = unsafe.Add(sPtr, uintptr(4))
	}
	return out
}
