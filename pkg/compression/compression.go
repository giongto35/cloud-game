package compression

import (
	"path/filepath"

	"github.com/giongto35/cloud-game/v2/pkg/compression/zip"
)

type Extractor interface {
	Extract(src string, dest string) ([]string, error)
}

func NewExtractorFromExt(path string) Extractor {
	switch filepath.Ext(path) {
	case zip.Ext:
		return zip.New()
	default:
		return nil
	}
}
