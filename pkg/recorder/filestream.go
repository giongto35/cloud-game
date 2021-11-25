package recorder

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type fileStream struct {
	io.Closer
	bufio.Writer
	Stream

	f *os.File
}

func newFileStream(dir string, name string) (*fileStream, error) {
	f, err := os.OpenFile(filepath.Join(dir, name), os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &fileStream{f: f, Writer: *bufio.NewWriter(f)}, nil
}

func (f *fileStream) Close() error { return f.f.Close() }

func (f *fileStream) Size() (int64, error) {
	inf, err := f.f.Stat()
	if err != nil {
		return -1, err
	}
	return inf.Size(), nil
}

func (f *fileStream) Write(data []byte) error {
	n, err := f.f.Write(data)
	if n < len(data) {
		return fmt.Errorf("write size mismatch [%v!=%v]", n, len(data))
	}
	if err != nil {
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
