package manager

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/worker/emulator/libretro"
)

type Manager interface {
	Sync() error
}

type BasicManager struct {
	Conf config.LibretroConfig
}

func (m BasicManager) GetInstalled() (installed []config.CoreInfo, err error) {
	dir := m.Conf.GetCoresStorePath()
	arch, err := libretro.GetCoreExt()
	if err != nil {
		return
	}

	files, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, file := range files {
		name := file.Name()
		if filepath.Ext(name) == arch.LibExt {
			installed = append(installed, config.CoreInfo{Name: strings.TrimSuffix(name, arch.LibExt)})
		}
	}
	return
}
