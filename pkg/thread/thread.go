// This package used for locking goroutines to
// the main OS thread.
// See: https://github.com/golang/go/wiki/LockOSThread
package thread

import (
	"runtime"

	"github.com/faiface/mainthread"
)

var isMacOs = runtime.GOOS == "darwin"

// MainWrapMaybe enables functions to be executed in the main thread.
// Enabled for macOS only.
func MainWrapMaybe(f func()) {
	if isMacOs {
		mainthread.Run(f)
	} else {
		f()
	}
}

// MainMaybe calls a function on the main thread.
// Enabled for macOS only.
func MainMaybe(f func()) {
	if isMacOs {
		mainthread.Call(f)
	} else {
		f()
	}
}
