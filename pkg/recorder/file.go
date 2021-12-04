package recorder

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

var defaultBufferSize = 4096

type file struct {
	io.Closer
	sync.Mutex

	f *os.File
	w *bufio.Writer
}

func newFile(dir string, name string) (*file, error) {
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &file{f: f, w: bufio.NewWriterSize(f, defaultBufferSize)}, nil
}

func (f *file) Flush() error {
	f.Lock()
	defer f.Unlock()
	return f.w.Flush()
}

func (f *file) Close() error { return f.f.Close() }

func (f *file) Size() (int64, error) {
	f.Lock()
	defer f.Unlock()
	inf, err := f.f.Stat()
	if err != nil {
		return -1, err
	}
	return inf.Size(), nil
}

func (f *file) Write(data []byte) error {
	f.Lock()
	n, err := f.w.Write(data)
	f.Unlock()
	if err != nil {
		if n < len(data) {
			return fmt.Errorf("write size mismatch [%v!=%v], %v", n, len(data), err)
		}
		return err
	}
	return nil
}

// WriteAtStart writes data into beginning of the file.
// Make sure that underling file doesn't use the O_APPEND directive.
func (f *file) WriteAtStart(data []byte) error {
	if _, err := f.f.Seek(0, 0); err != nil {
		return err
	}
	return f.Write(data)
}

func (f *file) WriteString(s string) (int, error) {
	f.Lock()
	defer f.Unlock()
	return f.w.WriteString(s)
}
