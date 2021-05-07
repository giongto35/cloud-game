package service

import (
	"context"
	"log"
)

type Service interface{}

type RunnableService interface {
	Service

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
		if v, ok := s.(RunnableService); ok {
			go v.Run()
		}
	}
}

// Shutdown !to add a proper HTTP(S) server shutdown (cws/handler bad loop)
func (svs *Services) Shutdown(ctx context.Context) {
	for _, s := range svs.list {
		if v, ok := s.(RunnableService); ok {
			if err := v.Shutdown(ctx); err != nil && err != context.Canceled {
				log.Printf("error: failed to stop [%s] because of %v", s, err)
			}
		}
	}
}
