package manager

import (
	"fmt"
	"io/ioutil"
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

func (m BasicManager) GetInstalled() (installed []string, err error) {
	dir := m.Conf.GetCoresStorePath()
	arch, err := core.GetCoreExt()
	if err != nil {
		return installed, err
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return installed, fmt.Errorf("couldn't read installed cores: %w", err)
	}

	for _, file := range files {
		name := file.Name()
		if filepath.Ext(name) == arch.LibExt {
			installed = append(installed, strings.TrimSuffix(name, arch.LibExt))
		}
	}
	return
}
