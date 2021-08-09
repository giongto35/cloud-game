package monitoring

type Config struct {
	Port             int
	URLPrefix        string
	MetricEnabled    bool `json:"metric_enabled"`
	ProfilingEnabled bool `json:"profiling_enabled"`
}

func (c *Config) IsEnabled() bool { return c.MetricEnabled || c.ProfilingEnabled }
