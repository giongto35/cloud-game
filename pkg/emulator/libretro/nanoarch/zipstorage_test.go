package nanoarch

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestZipStorage(t *testing.T) {
	testDir := os.TempDir()
	fileName := "test-state"
	destPath := filepath.Join(testDir, fileName) + ".zip"
	expect := []byte{1, 2, 3, 4}
	z := &ZipStorage{
		Storage: &StateStorage{
			Path:     testDir,
			MainSave: fileName,
		},
	}
	if err := z.Save(destPath, expect); err != nil {
		t.Errorf("Zip storage error = %v", err)
	}
	defer func() {
		if err := os.Remove(destPath); err != nil {
			t.Errorf("Zip storage couldn't remove %v", destPath)
		}
	}()
	d, err := z.Load(destPath)
	if err != nil {
		t.Errorf("Zip storage error = %v", err)
	}
	if !reflect.DeepEqual(d, expect) {
		t.Errorf("Zip storage got = %v, want %v", d, expect)
	}
}
