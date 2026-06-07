package webrtc

import (
	"sync"

	"github.com/pion/webrtc/v4"
)

type IceCandidateBuffer struct {
	mu  sync.Mutex
	buf []webrtc.ICECandidateInit
}

func (b *IceCandidateBuffer) push(c webrtc.ICECandidateInit) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, c)
}

// FlushAll atomically swaps the internal buffer with a new, empty one
// and returns the old buffer's contents for processing.
func (b *IceCandidateBuffer) FlushAll() []webrtc.ICECandidateInit {
	b.mu.Lock()
	defer b.mu.Unlock()
	oldBuffer := b.buf
	b.buf = nil // Or b.buf = make([]Candidate, 0, initialCapacity)
	return oldBuffer
}

func (b *IceCandidateBuffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = nil
}
