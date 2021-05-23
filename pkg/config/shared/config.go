package shared

import (
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	flag "github.com/spf13/pflag"
)

type Environment environment.Env

type Server struct {
	Port       int
	HttpsPort  int
	HttpsKey   string
	HttpsChain string
}

func (s *Server) WithFlags() {
	flag.IntVar(&s.Port, "port", s.Port, "HTTP server port")
	flag.IntVar(&s.HttpsPort, "httpsPort", s.HttpsPort, "HTTPS server port (just why?)")
	flag.StringVar(&s.HttpsKey, "httpsKey", s.HttpsKey, "HTTPS key")
	flag.StringVar(&s.HttpsChain, "httpsChain", s.HttpsChain, "HTTPS chain")
}

func (env *Environment) Get() environment.Env {
	return (environment.Env)(*env)
}

func (env *Environment) WithFlags() {
	flag.StringVar((*string)(env), "env", string(*env), "Specify environment type: [dev, staging, prod]")
}
