package worker

import (
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/room"
)

type Watcher struct {
	r    *room.GameRouter
	t    *time.Ticker
	done chan struct{}
	log  *logger.Logger
}

func NewWatcher(p time.Duration, router *room.GameRouter, log *logger.Logger) *Watcher {
	return &Watcher{
		r:    router,
		t:    time.NewTicker(p),
		done: make(chan struct{}),
		log:  log,
	}
}

func (w *Watcher) Run() {
	go func() {
		for {
			select {
			case <-w.t.C:
				if w.r.HasRoom() && w.r.Users().Empty() {
					w.r.Close()
					w.log.Warn().Msgf("Forced room close!")
				}
			case <-w.done:
				return
			}
		}
	}()
}

func (w *Watcher) Stop() error {
	w.t.Stop()
	close(w.done)
	return nil
}
