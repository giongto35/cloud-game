package nanoarch

import (
	"path/filepath"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/compression"
)

type (
	ZipStorage struct {
		*StateStorage
	}
	ZipReaderWriter struct {
		ReaderWriter
	}
)

const zip = ".zip"

func NewZipStorage(store *StateStorage) *ZipStorage {
	store.rw = &ZipReaderWriter{store.rw}
	return &ZipStorage{StateStorage: store}
}

func (z *ZipStorage) GetSavePath() string { return z.StateStorage.GetSavePath() + zip }
func (z *ZipStorage) GetSRAMPath() string { return z.StateStorage.GetSRAMPath() + zip }

// Write writes the state to a file with the path.
func (zrw *ZipReaderWriter) Write(path string, data []byte) error {
	_, name := filepath.Split(path)
	if name == "" || name == "." {
		return compression.ErrorInvalidName
	}
	name = strings.TrimSuffix(name, zip)
	compress, err := compression.Compress(data, name)
	if err != nil {
		return err
	}
	return zrw.ReaderWriter.Write(path, compress)
}

// Read reads the state from a file with the path.
func (zrw *ZipReaderWriter) Read(path string) ([]byte, error) {
	data, err := zrw.ReaderWriter.Read(path)
	if err != nil {
		return nil, err
	}
	d, _, err := compression.Read(data)
	if err != nil {
		return nil, err
	}
	return d, nil
}
