package manager

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/giongto35/cloud-game/v3/pkg/config"
)

type BasicManager struct {
	Conf config.LibretroConfig
}

func (m BasicManager) GetInstalled(libExt string) (installed []config.CoreInfo, err error) {
	if libExt == "" {
		return
	}
	dir := m.Conf.GetCoresStorePath()
	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, file := range files {
		name := file.Name()
		if filepath.Ext(name) == libExt {
			installed = append(installed, config.CoreInfo{Name: strings.TrimSuffix(name, libExt)})
		}
	}
	return
}
