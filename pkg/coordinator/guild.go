package coordinator

import (
	"github.com/giongto35/cloud-game/v2/pkg/cache"
	"github.com/giongto35/cloud-game/v2/pkg/client"
)

// Guild is an abstraction over list of workers and their jobs.
type Guild struct {
	cache.Cache
}

func NewGuild() Guild {
	return Guild{cache.New(make(map[string]client.NetClient, 10))}
}

func (g *Guild) add(worker *Worker) { g.Add(string(worker.Id()), worker) }

func (g *Guild) Remove(w *Worker) { g.Cache.Remove(string(w.Id())) }

func (g *Guild) findFreeByIp(addr string) *Worker {
	worker, err := g.FindBy(func(cl client.NetClient) bool {
		worker := cl.(*Worker)
		return worker.HasGameSlot() && worker.Address == addr
	})
	if err != nil {
		return nil
	}
	return worker.(*Worker)
}

func (g *Guild) findByPingServer(address string) *Worker {
	worker, err := g.FindBy(func(cl client.NetClient) bool {
		worker := cl.(*Worker)
		return worker.PingServer == address
	})
	if err != nil {
		return nil
	}
	return worker.(*Worker)
}

func (g *Guild) filter(fn func(w *Worker) bool) []*Worker {
	var list []*Worker
	g.ForEach(func(cl client.NetClient) {
		worker := cl.(*Worker)
		if fn(worker) {
			list = append(list, worker)
		}
	})
	return list
}
