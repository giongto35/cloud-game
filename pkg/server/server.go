package server

import "context"

type Server interface {
	Init(conf interface{}) error
	Run() error
	Shutdown(ctx context.Context) error
}
