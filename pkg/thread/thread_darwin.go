// This package used for locking goroutines to
// the main OS thread.
// See: https://github.com/golang/go/wiki/LockOSThread
package thread

// Wrap enables functions to be executed in the main thread.
func Wrap(f func()) { Run(f) }

// Main calls a function on the main thread.
func Main(f func()) { Call(f) }
