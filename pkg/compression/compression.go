package compression

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
)

var (
	ErrorNotFound    = errors.New("not found")
	ErrorInvalidName = errors.New("invalid name")
)

// Compress compresses the bytes (a single file) with a name specified into a ZIP file (as bytes).
func Compress(data []byte, name string) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	z, err := w.Create(name)
	if err != nil {
		return nil, err
	}
	_, err = z.Write(data)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Read reads a single ZIP file from the bytes array.
// It will return un-compressed data and the name of that file.
func Read(zd []byte) ([]byte, string, error) {
	r, err := zip.NewReader(bytes.NewReader(zd), int64(len(zd)))
	if err != nil {
		return nil, "", err
	}
	for _, f := range r.File {
		if f.FileInfo().IsDir() {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, "", err
		}
		b, err := io.ReadAll(rc)
		if err != nil {
			return nil, "", err
		}
		if err := rc.Close(); err != nil {
			return nil, "", err
		}
		return b, f.FileInfo().Name(), nil
	}
	return nil, "", ErrorNotFound
}
