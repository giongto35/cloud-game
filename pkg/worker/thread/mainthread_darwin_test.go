package thread

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	Wrap(func() { os.Exit(m.Run()) })
}

func TestMainThread(t *testing.T) {
	_ = 10
}
