package nanoarch

import (
	"io/ioutil"
	"path/filepath"
)

type (
	ReaderWriter interface {
		Read(path string) ([]byte, error)
		Write(path string, data []byte) error
	}
	Storage interface {
		GetSavePath() string
		GetSRAMPath() string
		Load(path string) ([]byte, error)
		Save(path string, data []byte) error
	}
	// FileReaderWriter reads and writes files as is into a file system.
	FileReaderWriter struct{}
	StateStorage     struct {
		// save path without the dir slash in the end
		Path string
		// contains the name of the main save file
		// e.g. abc<...>293.dat
		// needed for Google Cloud save/restore which
		// doesn't support multiple files
		MainSave string
		// a custom RW function for saves
		rw ReaderWriter
	}
)

func NewStateStorage(path string, main string) *StateStorage {
	return &StateStorage{Path: path, MainSave: main, rw: FileReaderWriter{}}
}

func (s *StateStorage) GetSavePath() string { return filepath.Join(s.Path, s.MainSave+".dat") }
func (s *StateStorage) GetSRAMPath() string { return filepath.Join(s.Path, s.MainSave+".srm") }

func (s *StateStorage) Load(path string) ([]byte, error) {
	if st, err := s.rw.Read(path); err != nil {
		return nil, err
	} else {
		return st, nil
	}
}

func (s *StateStorage) Save(path string, data []byte) error {
	if err := s.rw.Write(path, data); err != nil {
		return err
	}
	return nil
}

// Write writes the state to a file with the path.
func (f FileReaderWriter) Write(path string, data []byte) error {
	return ioutil.WriteFile(path, data, 0644)
}

// Read reads the state from a file with the path.
func (f FileReaderWriter) Read(path string) ([]byte, error) {
	if bytes, err := ioutil.ReadFile(path); err == nil {
		return bytes, nil
	} else {
		return []byte{}, err
	}
}
