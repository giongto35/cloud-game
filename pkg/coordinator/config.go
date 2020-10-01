package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/monitoring"
	"github.com/spf13/pflag"
)

type Config struct {
	Port              int
	PublicDomain      string
	PingServer        string
	URLPrefix         string
	DebugHost         string
	LibraryMonitoring bool

	MonitoringConfig monitoring.ServerMonitoringConfig
}

func NewDefaultConfig() Config {
	return Config{
		Port:              8800,
		PublicDomain:      "http://localhost:8000",
		PingServer:        "",
		LibraryMonitoring: false,

		MonitoringConfig: monitoring.ServerMonitoringConfig{
			Port:             6601,
			URLPrefix:        "/coordinator",
			MetricEnabled:    false,
			ProfilingEnabled: false,
		},
	}
}

func (c *Config) AddFlags(fs *pflag.FlagSet) *Config {
	fs.IntVarP(&c.Port, "port", "", 8800, "Coordinator server port")

	fs.BoolVarP(&c.MonitoringConfig.MetricEnabled, "monitoring.metric", "m", c.MonitoringConfig.MetricEnabled, "Enable prometheus metric for server")
	fs.BoolVarP(&c.MonitoringConfig.ProfilingEnabled, "monitoring.pprof", "p", c.MonitoringConfig.ProfilingEnabled, "Enable golang pprof for server")
	fs.IntVarP(&c.MonitoringConfig.Port, "monitoring.port", "", c.MonitoringConfig.Port, "Monitoring server port")
	fs.StringVarP(&c.MonitoringConfig.URLPrefix, "monitoring.prefix", "", c.MonitoringConfig.URLPrefix, "Monitoring server url prefix")
	fs.StringVarP(&c.DebugHost, "debughost", "d", "", "Specify the server want to connect directly to debug")
	fs.StringVarP(&c.PublicDomain, "domain", "n", c.PublicDomain, "Specify the public domain of the coordinator")
	fs.StringVarP(&c.PingServer, "pingServer", "", c.PingServer, "Specify the worker address that the client can ping (with protocol and port)")
	fs.BoolVarP(&c.LibraryMonitoring, "libMonitor", "", c.LibraryMonitoring, "Enable ROM library monitoring")

	return c
}
