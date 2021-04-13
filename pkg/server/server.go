package server

import (
	"context"
	"log"
)

type Server interface {
	Run() error
	Shutdown(ctx context.Context) error
}

type Services []Server

func (svs *Services) Start() {
	for _, s := range *svs {
		s := s
		go func() {
			if err := s.Run(); err != nil {
				log.Printf("error: failed to start service [%s] due to [%v]", s, err)
			}
		}()
	}
}

// !to add a proper HTTP(S) server shutdown (cws/handler bad loop)
func (svs *Services) Shutdown(ctx context.Context) {
	for _, s := range *svs {
		if err := s.Shutdown(ctx); err != nil {
			log.Printf("error: failed to stop [%s] because of %v", s, err)
		}
	}
}
