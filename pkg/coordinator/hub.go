package coordinator

import (
	"log"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/cache"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
	"github.com/giongto35/cloud-game/v2/pkg/service"
	"github.com/giongto35/cloud-game/v2/pkg/util"
)

type Hub struct {
	service.Service

	cfg      coordinator.Config
	launcher launcher.Launcher
	crowd    cache.Cache // stores users
	guild    Guild       // stores workers
	rooms    cache.Cache // stores user rooms
}

func NewHub(cfg coordinator.Config, lib games.GameLibrary) *Hub {
	// scan the lib right away
	lib.Scan()

	return &Hub{
		cfg:      cfg,
		launcher: launcher.NewGameLauncher(lib),
		crowd:    cache.New(make(map[string]client.NetClient, 42)),
		guild:    NewGuild(),
		rooms:    cache.New(make(map[string]client.NetClient, 10)),
	}
}

// handleWebsocketUserConnection handles all connections from user/frontend.
func (h *Hub) handleWebsocketUserConnection(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("error: recovered user client from (%v)", r)
		}
	}()

	conn, err := ipc.NewClientServer(w, r)
	if err != nil {
		log.Fatalf("error: couldn't init user connection")
	}
	usr := NewUser(conn)
	defer usr.Close()
	uid := string(usr.Id())
	usr.Printf("Connected")

	roomId := r.URL.Query().Get("room_id")
	region := r.URL.Query().Get("zone")

	usr.Printf("Searching for a free worker")
	var wkr *Worker
	if wkr = h.findWorkerByRoom(roomId, region); wkr != nil {
		usr.Printf("An existing worker has been found for room [%v]", roomId)
	} else if wkr = h.findWorkerByIp(h.cfg.Coordinator.DebugHost); wkr != nil {
		usr.Printf("The worker has been found with provided address: %v", h.cfg.Coordinator.DebugHost)
		if wkr = h.findAnyFreeWorker(region); wkr != nil {
			usr.Printf("A free worker has been found right away")
		}
	} else if wkr = h.findFastestWorker(region,
		func(servers []string) (map[string]int64, error) { return usr.CheckLatency(servers) }); wkr != nil {
		usr.Printf("The fastest worker has been found")
	} else {
		usr.Printf("error: no workers")
		return
	}

	usr.AssignWorker(wkr)

	wkr.ChangeUserQuantityBy(1)
	defer wkr.ChangeUserQuantityBy(-1)

	h.crowd.Add(uid, &usr)
	defer h.crowd.Remove(uid)
	usr.HandleRequests(h.launcher)
	usr.InitSession(h.cfg.Webrtc.IceServers, h.launcher.GetAppNames())

	usr.Listen()
	usr.RetainWorker()
	usr.Worker.TerminateSession(usr.Id())
	usr.Printf("Disconnected")
}

// handleWebsocketWorkerConnection handles all connections from a new worker to coordinator.
func (h *Hub) handleWebsocketWorkerConnection(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("error: recovered worker client from (%v)", r)
		}
	}()

	connRt, err := GetConnectionRequest(r.URL.Query().Get("data"))
	if err != nil {
		log.Printf("error: got a malformed request, %v", err.Error())
		return
	}

	if connRt.PingAddr == "" {
		log.Printf("Warning! Ping address is not set.")
	}

	if h.cfg.Coordinator.Server.Https && !connRt.IsHTTPS {
		log.Printf("Warning! Unsecure connection. The worker may not work properly without HTTPS on its side!")
	}

	conn, err := ipc.NewClientServer(w, r)
	if err != nil {
		log.Fatalf("error: couldn't init worker connection")
	}
	backend := NewWorker(conn)
	backend.Printf("Connect")
	defer backend.Close()

	address := util.GetRemoteAddress(conn.GetRemoteAddr())
	public := util.IsPublicIP(address)
	backend.Zone = connRt.Zone
	backend.PingServer = connRt.PingAddr
	backend.Printf("addr: %v | zone: %v | pub: %v | ping: %v", address, backend.Zone, public, backend.PingServer)

	// !to rewrite
	// In case wkr and coordinator in the same host
	if !public && h.cfg.Environment.Get() == environment.Production {
		// Don't accept private IP for wkr's address in prod mode
		// However, if the wkr in the same host with coordinator, we can get public IP of wkr
		backend.Printf("[!] Address %s is invalid", address)

		address = util.GetHostPublicIP()
		backend.Printf("Find public address: %s", address)

		if address == "" || !util.IsPublicIP(address) {
			// Skip this wkr because we cannot find public IP
			backend.Printf("[!] Unable to find public address, reject wkr")
			return
		}
	}
	backend.Address = address
	backend.HandleRequests(&h.rooms, &h.crowd)

	h.guild.add(&backend)
	defer func() {
		h.guild.Remove(&backend)
		h.rooms.RemoveAllWithId(string(backend.Id()))
	}()

	backend.Listen()
	backend.Printf("Disconnect")
}
