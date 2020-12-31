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
	switch filepath.Ext(path) {
	case zipExt:
		return zip.New()
	default:
		return nil
	}
}
