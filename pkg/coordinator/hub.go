package coordinator

import (
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/comm"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/service"
)

type Hub struct {
	service.Service

	conf          coordinator.Config
	launcher      launcher.Launcher
	users         comm.NetMap[*User]
	workers       comm.NetMap[*Worker]
	rooms2workers comm.NetMap[comm.NetClient]
	log           *logger.Logger

	wConn, uConn *comm.Connector
}

func NewHub(conf coordinator.Config, lib games.GameLibrary, log *logger.Logger) *Hub {
	return &Hub{
		conf:          conf,
		users:         comm.NewNetMap[*User](),
		workers:       comm.NewNetMap[*Worker](),
		rooms2workers: comm.NewNetMap[comm.NetClient](),
		launcher:      launcher.NewGameLauncher(lib),
		log:           log,
		wConn: comm.NewConnector(
			comm.WithOrigin(conf.Coordinator.Origin.WorkerWs),
			comm.WithTag("w"),
		),
		uConn: comm.NewConnector(
			comm.WithOrigin(conf.Coordinator.Origin.UserWs),
			comm.WithTag("u"),
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
	usr.ProcessMessages()

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
	usr.Wait()
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
	wc := NewWorkerClientServer(network.Uid(handshake.Id), conn)
	defer func() {
		if wc == nil {
			return
		}
		wc.Disconnect()
		h.workers.Remove(wc)
		h.rooms2workers.RemoveAll(wc)
	}()

	wc.Addr = handshake.Addr
	wc.Zone = handshake.Zone
	wc.PingServer = handshake.PingURL
	wc.Port = handshake.Port
	wc.Tag = handshake.Tag

	h.log.Info().Msgf("New worker -- addr: %v, port: %v, zone: %v, ping addr: %v, tag: %v",
		wc.Addr, wc.Port, wc.Zone, wc.PingServer, wc.Tag)
	wc.HandleRequests(&h.rooms2workers, &h.users)
	h.workers.Add(wc)
	wc.Listen()
}

func (h *Hub) getServerList() (r []api.Server) {
	for _, w := range h.workers.List() {
		r = append(r, api.Server{
			Addr:    w.Addr,
			Id:      w.Id().String(),
			IsBusy:  !w.HasGameSlot(),
			PingURL: w.PingServer,
			Port:    w.Port,
			Tag:     w.Tag,
			Zone:    w.Zone,
		})
	}
	return
}
