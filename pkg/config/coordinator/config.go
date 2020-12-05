package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/spf13/pflag"
)

type Config struct {
	Shared shared.Config

	PublicDomain      string
	PingServer        string
	URLPrefix         string
	DebugHost         string
	LibraryMonitoring bool
}

func NewDefaultConfig() Config {
	conf := Config{
		PublicDomain:      "http://localhost:8000",
		PingServer:        "",
		LibraryMonitoring: false,
	}

	conf.Shared.Monitoring = monitoring.ServerMonitoringConfig{
		Port:             6601,
		URLPrefix:        "/coordinator",
		MetricEnabled:    false,
		ProfilingEnabled: false,
	}

	return conf
}

const StunTurnTemplate = `[{"urls":"stun:stun.l.google.com:19302"},{"urls":"stun:%s:3478"},{"urls":"turn:%s:3478","username":"root","credential":"root"}]`

var FrontendSTUNTURN string

const AUDIO_RATE = 48000
const AUDIO_CHANNELS = 2
const AUDIO_MS = 20
const AUDIO_FRAME = AUDIO_RATE * AUDIO_MS / 1000 * AUDIO_CHANNELS

var EmulatorExtension = []string{".so", ".armv7-neon-hf.so", ".dylib", ".dll"}

var SupportedRomExtensions = []string{
	"gba", "gbc", "cue", "zip", "nes", "smc", "sfc", "swc", "fig", "bs", "n64", "v64", "z64",
}

func (c *Config) AddFlags(fs *pflag.FlagSet) *Config {
	c.Shared.AddFlags(fs)

	fs.StringVarP(&c.DebugHost, "debughost", "d", "", "Specify the server want to connect directly to debug")
	fs.StringVarP(&c.PublicDomain, "domain", "n", c.PublicDomain, "Specify the public domain of the coordinator")
	fs.StringVarP(&c.PingServer, "pingServer", "", c.PingServer, "Specify the worker address that the client can ping (with protocol and port)")
	fs.BoolVarP(&c.LibraryMonitoring, "libMonitor", "", c.LibraryMonitoring, "Enable ROM library monitoring")
	return c
}
