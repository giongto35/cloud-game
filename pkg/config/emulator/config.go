package emulator

import "path"

type Emulator struct {
	Scale       int
	AspectRatio struct {
		Keep   bool
		Width  int
		Height int
	}
	Width    int
	Height   int
	Libretro struct {
		Cores struct {
			Paths struct {
				Libs    string
				Configs string
			}
			List map[string]LibretroCoreConfig
		}
	}
}

type LibretroCoreConfig struct {
	Lib         string
	Config      string
	Roms        []string
	Width       int
	Height      int
	Ratio       float64
	IsGlAllowed bool
	UsesLibCo   bool
	HasMultitap bool

	// hack: keep it here to pass it down the emulator
	AutoGlContext bool
}

// GetLibretroCoreConfig returns a core config with expanded paths.
func (e *Emulator) GetLibretroCoreConfig(emulator string) LibretroCoreConfig {
	cores := e.Libretro.Cores
	conf := cores.List[emulator]
	conf.Lib = path.Join(cores.Paths.Libs, conf.Lib)
	if conf.Config != "" {
		conf.Config = path.Join(cores.Paths.Configs, conf.Config)
	}
	return conf
}

// GetEmulatorByRom returns emulator name by its supported ROM name.
// !to cache into an optimized data structure
func (e *Emulator) GetEmulatorByRom(rom string) string {
	for emu, core := range e.Libretro.Cores.List {
		for _, romName := range core.Roms {
			if rom == romName {
				return emu
			}
		}
	}
	return ""
}

func (e *Emulator) GetSupportedExtensions() []string {
	var extensions []string
	for _, core := range e.Libretro.Cores.List {
		extensions = append(extensions, core.Roms...)
	}
	return extensions
}
