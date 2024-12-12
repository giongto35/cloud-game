package media

import "fmt"

type buffer2 struct {
	s       samples
	wi      int
	dst     int
	stretch bool
	frameHz []int

	dstHz2 int

	buckets []Bucket
	cur     *Bucket

	sym bool
}

type Bucket struct {
	mem samples
	vol int
	lv  int
}

func NewBucket(level int, size int) Bucket {
	return Bucket{
		mem: make(samples, size),
		vol: level,
	}
}

func (b *Bucket) Reset() {
	b.lv = 0
}

func (b *Bucket) IsEmpty() bool {
	return b.lv == 0
}

var frames = [...]int{10, 5}

func newOpusBuffer(hz int) buffer2 {
	buf := buffer2{}

	var fz = make([]int, 3)
	sum := 0
	for i, f := range frames {
		sum += f
		fz[i] = frame(hz, float32(f))
		buf.buckets = append(buf.buckets, NewBucket(f, fz[i]))
	}
	buf.cur = &buf.buckets[0]

	//buf.enableStretch(frame(hz, float32(buf.cur.vol)))

	return buf
}

func (b *buffer2) chooseBucket(l int) {
	//b.cur = &b.buckets[0]
	for _, bb := range b.buckets {
		if l >= len(bb.mem) {
			b.cur = &bb
			b.sym = false
			if b.stretch {
				b.enableStretch(frame(b.dstHz2, float32(b.cur.vol)))
			}
			break
		}
	}
}

// enableStretch adds a simple stretching of buffer to a desired size before
// the onFull callback call.
func (b *buffer2) enableStretch(l int) { b.stretch = true; b.dst = frame(b.dstHz2, float32(b.cur.vol)) }

func (b *buffer2) dstHz(hz int) {
	b.dstHz2 = hz
	b.enableStretch(frame(hz, float32(b.cur.vol)))
}

func (b *buffer2) write(s samples, onFull func(samples, int)) (r int) {
	// select bucket
	//b.chooseBucket(len(s))
	for r < len(s) {
		buf := b.cur

		w := copy(buf.mem[buf.lv:], s[r:])
		r += w
		buf.lv += w
		if buf.lv == len(buf.mem) {
			b.sym = true
			if b.stretch {
				onFull(buf.mem.stretch(b.dst), buf.vol)
			} else {
				onFull(buf.mem, buf.vol)
			}
			if !b.sym {
				fmt.Printf(">>>>>>>>>")
			}
			b.chooseBucket(len(s) - r)
			b.cur.Reset()
		}
	}
	return
}
