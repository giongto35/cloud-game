package os

import (
	"os"
	"path/filepath"

	"github.com/gofrs/flock"
)

type Flock struct {
	f *flock.Flock
}

func NewFileLock(path string) (*Flock, error) {
	if path == "" {
		path = os.TempDir() + string(os.PathSeparator) + "cloud_game.lock"
	}

	if err := os.MkdirAll(filepath.Dir(path), 0770); err != nil {
		return nil, err
	} else {
		f, err := os.Create(path)
		defer func() { _ = f.Close() }()
		if err != nil {
			return nil, err
		}
	}

	f := Flock{
		f: flock.New(path),
	}

	return &f, nil
}

func (f *Flock) Lock() error   { return f.f.Lock() }
func (f *Flock) Unlock() error { return f.f.Unlock() }
