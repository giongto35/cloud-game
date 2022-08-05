package nanoarch

import (
	"path/filepath"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/compression/zip"
)

type ZipStorage struct {
	Storage
}

func (z *ZipStorage) GetSavePath() string { return z.Storage.GetSavePath() + zip.Ext }
func (z *ZipStorage) GetSRAMPath() string { return z.Storage.GetSRAMPath() + zip.Ext }

// Load loads a zip file with the path specified.
func (z *ZipStorage) Load(path string) ([]byte, error) {
	data, err := z.Storage.Load(path)
	if err != nil {
		return nil, err
	}
	d, _, err := zip.Read(data)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// Save saves the array of bytes into a file with the specified path.
func (z *ZipStorage) Save(path string, data []byte) error {
	_, name := filepath.Split(path)
	if name == "" || name == "." {
		return zip.ErrorInvalidName
	}
	name = strings.TrimSuffix(name, zip.Ext)
	compress, err := zip.Compress(data, name)
	if err != nil {
		return err
	}
	return z.Storage.Save(path, compress)
}
