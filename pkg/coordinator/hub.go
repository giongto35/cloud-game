package coordinator

import (
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/network"
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
}

func NewHub(conf coordinator.Config, lib games.GameLibrary, log *logger.Logger) *Hub {
	// scan the lib right away
	lib.Scan()

	return &Hub{
		conf:     conf,
		crowd:    client.NewNetMap(make(map[string]client.NetClient, 42)),
		guild:    NewGuild(),
		launcher: launcher.NewGameLauncher(lib),
		rooms:    client.NewNetMap(make(map[string]client.NetClient, 10)),
		log:      log,
	}
}

// handleWebsocketUserConnection handles all connections from user/frontend.
func (h *Hub) handleWebsocketUserConnection(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			h.log.Error().Msgf("recovered user client from (%v)", r)
		}
	}()

	conn, err := ipc.NewClientServer(w, r, h.log)
	if err != nil {
		h.log.Error().Err(err).Msg("couldn't init user connection")
	}
	usr := NewUserClient(conn, h.log)
	defer usr.Close()

	q := r.URL.Query()
	roomId := q.Get("room_id")
	region := q.Get("zone")

	usr.GetLogger().Info().Msg("Searching for a free worker")
	var wkr *Worker
	if wkr = h.findWorkerByRoom(roomId, region); wkr != nil {
		usr.GetLogger().Info().Str("room", roomId).Msg("An existing worker has been found")
	} else if wkr = h.findWorkerByIp(h.conf.Coordinator.DebugHost); wkr != nil {
		usr.GetLogger().Info().Str("debug.addr", h.conf.Coordinator.DebugHost).
			Msg("The worker has been found with the provided address")
		if wkr = h.findAnyFreeWorker(region); wkr != nil {
			usr.GetLogger().Info().Msg("A free worker has been found right away")
		}
	} else if wkr = h.findFastestWorker(region,
		func(servers []string) (map[string]int64, error) { return usr.CheckLatency(servers) }); wkr != nil {
		usr.GetLogger().Info().Msg("The fastest worker has been found")
	} else {
		usr.GetLogger().Warn().Msg("no free workers")
		return
	}

	usr.SetWorker(wkr)
	defer usr.FreeWorker()
	h.crowd.Add(&usr)
	defer h.crowd.Remove(&usr)
	usr.HandleRequests(h.launcher)
	usr.InitSession(h.conf.Webrtc.IceServers, h.launcher.GetAppNames())
	usr.Listen()
}

// handleWebsocketWorkerConnection handles all connections from a new worker to coordinator.
func (h *Hub) handleWebsocketWorkerConnection(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			h.log.Error().Msgf("recovered worker client from (%v)", r)
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

	conn, err := ipc.NewClientServer(w, r, h.log)
	if err != nil {
		h.log.Error().Err(err).Msg("couldn't init worker connection")
	}
	backend := NewWorkerClient(conn, h.log)
	defer backend.Close()

	address := network.GetRemoteAddress(conn.GetRemoteAddr())
	public := network.IsPublicIP(address)
	backend.Zone = connRt.Zone
	backend.PingServer = connRt.PingAddr
	h.log.Info().
		Fields(map[string]interface{}{
			"pub":  public,
			"addr": address,
			"zone": backend.Zone,
			"ping": backend.PingServer,
		}).
		Msg("Worker info")

	// !to rewrite
	// In case wkr and coordinator in the same host
	if !public && h.conf.Environment.Get() == environment.Production {
		// Don't accept private IP for wkr's address in prod mode
		// However, if the wkr in the same host with coordinator, we can get public IP of wkr
		backend.GetLogger().Warn().Msgf("Invalid address [%s]", address)

		address = network.GetHostPublicIP()
		backend.GetLogger().Info().Msgf("Find public address [%s]", address)

		if address == "" || !network.IsPublicIP(address) {
			// Skip this wkr because we cannot find public IP
			backend.GetLogger().Error().Msg("unable to find public address, rejecting worker")
			return
		}
	}
	backend.Address = address
	backend.HandleRequests(&h.rooms, &h.crowd)
	h.guild.add(&backend)
	defer func() {
		h.guild.Remove(&backend)
		h.rooms.RemoveAll(&backend)
	}()
	backend.Listen()
}
