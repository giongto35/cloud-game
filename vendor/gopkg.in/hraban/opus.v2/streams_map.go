// Copyright © 2015-2017 Go Opus Authors (see AUTHORS file)
//
// License for use of this code is detailed in the LICENSE file

package opus

import (
	"sync"
	"sync/atomic"
)

// A map of simple integers to the actual pointers to stream structs. Avoids
// passing pointers into the Go heap to C.
//
// As per the CGo pointers design doc for go 1.6:
//
// A particular unsafe area is C code that wants to hold on to Go func and
// pointer values for future callbacks from C to Go. This works today but is not
// permitted by the invariant. It is hard to detect. One safe approach is: Go
// code that wants to preserve funcs/pointers stores them into a map indexed by
// an int. Go code calls the C code, passing the int, which the C code may store
// freely. When the C code wants to call into Go, it passes the int to a Go
// function that looks in the map and makes the call. An explicit call is
// required to release the value from the map if it is no longer needed, but
// that was already true before.
//
// - https://github.com/golang/proposal/blob/master/design/12416-cgo-pointers.md
type streamsMap struct {
	sync.RWMutex
	m       map[uintptr]*Stream
	counter uintptr
}

func (sm *streamsMap) Get(id uintptr) *Stream {
	sm.RLock()
	defer sm.RUnlock()
	return sm.m[id]
}

func (sm *streamsMap) Del(s *Stream) {
	sm.Lock()
	defer sm.Unlock()
	delete(sm.m, s.id)
}

// NextId returns a unique ID for each call.
func (sm *streamsMap) NextId() uintptr {
	return atomic.AddUintptr(&sm.counter, 1)
}

func (sm *streamsMap) Save(s *Stream) {
	sm.Lock()
	defer sm.Unlock()
	sm.m[s.id] = s
}

func newStreamsMap() *streamsMap {
	return &streamsMap{
		counter: 0,
		m:       map[uintptr]*Stream{},
	}
}
