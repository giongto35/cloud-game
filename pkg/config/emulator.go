package config

import (
	"path"
	"path/filepath"
	"strings"
)

type Emulator struct {
	Threads     int
	Storage     string
	LocalPath   string
	Libretro    LibretroConfig
	AutosaveSec int
}

type LibretroConfig struct {
	Cores struct {
		Paths struct {
			Libs string
		}
		Repo struct {
			Sync      bool
			ExtLock   string
			Main      LibretroRepoConfig
			Secondary LibretroRepoConfig
		}
		List map[string]LibretroCoreConfig
	}
	DebounceMs      int
	Dup             bool
	SaveCompression bool
	LogLevel        int
}

type LibretroRepoConfig struct {
	Type        string
	Url         string
	Compression string
}

type LibretroCoreConfig struct {
	AltRepo         bool
	AutoGlContext   bool // hack: keep it here to pass it down the emulator
	CoreAspectRatio bool
	Folder          string
	Hacks           []string
	Height          int
	Hid             map[int][]int
	IsGlAllowed     bool
	Lib             string
	Options         map[string]string
	Options4rom     map[string]map[string]string // <(^_^)>
	Roms            []string
	Scale           float64
	UsesLibCo       bool
	VFR             bool
	Width           int
}

type CoreInfo struct {
	Id      string
	Name    string
	AltRepo bool
}

// GetLibretroCoreConfig returns a core config with expanded paths.
func (e Emulator) GetLibretroCoreConfig(emulator string) LibretroCoreConfig {
	cores := e.Libretro.Cores
	conf := cores.List[emulator]
	conf.Lib = path.Join(cores.Paths.Libs, conf.Lib)
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

func (l *LibretroConfig) GetCores() (cores []CoreInfo) {
	for k, core := range l.Cores.List {
		cores = append(cores, CoreInfo{Id: k, Name: core.Lib, AltRepo: core.AltRepo})
	}
	return
}

func (l *LibretroConfig) GetCoresStorePath() string {
	pth, err := filepath.Abs(l.Cores.Paths.Libs)
	if err != nil {
		return ""
	}
	return pth
}
