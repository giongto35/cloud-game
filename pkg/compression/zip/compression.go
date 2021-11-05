package zip

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

const Ext = ".zip"

var (
	ErrorNotFound    = errors.New("not found")
	ErrorInvalidName = errors.New("invalid name")
)

type Extractor struct {
	log *logger.Logger
}

func New(log *logger.Logger) Extractor {
	return Extractor{
		log: log,
	}
}

// Compress compresses the bytes (a single file) with a name specified into a ZIP file (as bytes).
func Compress(data []byte, name string) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	//w.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
	//	return flate.NewWriter(out, flate.BestCompression)
	//})

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

func (e Extractor) Extract(src string, dest string) (files []string, err error) {
	r, err := zip.OpenReader(src)
	if err != nil {
		return files, err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)

		// negate ZipSlip vulnerability (http://bit.ly/2MsjAWE)
		if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
			log.Printf("warning: %s is illegal path", path)
			continue
		}
		// remake directory
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(path, os.ModePerm); err != nil {
				log.Printf("error: %v", err)
			}
			continue
		}
		// make file
		if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
			log.Printf("error: %v", err)
			continue
		}
		out, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			log.Printf("error: %v", err)
			continue
		}
		rc, err := f.Open()
		if err != nil {
			log.Printf("error: %v", err)
			continue
		}

		if _, err = io.Copy(out, rc); err != nil {
			log.Printf("error: %v", err)
			_ = out.Close()
			_ = rc.Close()
			continue
		}

		_ = out.Close()
		_ = rc.Close()

		files = append(files, path)
	}
	return files, nil
}
