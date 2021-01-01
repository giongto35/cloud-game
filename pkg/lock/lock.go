package lock

import "time"

type TimeLock struct {
	l chan struct{}
}

func NewLock() *TimeLock {
	return &TimeLock{
		l: make(chan struct{}, 1),
	}
}

func (tl *TimeLock) Lock() {
	<-tl.l
}

func (tl *TimeLock) Unlock() {
	tl.l <- struct{}{}
}

func (tl *TimeLock) LockFor(d time.Duration) {
	select {
	case <-tl.l:
	case <-time.After(d):
	}
}
