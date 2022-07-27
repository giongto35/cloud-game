package compression

import (
	"path/filepath"

	"github.com/giongto35/cloud-game/v2/pkg/compression/zip"
)

type Extractor interface {
	Extract(src string, dest string) ([]string, error)
}

const zipExt = ".zip"

func NewExtractorFromExt(path string) Extractor {
	switch filepath.Ext(path) {
	case zipExt:
		return zip.New()
	default:
		return nil
	}
}
