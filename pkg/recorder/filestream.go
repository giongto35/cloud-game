package recorder

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
)

type fileStream struct {
	io.Closer
	Stream

	f  *os.File
	w  *bufio.Writer
	mu *sync.Mutex
}

func newFileStream(dir string, name string) (*fileStream, error) {
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &fileStream{f: f, w: bufio.NewWriter(f), mu: &sync.Mutex{}}, nil
}

func (f *fileStream) Flush() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.w.Flush()
}

func (f *fileStream) Close() error { return f.f.Close() }

func (f *fileStream) Size() (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	inf, err := f.f.Stat()
	if err != nil {
		return -1, err
	}
	return inf.Size(), nil
}

func (f *fileStream) Write(data []byte) error {
	f.mu.Lock()
	n, err := f.w.Write(data)
	f.mu.Unlock()
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
func (f *fileStream) WriteAtStart(data []byte) error {
	if _, err := f.f.Seek(0, 0); err != nil {
		return err
	}
	return f.Write(data)
}

func (f *fileStream) WriteString(s string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.w.WriteString(s)
}
