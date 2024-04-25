//go:build !darwin

package thread

func Wrap(f func())         { f() }
func Main(f func())         { f() }
func SwitchGraphics(s bool) {}
