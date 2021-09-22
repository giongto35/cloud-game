package coordinator

import "github.com/giongto35/cloud-game/v2/pkg/client"

func (h *Hub) findWorkerByRoom(id string, region string) *Worker {
	w, err := h.rooms.Find(id)
	if err == nil {
		if w.(client.RegionalClient).In(region) {
			return w.(*Worker)
		}
		// if there is zone param, we need to ensure ther worker in that zone
		// if not we consider the room is missing
	}
	return nil
}

func (h *Hub) findWorkerByIp(address string) *Worker {
	if address == "" {
		return nil
	}
	return h.guild.findFreeByIp(address)
}

func (h *Hub) getAvailableWorkers(region string) []*Worker {
	return h.guild.filter(func(w *Worker) bool { return w.HasGameSlot() && w.In(region) })
}

func (h *Hub) findAnyFreeWorker(region string) *Worker {
	workers := h.getAvailableWorkers(region)
	if len(workers) > 0 {
		return workers[0]
	}
	return nil
}

// findFastestWorker returns the best server for a session.
// All workers addresses are sent to user and user will ping to get latency.
// !to rewrite
func (h *Hub) findFastestWorker(region string, fn func(addresses []string) (map[string]int64, error)) *Worker {
	workers := h.getAvailableWorkers(region)
	if len(workers) == 0 {
		return nil
	}

	var addresses []string
	group := map[string][]struct{}{}
	for _, w := range workers {
		if _, ok := group[w.PingServer]; !ok {
			addresses = append(addresses, w.PingServer)
		}
		group[w.PingServer] = append(group[w.PingServer], struct{}{})
	}

	latencies, err := fn(addresses)
	if len(latencies) == 0 || err != nil {
		return nil
	}

	var bestWorker *Worker
	var minLatency int64 = 1<<31 - 1
	// get a worker with the lowest latency
	for addr, ping := range latencies {
		if ping < minLatency {
			bestWorker = h.guild.findByPingServer(addr)
			minLatency = ping
		}
	}
	return bestWorker
}
