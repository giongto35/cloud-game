package compression

import (
	"path/filepath"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/compression/zip"
)

type Extractor interface {
	Extract(src string, dest string) ([]string, error)
}

func NewFromExt(path string, log *logger.Logger) Extractor {
	switch filepath.Ext(path) {
	case zip.Ext:
		return zip.New(log)
	default:
		return nil
	}
}
