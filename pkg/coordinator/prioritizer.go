package coordinator

func (h *Hub) findWorkerByRoom(id string, region string) *WorkerClient {
	if id == "" {
		return nil
	}

	if w, ok := h.rooms[id]; ok {
		if w.inRegion(region) {
			return w
		}
		// if there is zone param, we need to ensure ther worker in that zone
		// if not we consider the room is missing
	}
	return nil
}

func (h *Hub) findWorkerByIp(address string) *WorkerClient {
	if address == "" {
		return nil
	}
	return h.guild.findFreeByIp(address)
}

func (h *Hub) getAvailableWorkers(region string) []*WorkerClient {
	return h.guild.filter(func(w *WorkerClient) bool { return w.IsFree && w.inRegion(region) })
}

func (h *Hub) findAnyFreeWorker(region string) *WorkerClient {
	workers := h.getAvailableWorkers(region)
	if len(workers) > 0 {
		return workers[0]
	}
	return nil
}

// findFastestWorker returns the best server for a session.
// All workers addresses are sent to user and user will ping to get latency.
func (h *Hub) findFastestWorker(region string, fn func(addresses []string) (error, map[string]int64)) *WorkerClient {
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

	err, latencies := fn(addresses)
	if len(latencies) == 0 || err != nil {
		return nil
	}

	var bestWorker *WorkerClient
	var minLatency int64 = 1<<31 - 1
	// get the worker with lowest latency to user
	for addr, ping := range latencies {
		if ping < minLatency {
			bestWorker = h.guild.findByPingServer(addr)
			minLatency = ping
		}
	}
	return bestWorker
}
