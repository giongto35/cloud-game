package nanoarch

import "path/filepath"

type Storage struct {
	// save path without the dir slash in the end
	Path string
	// contains the name of the main save file
	// e.g. abc<...>293.dat
	// needed for Google Cloud save/restore which
	// doesn't support multiple files
	MainSave string
}

func (s *Storage) GetSavePath() string { return filepath.Join(s.Path, s.MainSave+".dat") }
func (s *Storage) GetSRAMPath() string { return filepath.Join(s.Path, s.MainSave+".srm") }
