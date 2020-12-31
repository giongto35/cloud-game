package zip

import (
	"archive/zip"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

type Extractor struct{}

func New() Extractor { return Extractor{} }

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
