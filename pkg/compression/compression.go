package compression

import (
	"path/filepath"

	"github.com/giongto35/cloud-game/v2/pkg/compression/zip"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
)

type Extractor interface {
	Extract(src string, dest string) ([]string, error)
}

const zipExt = ".zip"

func NewFromExt(path string, log *logger.Logger) Extractor {
	switch filepath.Ext(path) {
	case zip.Ext:
		return zip.New(log)
	default:
		return nil
	}
}
