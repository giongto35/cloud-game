package shared

import (
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	flag "github.com/spf13/pflag"
)

type Environment environment.Env

type Server struct {
	Address      string
	HttpsAddress string
	HttpsKey     string
	HttpsChain   string
}

func (s *Server) WithFlags() {
	flag.StringVar(&s.Address, "address", s.Address, "HTTP server address (host:port)")
	flag.StringVar(&s.HttpsAddress, "httpsAddress", s.HttpsAddress, "HTTPS server address (host:port)")
	flag.StringVar(&s.HttpsKey, "httpsKey", s.HttpsKey, "HTTPS key")
	flag.StringVar(&s.HttpsChain, "httpsChain", s.HttpsChain, "HTTPS chain")
}

func (env *Environment) Get() environment.Env {
	return (environment.Env)(*env)
}

func (env *Environment) WithFlags() {
	flag.StringVar((*string)(env), "env", string(*env), "Specify environment type: [dev, staging, prod]")
}
