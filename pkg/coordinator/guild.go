package coordinator

import (
	"log"
	"sync"

	"github.com/giongto35/cloud-game/v2/pkg/network"
)

// Guild denotes some abstraction over list of workers
// and their jobs.
type Guild struct {
	mu sync.Mutex

	workers map[network.Uid]*WorkerClient
}

func NewGuild() Guild {
	return Guild{
		workers: map[network.Uid]*WorkerClient{},
	}
}

func (g *Guild) add(w *WorkerClient) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.workers[w.Id] = w
	log.Printf("Guild: %v", g.workers)
}

func (g *Guild) remove(w *WorkerClient) {
	w.Printf("Has done his duty")
	g.mu.Lock()
	defer g.mu.Unlock()

	delete(g.workers, w.Id)
	w.Close()
	log.Printf("Guild: %v", g.workers)
}

func (g *Guild) findFreeByIp(addr string) *WorkerClient {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, w := range g.workers {
		if w.IsFree && w.Address == addr {
			return w
		}
	}
	return nil
}

func (g *Guild) findByPingServer(address string) *WorkerClient {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, w := range g.workers {
		if w.PingServer == address {
			return w
		}
	}
	return nil
}

func (g *Guild) filter(fn func(w *WorkerClient) bool) []*WorkerClient {
	g.mu.Lock()
	defer g.mu.Unlock()

	var list []*WorkerClient
	for _, w := range g.workers {
		if fn(w) {
			list = append(list, w)
		}
	}
	return list
}
