package manager

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/core"
)

type Manager interface {
	Sync() error
}

type BasicManager struct {
	Conf emulator.LibretroConfig
}

func (m BasicManager) GetInstalled() (installed []emulator.CoreInfo, err error) {
	dir := m.Conf.GetCoresStorePath()
	arch, err := core.GetCoreExt()
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
			installed = append(installed, emulator.CoreInfo{Name: strings.TrimSuffix(name, arch.LibExt)})
		}
	}
	return
}
