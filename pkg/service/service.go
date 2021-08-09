package service

import (
	"context"
	"log"
)

// Service defines a generic service.
type Service interface{}

// RunnableService defines a service that can be run.
type RunnableService interface {
	Service

	Run()
	Shutdown(ctx context.Context) error
}

// Group is a container for managing a bunch of services.
type Group struct {
	list []Service
}

func (g *Group) Add(services ...Service) {
	for _, s := range services {
		g.list = append(g.list, s)
	}
}

// Start starts each service in the group.
func (g *Group) Start() {
	for _, s := range g.list {
		if v, ok := s.(RunnableService); ok {
			go v.Run()
		}
	}
}

// Shutdown terminates a group of services.
func (g *Group) Shutdown(ctx context.Context) {
	for _, s := range g.list {
		if v, ok := s.(RunnableService); ok {
			if err := v.Shutdown(ctx); err != nil && err != context.Canceled {
				log.Printf("error: failed to stop [%s] because of %v", s, err)
			}
		}
	}
}
