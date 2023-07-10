package caged

import (
	"errors"
	"reflect"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/app"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro"
)

type Manager struct {
	list map[ModName]app.App
	log  *logger.Logger
}

type ModName string

const Libretro ModName = "libretro"

func NewManager(log *logger.Logger) *Manager {
	return &Manager{log: log, list: make(map[ModName]app.App)}
}

func (m *Manager) Get(name ModName) app.App { return m.list[name] }

func (m *Manager) Load(name ModName, conf any) error {
	if name == Libretro {
		caged, err := m.loadLibretro(conf)
		if err != nil {
			return err
		}
		m.list[name] = caged
	}
	return nil
}

func (m *Manager) loadLibretro(conf any) (*libretro.Caged, error) {
	s := reflect.ValueOf(conf)

	e := s.FieldByName("Emulator")
	if !e.IsValid() {
		return nil, errors.New("no emulator conf")
	}
	r := s.FieldByName("Recording")
	if !r.IsValid() {
		return nil, errors.New("no recording conf")
	}

	c := libretro.CagedConf{
		Emulator:  e.Interface().(config.Emulator),
		Recording: r.Interface().(config.Recording),
	}

	caged := libretro.Cage(c, m.log)
	if err := caged.Init(); err != nil {
		return nil, err
	}
	return &caged, nil
}
