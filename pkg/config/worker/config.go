package worker

import (
	"log"
	"path"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	"github.com/spf13/pflag"
)

type Config struct {
	Shared shared.Config

	Network struct {
		CoordinatorAddress string
		Zone               string
	}

	Emulator struct {
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

	Encoder struct {
		WithoutGame bool
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

// allows custom config path
var configPath string

func NewDefaultConfig() *Config {
	var conf Config
	config.LoadConfig(&conf, configPath)

	log.Printf("%+v", conf)

	return &conf
}

func (c *Config) WithFlags(fs *pflag.FlagSet) *Config {
	c.Shared.AddFlags(fs)

	if err := fs.Set("port", "9000"); err != nil {
		log.Printf("error: couldn't override default port value, %v", err)
	}
	fs.StringVar(&c.Network.CoordinatorAddress, "coordinatorhost", c.Network.CoordinatorAddress, "Worker URL to connect")
	fs.StringVarP(&configPath, "conf", "c", "", "Set custom configuration file path")
	return c
}

// GetLibretroCoreConfig returns a core config with expanded paths.
func (c *Config) GetLibretroCoreConfig(emulator string) LibretroCoreConfig {
	cores := c.Emulator.Libretro.Cores
	conf := cores.List[emulator]
	conf.Lib = path.Join(cores.Paths.Libs, conf.Lib)
	if conf.Config != "" {
		conf.Config = path.Join(cores.Paths.Configs, conf.Config)
	}
	return conf
}

// GetEmulatorByRom returns emulator name by its supported ROM name.
// !to cache into an optimized data structure
func (c *Config) GetEmulatorByRom(rom string) string {
	for emu, core := range c.Emulator.Libretro.Cores.List {
		for _, romName := range core.Roms {
			if rom == romName {
				return emu
			}
		}
	}
	return ""
}
