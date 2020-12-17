package extractor

import (
	"path/filepath"

	"github.com/giongto35/cloud-game/v2/pkg/extractor/zip"
)

type Extractor interface {
	Extract(src string, dest string) ([]string, error)
}

const (
	zipExt = ".zip"
)

func NewFromExt(path string) Extractor {
	ext := filepath.Ext(path)
	switch ext {
	case zipExt:
		return zip.New()
	default:
		return nil
	}
}
