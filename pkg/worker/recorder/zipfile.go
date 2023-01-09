package recorder

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func compress(source, dest string) (err error) {
	f, err := os.Create(dest + ".zip")
	if err != nil {
		return err
	}
	defer func() { err = f.Close() }()

	// !to handle errors properly
	writer := zip.NewWriter(f)
	defer func() {
		err = writer.Flush()
		err = writer.Close()
	}()

	return filepath.Walk(source, func(path string, info os.FileInfo, err error) (er error) {
		if err != nil {
			return err
		}

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		header.Method = zip.Deflate

		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}
		if info.IsDir() {
			header.Name += "/"
		}

		headerWriter, err := writer.CreateHeader(header)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() { er = f.Close() }()

		_, err = io.Copy(headerWriter, f)
		return err
	})
}
