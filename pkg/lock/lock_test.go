package lock

import (
	"testing"
	"time"
)

func TestLock(t *testing.T) {
	a := 1
	lock := NewLock()
	wait := time.Millisecond * 10

	lock.Unlock()
	lock.Unlock()
	lock.Unlock()

	go func(timeLock *TimeLock) {
		time.Sleep(time.Second * 1)
		lock.Unlock()
	}(lock)

	lock.LockFor(time.Second * 30)
	lock.LockFor(wait)
	lock.LockFor(wait)
	lock.LockFor(wait)
	lock.LockFor(wait)
	lock.LockFor(time.Millisecond * 10)
	go func(timeLock *TimeLock) {
		time.Sleep(time.Millisecond * 1)
		lock.Unlock()
	}(lock)
	lock.Lock()

	a -= 1
	if a != 0 {
		t.Errorf("lock test failed because a != 0")
	}
}
