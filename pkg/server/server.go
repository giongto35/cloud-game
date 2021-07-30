package server

import (
	"context"
	"log"
)

type Server interface {
	Run() error
	Shutdown(ctx context.Context) error
}

type Services struct {
	list []Server
}

func (svs *Services) AddIf(condition bool, services ...Server) *Services {
	if !condition {
		return svs
	}
	for _, s := range services {
		svs.list = append(svs.list, s)
	}
	return svs
}

func (svs *Services) Start() {
	for _, s := range svs.list {
		s := s
		go func() {
			if err := s.Run(); err != nil {
				log.Printf("error: failed to start service [%s] due to [%v]", s, err)
			}
		}()
	}
}

// Shutdown !to add a proper HTTP(S) server shutdown (cws/handler bad loop)
func (svs *Services) Shutdown(ctx context.Context) {
	for _, s := range svs.list {
		if err := s.Shutdown(ctx); err != nil {
			log.Printf("error: failed to stop [%s] because of %v", s, err)
		}
	}
}
