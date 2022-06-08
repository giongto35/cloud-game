//go:build !darwin
// +build !darwin

package thread

// MainWrapMaybe enables functions to be executed in the main thread.
// Enabled for macOS only.
func MainWrapMaybe(f func()) { f() }

// MainMaybe calls a function on the main thread.
// Enabled for macOS only.
func MainMaybe(f func()) { f() }
