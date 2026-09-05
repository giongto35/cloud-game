package nanoarch

import (
	"os"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimit(t *testing.T) {
	c := atomic.Int32{}
	lim := NewLimit(50 * time.Millisecond)

	for range 10 {
		lim(func() {
			c.Add(1)
		})
	}

	if c.Load() > 1 {
		t.Errorf("should be just 1")
	}
}

func TestSaveDirStateAfterRelease(t *testing.T) {
	local := t.TempDir()
	n := NewNano(local)

	if n.cSaveDirectory == nil || n.cSystemDirectory == nil || n.cUserName == nil {
		t.Fatal("should allocate the C strings")
	}
	saveDir := local + "/legacy_save"

	if err := os.MkdirAll(saveDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := n.SaveDir(); got != saveDir {
		t.Fatalf("SaveDir() = %q, want %q", got, saveDir)
	}
	if err := n.DeleteSaveDir(); err != nil {
		t.Fatalf("DeleteSaveDir before release failed: %v", err)
	}
	if _, err := os.Stat(saveDir); !os.IsNotExist(err) {
		t.Fatalf("save dir should have been removed, stat err = %v", err)
	}

	n.freeCStrings()

	if n.cSaveDirectory != nil || n.cSystemDirectory != nil || n.cUserName != nil {
		t.Fatal("freed C string fields must be cleared to nil")
	}

	if err := n.DeleteSaveDir(); err != nil {
		t.Errorf("DeleteSaveDir after release should be a no-op, got %v", err)
	}
	if got := n.SaveDir(); got != "" {
		t.Errorf("SaveDir() after release should be empty, got %q", got)
	}
}
