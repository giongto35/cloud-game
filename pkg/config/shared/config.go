package shared

import (
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	flag "github.com/spf13/pflag"
)

type Environment environment.Env

type Server struct {
	Address string
	Https   bool
	Tls     struct {
		Address   string
		Domain    string
		HttpsKey  string
		HttpsCert string
	}
}

func (s *Server) WithFlags() {
	flag.StringVar(&s.Address, "address", s.Address, "HTTP server address (host:port)")
	flag.StringVar(&s.Tls.Address, "httpsAddress", s.Tls.Address, "HTTPS server address (host:port)")
	flag.StringVar(&s.Tls.HttpsKey, "httpsKey", s.Tls.HttpsKey, "HTTPS key")
	flag.StringVar(&s.Tls.HttpsCert, "httpsCert", s.Tls.HttpsCert, "HTTPS chain")
}

func (s *Server) GetAddr() string {
	if s.Https {
		return s.Tls.Address
	}
	return s.Address
}

func (env *Environment) Get() environment.Env {
	return (environment.Env)(*env)
}

func (env *Environment) WithFlags() {
	flag.StringVar((*string)(env), "env", string(*env), "Specify environment type: [dev, staging, prod]")
}
