package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/spf13/pflag"
)

type Config struct {
	Network struct {
		Port               int
		CoordinatorAddress string
		HttpPort           int
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
		Libretro map[string]LibretroConfig
	}

	Encoder struct {
		WithoutGame bool
	}

	Monitoring monitoring.ServerMonitoringConfig
}

type LibretroConfig struct {
	Path        string
	Config      string
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

func NewDefaultConfig() Config {
	var conf Config
	config.LoadConfig(&conf, configPath)
	return conf
}

func (c *Config) AddFlags(fs *pflag.FlagSet) *Config {
	fs.IntVarP(&c.Network.Port, "port", "", 8800, "Worker server port")
	fs.StringVarP(&c.Network.CoordinatorAddress, "coordinatorhost", "", c.Network.CoordinatorAddress, "Worker URL to connect")
	fs.IntVarP(&c.Network.HttpPort, "httpPort", "", c.Network.HttpPort, "Set external HTTP port")
	fs.StringVarP(&configPath, "conf", "c", "", "Set custom configuration file path")

	fs.IntVarP(&c.Monitoring.Port, "monitoring.port", "", c.Monitoring.Port, "Monitoring server port")
	return c
}
