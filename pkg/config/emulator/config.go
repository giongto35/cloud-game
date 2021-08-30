package emulator

import (
	"path"
	"path/filepath"
	"strings"
)

type Emulator struct {
	Scale       int
	AspectRatio struct {
		Keep   bool
		Width  int
		Height int
	}
	Storage  string
	Libretro LibretroConfig
}

type LibretroConfig struct {
	Cores struct {
		Paths struct {
			Libs    string
			Configs string
		}
		Repo struct {
			Sync      bool
			ExtLock   string
			Main      LibretroRepoConfig
			Secondary LibretroRepoConfig
		}
		List map[string]LibretroCoreConfig
	}
}

type LibretroRepoConfig struct {
	Type        string
	Url         string
	Compression string
}

type LibretroCoreConfig struct {
	Lib         string
	Config      string
	Roms        []string
	Folder      string
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
func (e Emulator) GetLibretroCoreConfig(emulator string) LibretroCoreConfig {
	cores := e.Libretro.Cores
	conf := cores.List[emulator]
	conf.Lib = path.Join(cores.Paths.Libs, conf.Lib)
	if conf.Config != "" {
		conf.Config = path.Join(cores.Paths.Configs, conf.Config)
	}
	return conf
}

// GetEmulator tries to find a suitable emulator.
// !to remove quadratic complexity
func (e Emulator) GetEmulator(rom string, path string) string {
	found := ""
	for emu, core := range e.Libretro.Cores.List {
		for _, romName := range core.Roms {
			if rom == romName {
				found = emu
				if p := strings.SplitN(filepath.ToSlash(path), "/", 2); len(p) > 1 {
					folder := p[0]
					if (folder != "" && folder == core.Folder) || folder == emu {
						return emu
					}
				}
			}
		}
	}
	return found
}

func (e Emulator) GetSupportedExtensions() []string {
	var extensions []string
	for _, core := range e.Libretro.Cores.List {
		extensions = append(extensions, core.Roms...)
	}
	return extensions
}

func (l *LibretroConfig) GetCores() []string {
	var cores []string
	for _, core := range l.Cores.List {
		cores = append(cores, core.Lib)
	}
	return cores
}

func (l *LibretroConfig) GetCoresStorePath() string {
	pth, err := filepath.Abs(l.Cores.Paths.Libs)
	if err != nil {
		return ""
	}
	return pth
}
