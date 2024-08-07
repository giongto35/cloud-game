package libretro

import (
	"path/filepath"
	"strings"

	"github.com/giongto35/cloud-game/v3/pkg/os"
	"github.com/giongto35/cloud-game/v3/pkg/worker/compression/zip"
)

type (
	Storage interface {
		GetSavePath() string
		GetSRAMPath() string
		SetMainSaveName(name string)
		SetNonBlocking(v bool)
		Load(path string) ([]byte, error)
		Save(path string, data []byte) error
	}
	StateStorage struct {
		// save path without the dir slash in the end
		Path string
		// contains the name of the main save file
		// e.g. abc<...>293.dat
		// needed for Google Cloud save/restore which
		// doesn't support multiple files
		MainSave string
		NonBlock bool
	}
	ZipStorage struct {
		Storage
	}
)

func (s *StateStorage) SetMainSaveName(name string)      { s.MainSave = name }
func (s *StateStorage) SetNonBlocking(v bool)            { s.NonBlock = v }
func (s *StateStorage) GetSavePath() string              { return filepath.Join(s.Path, s.MainSave+".dat") }
func (s *StateStorage) GetSRAMPath() string              { return filepath.Join(s.Path, s.MainSave+".srm") }
func (s *StateStorage) Load(path string) ([]byte, error) { return os.ReadFile(path) }
func (s *StateStorage) Save(path string, dat []byte) error {
	if s.NonBlock {
		go func() { _ = os.WriteFile(path, dat, 0644) }()
		return nil
	}

	return os.WriteFile(path, dat, 0644)
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
