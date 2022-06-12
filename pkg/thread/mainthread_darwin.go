package thread

import (
	"runtime"
	"sync"
)

type fun struct {
	fn   func()
	done chan struct{}
}

var dPool = sync.Pool{New: func() interface{} { return make(chan struct{}) }}
var fq = make(chan fun, runtime.GOMAXPROCS(0))

func init() {
	runtime.LockOSThread()
}

// Run is a wrapper for the main function.
// Run returns when run (argument) function finishes.
func Run(run func()) {
	done := make(chan struct{})
	go func() {
		run()
		done <- struct{}{}
	}()
	for {
		select {
		case f := <-fq:
			f.fn()
			f.done <- struct{}{}
		case <-done:
			return
		}
	}
}

// Call queues function f on the main thread and blocks until the function f finishes.
func Call(f func()) {
	done := dPool.Get().(chan struct{})
	defer dPool.Put(done)
	fq <- fun{fn: f, done: done}
	<-done
}
