package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	"github.com/spf13/pflag"
)

type Config struct {
	Shared shared.Config

	PublicDomain string
	PingServer   string
	DebugHost    string

	LibraryMonitoring bool

	Webrtc struct {
		IceServers []webrtc.IceServer
	}
}

func NewDefaultConfig() *Config {
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

	conf.Webrtc.IceServers = []webrtc.IceServer{
		{Url: "stun:stun.l.google.com:19302"},
		{Url: "stun:{server-ip}:3478"},
		{Url: "turn:{server-ip}:3478", Username: "root", Credential: "root"},
	}

	return &conf
}

var SupportedRomExtensions = []string{
	"gba", "gbc", "cue", "zip", "nes", "smc", "sfc", "swc", "fig", "bs", "n64", "v64", "z64",
}

func (c *Config) WithFlags(fs *pflag.FlagSet) *Config {
	c.Shared.AddFlags(fs)
	return c
}
