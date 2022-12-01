package coordinator

import (
	"bytes"

	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/rs/xid"
)

func (h *Hub) findWorkerByRoom(id string, region string) *Worker {
	w, err := h.rooms2workers.Find(id)
	if err == nil {
		if w.(com.RegionalClient).In(region) {
			return w.(*Worker)
		}
		// if there is zone param, we need to ensure the worker in that zone,
		// if not we consider the room is missing
	}
	return nil
}

func (h *Hub) getAvailableWorkers(region string) []*Worker {
	var workers []*Worker
	h.workers.ForEach(func(w *Worker) {
		if w.HasGameSlot() && w.In(region) {
			workers = append(workers, w)
		}
	})
	return workers
}

func (h *Hub) find1stFreeWorker(region string) *Worker {
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

	workers = h.getAvailableWorkers(region)
	if len(workers) == 0 {
		return nil
	}

	var bestWorker *Worker
	var minLatency int64 = 1<<31 - 1
	// get a worker with the lowest latency
	for addr, ping := range latencies {
		if ping < minLatency {
			for _, w := range workers {
				if w.PingServer == addr {
					bestWorker = w
				}
			}
			minLatency = ping
		}
	}
	return bestWorker
}

func (h *Hub) findWorkerById(workerId string, useAllWorkers bool) *Worker {
	// when we select one particular worker
	if workerId != "" {
		if xid_, err := xid.FromString(workerId); err == nil {
			if useAllWorkers {
				for _, w := range h.getAvailableWorkers("") {
					if xid_.String() == w.Id().String() {
						return w
					}
				}
			} else {
				for _, w := range h.getAvailableWorkers("") {
					xid__, err := xid.FromString(workerId)
					if err != nil {
						continue
					}
					if bytes.Equal(xid_.Machine(), xid__.Machine()) {
						return w
					}
				}
			}
		}
	}
	return nil
}
