package coordinator

import (
	"fmt"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
	"github.com/giongto35/cloud-game/v2/pkg/service"
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
	// scan the lib right away
	lib.Scan()

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
	usr.HandleRequests(h.launcher, h.conf)
	defer usr.Close()
	usr.ProcessMessages()

	q := r.URL.Query()
	roomId := q.Get("room_id")
	zone := q.Get("zone")

	usr.GetLogger().Info().Msg("Search available servers")
	var wkr *Worker
	if wkr = h.findWorkerByRoom(roomId, zone); wkr != nil {
		usr.GetLogger().Info().Str("room", roomId).Msg("An existing worker has been found")
	} else if wkr = h.findFastestWorker(zone,
		func(servers []string) (map[string]int64, error) { return usr.CheckLatency(servers) }); wkr != nil {
		usr.GetLogger().Info().Msg("The fastest worker has been found")
	}
	if wkr == nil {
		usr.GetLogger().Warn().Msg("no free workers")
		return
	}

	usr.SetWorker(wkr)
	defer usr.FreeWorker()
	h.crowd.Add(&usr)
	defer h.crowd.Remove(&usr)
	usr.InitSession(h.conf.Webrtc.IceServers, h.launcher.GetAppNames())
	usr.Wait()
}

// handleWebsocketWorkerConnection handles all connections from a new worker to coordinator.
func (h *Hub) handleWebsocketWorkerConnection(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := recover(); err != nil {
			h.log.Error().Err(fmt.Errorf("%v", err)).Msg("worker client crashed")
		}
	}()

	connRt, err := GetConnectionRequest(r.URL.Query().Get("data"))
	if err != nil {
		h.log.Error().Err(err).Msg("got a malformed request")
		return
	}

	if connRt.PingAddr == "" {
		h.log.Warn().Msg("Ping address is not set")
	}

	if h.conf.Coordinator.Server.Https && !connRt.IsHTTPS {
		h.log.Warn().Msg("Unsecure connection. The worker may not work properly without HTTPS on its side!")
	}

	conn, err := ipc.NewClientServer(w, r, &h.wwsu, h.log)
	if err != nil {
		h.log.Error().Err(err).Msg("couldn't init worker connection")
		return
	}
	backend := NewWorkerClient(conn, h.log)
	defer backend.Close()

	backend.Zone = connRt.Zone
	backend.PingServer = connRt.PingAddr
	h.log.Info().
		Fields(map[string]interface{}{
			"addr": conn.GetRemoteAddr(),
			"zone": backend.Zone,
			"ping": backend.PingServer,
		}).
		Msg("Worker info")
	backend.HandleRequests(&h.rooms, &h.crowd)
	h.guild.add(&backend)
	defer func() {
		h.guild.Remove(&backend)
		h.rooms.RemoveAll(&backend)
	}()
	backend.Listen()
}
