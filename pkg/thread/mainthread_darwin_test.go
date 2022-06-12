package thread

import "testing"

func init() {
	runtime.LockOSThread()
}

func TestMainThread(t *testing.T) {
	value := 0
	fn := func() { value = 1 }
	Main(fn)
	if value != 1 {
		t.Errorf("wrong value %v", value)
	}
}
