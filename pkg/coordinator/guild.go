package coordinator

import "github.com/giongto35/cloud-game/v2/pkg/client"

// Guild is an abstraction over list of workers and their jobs.
type Guild struct {
	client.NetMap
}

func NewGuild() Guild { return Guild{client.NewNetMap()} }

func (g *Guild) Remove(w *Worker) { g.NetMap.Remove(w) }

func (g *Guild) add(worker *Worker) { g.Add(worker) }

func (g *Guild) findByPingServer(address string) *Worker {
	w, _ := g.FindBy(func(w client.NetClient) bool {
		worker := w.(*Worker)
		if worker.PingServer == address {
			return true
		}
		return false
	})
	if w == nil {
		return nil
	}
	return w.(*Worker)
}

func (g *Guild) filter(fn func(w *Worker) bool) (l []*Worker) {
	g.ForEach(func(w client.NetClient) {
		worker := w.(*Worker)
		if fn(worker) {
			l = append(l, worker)
		}
	})
	return l
}
