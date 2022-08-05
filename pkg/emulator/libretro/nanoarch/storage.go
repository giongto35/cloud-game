package nanoarch

import (
	"io/ioutil"
	"path/filepath"
)

type (
	Storage interface {
		GetSavePath() string
		GetSRAMPath() string
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
	}
)

func (s *StateStorage) GetSavePath() string                { return filepath.Join(s.Path, s.MainSave+".dat") }
func (s *StateStorage) GetSRAMPath() string                { return filepath.Join(s.Path, s.MainSave+".srm") }
func (s *StateStorage) Load(path string) ([]byte, error)   { return ioutil.ReadFile(path) }
func (s *StateStorage) Save(path string, dat []byte) error { return ioutil.WriteFile(path, dat, 0644) }
