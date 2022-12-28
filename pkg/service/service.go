package service

import "fmt"

// Service defines a generic service.
type Service any

// RunnableService defines a service that can be run.
type RunnableService interface {
	Service

	Run()
	Stop() error
}

// Group is a container for managing a bunch of services.
type Group struct {
	list []Service
}

func (g *Group) Add(services ...Service) { g.list = append(g.list, services...) }

// Start starts each service in the group.
func (g *Group) Start() {
	for _, s := range g.list {
		if v, ok := s.(RunnableService); ok {
			v.Run()
		}
	}
}

// Stop terminates a group of services.
func (g *Group) Stop() (err error) {
	var errs []error
	for _, s := range g.list {
		if v, ok := s.(RunnableService); ok {
			if err := v.Stop(); err != nil {
				errs = append(errs, fmt.Errorf("error: failed to stop [%s] because of %v", s, err))
			}
		}
	}
	if len(errs) > 0 {
		err = fmt.Errorf("%s", errs)
	}
	return
}
