package media

// Buffer is a simple non-thread safe ring buffer for audio samples.
// It should be used for 16bit PCM (LE interleaved) data.
type (
	Buffer struct {
		s  Samples
		wi int
	}
	OnFull  func(s Samples)
	Samples []int16
)

func NewBuffer(numSamples int) Buffer { return Buffer{s: make(Samples, numSamples)} }

// Write fills the buffer with data calling a callback function when
// the internal buffer fills out.
//
// Consider two cases:
//
// 1. Underflow, when the length of written data is less than the buffer's available space.
// 2. Overflow, when the length exceeds the current available buffer space.
// In the both cases we overwrite any previous values in the buffer and move the internal
// write pointer on the length of written data.
// In the first case we won't call the callback, but it will be called every time
// when the internal buffer overflows until all samples are read.
func (b *Buffer) Write(s Samples, onFull OnFull) (r int) {
	for r < len(s) {
		w := copy(b.s[b.wi:], s[r:])
		r += w
		b.wi += w
		if b.wi == len(b.s) {
			b.wi = 0
			if onFull != nil {
				onFull(b.s)
			}
		}
	}
	return
}
