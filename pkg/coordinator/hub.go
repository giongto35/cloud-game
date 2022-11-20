package coordinator

import (
	"fmt"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
	"github.com/giongto35/cloud-game/v2/pkg/service"
	"github.com/rs/xid"
)

type Hub struct {
	service.Service

	conf     coordinator.Config
	launcher launcher.Launcher
	crowd    client.NetMap // stores users
	guild    Guild         // stores workers
	rooms    client.NetMap // stores user rooms
	log      *logger.Logger

	// custom ws upgrade handlers for Origin
	// !to encapsulate betterly
	wwsu, uwsu websocket.Upgrader
}

func NewHub(conf coordinator.Config, lib games.GameLibrary, log *logger.Logger) *Hub {
	h := &Hub{
		conf:     conf,
		crowd:    client.NewNetMap(),
		guild:    NewGuild(),
		launcher: launcher.NewGameLauncher(lib),
		rooms:    client.NewNetMap(),
		log:      log,
	}
	h.wwsu = websocket.NewUpgrader(conf.Coordinator.Origin.WorkerWs)
	h.uwsu = websocket.NewUpgrader(conf.Coordinator.Origin.UserWs)
	return h
}

// handleWebsocketUserConnection handles all connections from user/frontend.
func (h *Hub) handleWebsocketUserConnection(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			h.log.Error().Err(fmt.Errorf("%v", err)).Msg("user client crashed")
		}
	}()

	conn, err := ipc.NewClientServer(w, r, &h.uwsu, h.log)
	if err != nil {
		h.log.Error().Err(err).Msg("couldn't init user connection")
	}
	usr := NewUserClient(conn, h.log)
	defer h.cleanUser(&usr)
	usr.HandleRequests(h, h.launcher, h.conf)
	usr.ProcessMessages()

	q := r.URL.Query()
	roomId := q.Get(api.RoomIdQueryParam)
	zone := q.Get(api.ZoneQueryParam)
	wid := q.Get(api.WorkerIdParam)

	usr.GetLogger().Info().Msg("Search available workers")
	var wkr *Worker
	if wkr = h.findWorkerByRoom(roomId, zone); wkr != nil {
		usr.GetLogger().Info().Str("room", roomId).Msg("An existing worker has been found")
	} else if wkr = h.findWorkerById(wid, h.conf.Coordinator.Debug); wkr != nil {
		usr.GetLogger().Info().Msgf("Worker with id: %v has been found", wid)
	} else if wkr = h.findFastestWorker(zone,
		func(servers []string) (map[string]int64, error) { return usr.CheckLatency(servers) }); wkr != nil {
		usr.GetLogger().Info().Msg("The fastest worker has been found")
	}
	if wkr == nil {
		usr.GetLogger().Warn().Msg("no free workers")
		return
	}

	usr.SetWorker(wkr)
	h.crowd.Add(&usr)
	usr.InitSession(wkr.Id().String(), h.conf.Webrtc.IceServers, h.launcher.GetAppNames())
	usr.Wait()
}

// handleWebsocketWorkerConnection handles all connections from a new worker to coordinator.
func (h *Hub) handleWebsocketWorkerConnection(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			h.log.Error().Err(fmt.Errorf("%v", err)).Msg("worker client crashed")
		}
	}()

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

	conn, err := ipc.NewClientServer(w, r, &h.wwsu, h.log)
	if err != nil {
		h.log.Error().Err(err).Msg("couldn't init worker connection")
		return
	}

	var id network.Uid
	if handshake.Id != "" {
		if _, err := xid.FromString(handshake.Id); err != nil {
			id = network.NewUid()
		} else {
			id = network.Uid(handshake.Id)
		}
	} else {
		id = network.NewUid()
	}

	wc := NewWorkerClientWithId(id, conn, h.log)
	defer h.cleanWorker(&wc)

	wc.Addr = handshake.Addr
	wc.Zone = handshake.Zone
	wc.PingServer = handshake.PingURL
	wc.Port = handshake.Port
	wc.Tag = handshake.Tag

	h.log.Info().Msgf("New worker -- addr: %v, port: %v, zone: %v, ping addr: %v, tag: %v",
		wc.Addr, wc.Port, wc.Zone, wc.PingServer, wc.Tag)
	wc.HandleRequests(&h.rooms, &h.crowd)
	h.guild.add(&wc)
	wc.Listen()
}

func (h *Hub) getServerList() (r []api.Server) {
	workers := h.guild.filter(func(w *Worker) bool { return true })
	for _, w := range workers {
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

func (h *Hub) cleanWorker(w *Worker) {
	if w == nil {
		return
	}
	w.Close()
	h.guild.Remove(w)
	h.rooms.RemoveAll(w)
}

func (h *Hub) cleanUser(u *User) {
	if u == nil {
		return
	}
	u.Close()
	u.FreeWorker()
	h.crowd.Remove(u)
}
