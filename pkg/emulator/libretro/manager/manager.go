package manager

import (
	"io/ioutil"
	"log"
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

func (m BasicManager) GetInstalled() (installed []string) {
	dir := m.Conf.GetCoresStorePath()
	arch, err := core.GetCoreExt()
	if err != nil {
		log.Printf("error: %v", err)
		return
	}

	files, err := ioutil.ReadDir(dir)
	if err != nil {
		log.Printf("error: couldn't get installed cores, %v", err)
		return
	}

	for _, file := range files {
		name := file.Name()
		if filepath.Ext(name) == arch.LibExt {
			installed = append(installed, strings.TrimSuffix(name, arch.LibExt))
		}
	}
	return
}
