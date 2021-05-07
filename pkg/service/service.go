package service

import (
	"context"
	"log"
)

type Service interface {
	Run()
	Shutdown(ctx context.Context) error
}

type Services struct {
	list []Service
}

func (svs *Services) Add(services ...Service) {
	for _, s := range services {
		svs.list = append(svs.list, s)
	}
}

func (svs *Services) Start() {
	for _, s := range svs.list {
		go s.Run()
	}
}

// Shutdown !to add a proper HTTP(S) server shutdown (cws/handler bad loop)
func (svs *Services) Shutdown(ctx context.Context) {
	for _, s := range svs.list {
		if err := s.Shutdown(ctx); err != nil && err != context.Canceled {
			log.Printf("error: failed to stop [%s] because of %v", s, err)
		}
	}
}
