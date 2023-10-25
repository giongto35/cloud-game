package nanoarch

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestLimit(t *testing.T) {
	c := atomic.Int32{}
	lim := NewLimit(50 * time.Millisecond)

	for i := 0; i < 10; i++ {
		lim(func() {
			c.Add(1)
		})
	}

	if c.Load() > 1 {
		t.Errorf("should be just 1")
	}
}
