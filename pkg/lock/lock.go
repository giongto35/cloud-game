package lock

import (
	"sync/atomic"
	"time"
)

type TimeLock struct {
	l      chan struct{}
	locked int32
}

// NewLock returns new lock (mutex) with a timeout option.
func NewLock() *TimeLock {
	return &TimeLock{l: make(chan struct{}, 1)}
}

// Lock unconditionally blocks the execution.
func (tl *TimeLock) Lock() {
	if tl.isLocked() {
		return
	}
	tl.lock()
	<-tl.l
}

// LockFor blocks the execution at most for
// the given period of time.
func (tl *TimeLock) LockFor(d time.Duration) {
	tl.lock()
	select {
	case <-tl.l:
	case <-time.After(d):
	}
}

// Unlock removes the current block if any.
func (tl *TimeLock) Unlock() {
	if !tl.isLocked() {
		return
	}
	tl.unlock()
	tl.l <- struct{}{}
}

func (tl *TimeLock) isLocked() bool { return atomic.LoadInt32(&tl.locked) == 1 }
func (tl *TimeLock) lock()          { atomic.StoreInt32(&tl.locked, 1) }
func (tl *TimeLock) unlock()        { atomic.StoreInt32(&tl.locked, 0) }
