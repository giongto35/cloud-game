package opus

type Buffer struct {
	Data []int16
	idx  int
}

func (b *Buffer) Write(samples []int16) (written int) {
	w := copy(b.Data[b.idx:], samples)
	b.idx += w
	return w
}

func (b *Buffer) Full() bool {
	full := b.idx == len(b.Data)
	if full {
		b.idx = 0
	}
	return full
}
