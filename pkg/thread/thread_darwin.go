// This package used for locking goroutines to
// the main OS thread.
// See: https://github.com/golang/go/wiki/LockOSThread
package thread

// MainWrapMaybe enables functions to be executed in the main thread.
// Enabled for macOS only.
func MainWrapMaybe(f func()) { Run(f) }

// MainMaybe calls a function on the main thread.
// Enabled for macOS only.
func MainMaybe(f func()) { Call(f) }
