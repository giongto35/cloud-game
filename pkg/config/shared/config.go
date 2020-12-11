package shared

import (
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"github.com/spf13/pflag"
)

type Config struct {
	Environment struct {
		Mode environment.Env
	}

	Server struct {
		Port       int
		HttpsPort  int
		HttpsKey   string
		HttpsChain string
	}
}

func (c *Config) AddFlags(fs *pflag.FlagSet) *Config {
	fs.StringVar((*string)(&c.Environment.Mode), "mode", "dev", "Specify environment type: [dev, staging, prod]")
	fs.IntVar(&c.Server.Port, "port", 8000, "HTTP server port")
	fs.IntVar(&c.Server.HttpsPort, "httpsPort", 443, "HTTPS server port (just why?)")
	fs.StringVar(&c.Server.HttpsKey, "httpsKey", "", "HTTPS key")
	fs.StringVar(&c.Server.HttpsChain, "httpsChain", "", "HTTPS chain")
	return c
}
