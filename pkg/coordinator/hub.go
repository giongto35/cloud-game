package coordinator

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/rs/xid"
)

type Connection interface {
	Disconnect()
	Id() com.Uid
	Listen() chan struct{}
	OnPacket(func(api.In[com.Uid]) error)

	Send(api.PT, any) ([]byte, error)
	Notify(api.PT, any)
}

type Hub struct {
	conf     coordinator.Config
	launcher games.Launcher
	log      *logger.Logger
	users    com.NetMap[com.Uid, *User]
	workers  com.NetMap[com.Uid, *Worker]
}

func NewHub(conf coordinator.Config, lib games.GameLibrary, log *logger.Logger) *Hub {
	return &Hub{
		conf:     conf,
		users:    com.NewNetMap[com.Uid, *User](),
		workers:  com.NewNetMap[com.Uid, *Worker](),
		launcher: games.NewGameLauncher(lib),
		log:      log,
	}
}

// handleUserConnection handles all connections from user/frontend.
func (h *Hub) handleUserConnection() http.HandlerFunc {
	connector := com.Server{}
	connector.Origin(h.conf.Coordinator.Origin.UserWs)

	log1 := h.log.Extend(h.log.With().Str(logger.ClientField, "u"))

	return func(w http.ResponseWriter, r *http.Request) {
		log1.Debug().Str(logger.DirectionField, "←").Msgf("Handshake %v", r.Host)

		conn, err := connector.Connect(w, r)
		if err != nil {
			h.log.Error().Err(err).Msg("couldn't init user connection")
			return
		}
		client := com.NewConnection[com.Uid, api.PT, api.In[com.Uid], api.Out](conn, com.NewUid(), true, log1)
		cidLog := h.log.Extend(h.log.With().Str("cid", client.Id().Short()))
		user := &User{Connection: client, log: cidLog}
		defer h.users.RemoveDisconnect(user)
		done := user.HandleRequests(h, h.launcher, h.conf)

		worker := h.findWorkerFor(user, r.URL.Query())
		if worker == nil {
			h.log.Info().Msg("no free workers")
			return
		}

		user.SetWorker(worker)
		h.users.Add(user)
		user.InitSession(worker.Id().String(), h.conf.Webrtc.IceServers, h.launcher.GetAppNames())
		log1.Info().Str(logger.DirectionField, "+").Msgf("user %s", user.Id())
		<-done
	}
}

func RequestToHandshake(data string) (*api.ConnectionRequest[com.Uid], error) {
	if data == "" {
		return nil, api.ErrMalformed
	}
	handshake, err := api.UnwrapChecked[api.ConnectionRequest[com.Uid]](base64.URLEncoding.DecodeString(data))
	if err != nil || handshake == nil {
		return nil, fmt.Errorf("%v (%v)", err, handshake)
	}
	return handshake, nil
}

// handleWorkerConnection handles all connections from a new worker to coordinator.
func (h *Hub) handleWorkerConnection() http.HandlerFunc {
	connector := com.Server{}
	connector.Origin(h.conf.Coordinator.Origin.WorkerWs)

	cLog := h.log.Extend(h.log.With().Str(logger.ClientField, "w"))

	return func(w http.ResponseWriter, r *http.Request) {
		cLog.Debug().Str(logger.DirectionField, "←").Msgf("Handshake %v", r.Host)

		handshake, err := RequestToHandshake(r.URL.Query().Get(api.DataQueryParam))
		if err != nil {
			h.log.Error().Err(err).Msg("handshake fail")
			return
		}

		if handshake.PingURL == "" {
			h.log.Warn().Msg("Ping address is not set")
		}

		if h.conf.Coordinator.Server.Https && !handshake.IsHTTPS {
			h.log.Warn().Msg("Unsecure worker connection. Unsecure to secure may be bad.")
		}

		// set connection uid from the handshake
		if handshake.Id != com.NilUid {
			h.log.Debug().Msgf("Worker uid will be set to %v", handshake.Id)
		}

		conn, err := connector.Connect(w, r)
		if err != nil {
			h.log.Error().Err(err).Msg("couldn't init worker connection")
			return
		}
		client := com.NewConnection[com.Uid, api.PT, api.In[com.Uid], api.Out](conn, handshake.Id, true, cLog)
		cidLog := h.log.Extend(h.log.With().Str("cid", client.Id().Short()))
		worker := NewWorker(client, *handshake, cidLog)
		defer h.workers.RemoveDisconnect(worker)
		done := worker.HandleRequests(&h.users)
		h.workers.Add(worker)
		cLog.Info().Str(logger.DirectionField, "+").Msgf("worker %s", worker.PrintInfo())
		<-done
	}
}

func (h *Hub) GetServerList() (r []api.Server) {
	for _, w := range h.workers.List() {
		r = append(r, api.Server{
			Addr:    w.Addr,
			Id:      w.Id(),
			IsBusy:  !w.HasSlot(),
			Machine: string(w.Id().Machine()),
			PingURL: w.PingServer,
			Port:    w.Port,
			Tag:     w.Tag,
			Zone:    w.Zone,
		})
	}
	return
}

// findWorkerFor searches a free worker for the user depending on
// various conditions.
func (h *Hub) findWorkerFor(usr *User, q url.Values) *Worker {
	usr.log.Debug().Msg("Search available workers")
	roomId := q.Get(api.RoomIdQueryParam)
	zone := q.Get(api.ZoneQueryParam)
	wid := q.Get(api.WorkerIdParam)

	var worker *Worker
	if worker = h.findWorkerByRoom(roomId, zone); worker != nil {
		usr.log.Debug().Str("room", roomId).Msg("An existing worker has been found")
	} else if worker = h.findWorkerById(wid, h.conf.Coordinator.Debug); worker != nil {
		usr.log.Debug().Msgf("Worker with id: %v has been found", wid)
	} else {
		switch h.conf.Coordinator.Selector {
		case coordinator.SelectByPing:
			usr.log.Debug().Msgf("Searching fastest free worker...")
			if worker = h.findFastestWorker(zone,
				func(servers []string) (map[string]int64, error) { return usr.CheckLatency(servers) }); worker != nil {
				usr.log.Debug().Msg("The fastest worker has been found")
			}
		default:
			usr.log.Debug().Msgf("Searching any free worker...")
			if worker = h.find1stFreeWorker(zone); worker != nil {
				usr.log.Debug().Msgf("Found next free worker")
			}
		}
	}
	return worker
}

func (h *Hub) findWorkerByRoom(id string, region string) *Worker {
	if id == "" {
		return nil
	}
	// if there is zone param, we need to ensure the worker in that zone,
	// if not we consider the room is missing
	w, _ := h.workers.FindBy(func(w *Worker) bool { return w.RoomId == id && w.In(region) })
	return w
}

func (h *Hub) getAvailableWorkers(region string) []*Worker {
	var workers []*Worker
	h.workers.ForEach(func(w *Worker) {
		if w.HasSlot() && w.In(region) {
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
