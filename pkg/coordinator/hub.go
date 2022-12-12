package coordinator

import (
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

type Hub struct {
	service.Service

	conf          coordinator.Config
	launcher      games.Launcher
	users         com.NetMap[*User]
	workers       com.NetMap[*Worker]
	rooms2workers com.NetMap[com.NetClient]
	log           *logger.Logger

	wConn, uConn *com.Connector
}

func NewHub(conf coordinator.Config, lib games.GameLibrary, log *logger.Logger) *Hub {
	return &Hub{
		conf:          conf,
		users:         com.NewNetMap[*User](),
		workers:       com.NewNetMap[*Worker](),
		rooms2workers: com.NewNetMap[com.NetClient](),
		launcher:      games.NewGameLauncher(lib),
		log:           log,
		wConn: com.NewConnector(
			com.WithOrigin(conf.Coordinator.Origin.WorkerWs),
			com.WithTag("w"),
		),
		uConn: com.NewConnector(
			com.WithOrigin(conf.Coordinator.Origin.UserWs),
			com.WithTag("u"),
		),
	}
}

// handleUserConnection handles all connections from user/frontend.
func (h *Hub) handleUserConnection(w http.ResponseWriter, r *http.Request) {
	h.log.Info().Str("c", "u").Str("d", "←").Msgf("Handshake %v", r.Host)
	usr, err := NewUserClientServer(h.uConn.NewClientServer(w, r, h.log))
	if err != nil {
		h.log.Error().Err(err).Msg("couldn't init user connection")
	}
	defer func() {
		if usr != nil {
			usr.Disconnect()
			h.users.Remove(usr)
		}
	}()
	usr.HandleRequests(h, h.launcher, h.conf)

	q := r.URL.Query()
	roomId := q.Get(api.RoomIdQueryParam)
	zone := q.Get(api.ZoneQueryParam)
	wid := q.Get(api.WorkerIdParam)

	usr.Log.Info().Msg("Search available workers")
	var wkr *Worker
	if wkr = h.findWorkerByRoom(roomId, zone); wkr != nil {
		usr.Log.Info().Str("room", roomId).Msg("An existing worker has been found")
	} else if wkr = h.findWorkerById(wid, h.conf.Coordinator.Debug); wkr != nil {
		usr.Log.Info().Msgf("Worker with id: %v has been found", wid)
	} else if wkr = h.findFastestWorker(zone,
		func(servers []string) (map[string]int64, error) { return usr.CheckLatency(servers) }); wkr != nil {
		usr.Log.Info().Msg("The fastest worker has been found")
	}
	if wkr == nil {
		usr.Log.Warn().Msg("no free workers")
		return
	}

	usr.SetWorker(wkr)
	h.users.Add(usr)
	usr.InitSession(wkr.Id().String(), h.conf.Webrtc.IceServers, h.launcher.GetAppNames())
	<-usr.Done()
}

// handleWorkerConnection handles all connections from a new worker to coordinator.
func (h *Hub) handleWorkerConnection(w http.ResponseWriter, r *http.Request) {
	h.log.Info().Str("c", "w").Str("d", "←").Msgf("Handshake %v", r.Host)

	data := r.URL.Query().Get(api.DataQueryParam)
	handshake, err := GetConnectionRequest(data)
	if err != nil || handshake == nil {
		h.log.Error().Err(err).Msg("got a malformed request")
		return
	}

	if handshake.PingURL == "" {
		h.log.Warn().Msg("Ping address is not set")
	}

	if h.conf.Coordinator.Server.Https && !handshake.IsHTTPS {
		h.log.Warn().Msg("Unsecure connection. The worker may not work properly without HTTPS on its side!")
	}

	conn, err := h.wConn.NewClientServer(w, r, h.log)
	if err != nil {
		h.log.Error().Err(err).Msg("couldn't init worker connection")
		return
	}

	worker := &Worker{
		SocketClient: *conn,
		Addr:         handshake.Addr,
		PingServer:   handshake.PingURL,
		Port:         handshake.Port,
		Tag:          handshake.Tag,
		Zone:         handshake.Zone,
	}
	// we duplicate uid from the handshake
	hid := network.Uid(handshake.Id)
	if !(handshake.Id == "" || !network.ValidUid(hid)) {
		conn.SetId(hid)
		worker.Log.Debug().Msgf("connection id has been changed to %s", hid)
	}
	defer func() {
		if worker != nil {
			worker.Disconnect()
			h.workers.Remove(worker)
			h.rooms2workers.RemoveAll(worker)
		}
	}()

	h.log.Info().Msgf("New worker / addr: %v, port: %v, zone: %v, ping addr: %v, tag: %v",
		worker.Addr, worker.Port, worker.Zone, worker.PingServer, worker.Tag)
	worker.HandleRequests(&h.rooms2workers, &h.users)
	h.workers.Add(worker)
	worker.Listen()
}

func (h *Hub) getServerList() (r []api.Server) {
	for _, w := range h.workers.List() {
		r = append(r, api.Server{
			Addr:    w.Addr,
			Id:      w.Id(),
			IsBusy:  !w.HasGameSlot(),
			PingURL: w.PingServer,
			Port:    w.Port,
			Tag:     w.Tag,
			Zone:    w.Zone,
		})
	}
	return
}
