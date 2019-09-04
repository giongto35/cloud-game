package client

import (
	"sync/atomic"
)

// AtomicBool is an atomic boolean struct
type AtomicBool struct {
	n int32
}

// NewAtomicBool creates a new instance of AtomicBool
func NewAtomicBool(initiallyTrue bool) *AtomicBool {
	var n int32
	if initiallyTrue {
		n = 1
	}
	return &AtomicBool{n: n}
}

// SetToTrue sets this value to true
func (b *AtomicBool) SetToTrue() {
	atomic.StoreInt32(&b.n, 1)
}

// SetToFalse sets this value to false
func (b *AtomicBool) SetToFalse() {
	atomic.StoreInt32(&b.n, 0)
}

// True returns true if it is set to true
func (b *AtomicBool) True() bool {
	return atomic.LoadInt32(&b.n) != int32(0)
}

// False return true if it is set to false
func (b *AtomicBool) False() bool {
	return atomic.LoadInt32(&b.n) == int32(0)
}
