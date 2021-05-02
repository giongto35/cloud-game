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
	"github.com/giongto35/cloud-game/v2/pkg/util"
)

type Hub struct {
	cfg      coordinator.Config
	launcher launcher.Launcher
	crowd    cache.Cache // stores users
	guild    Guild       // stores workers
	rooms    cache.Cache // stores user rooms
}

func NewHub(cfg coordinator.Config, lib games.GameLibrary) *Hub {
	return &Hub{
		cfg:      cfg,
		launcher: launcher.NewGameLauncher(lib),
		crowd:    cache.New(make(map[string]client.NetClient, 42)),
		guild:    NewGuild(),
		rooms:    cache.New(make(map[string]client.NetClient, 10)),
	}
}

// handleNewWebsocketUserConnection handles all connections from user/frontend.
func (h *Hub) handleNewWebsocketUserConnection(w http.ResponseWriter, r *http.Request) {
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
		usr.Printf("An existing wkr has been found for room [%v]", roomId)
	} else if wkr = h.findWorkerByIp(h.cfg.Coordinator.DebugHost); wkr != nil {
		usr.Printf("The wkr has been found with provided address: %v", h.cfg.Coordinator.DebugHost)
		if wkr = h.findAnyFreeWorker(region); wkr != nil {
			usr.Printf("A free wkr has been found right away")
		}
	} else if wkr = h.findFastestWorker(region,
		func(servers []string) (map[string]int64, error) { return usr.CheckLatency(servers) }); wkr != nil {
		usr.Printf("The fastest wkr has been found")
	} else {
		usr.Printf("error: THERE ARE NO FREE WORKERS")
		return
	}

	usr.AssignWorker(wkr)
	h.crowd.Add(uid, &usr)
	defer h.crowd.Remove(uid)
	usr.HandleRequests(h.launcher)
	usr.InitSession(h.cfg.Webrtc.IceServers, h.launcher.GetAppNames())

	usr.Listen()
	usr.RetainWorker()
	usr.Worker.TerminateSession(usr.Id())
	usr.Printf("Disconnected")
}

// handleNewWebsocketWorkerConnection handles all connections from a new worker to coordinator.
func (h *Hub) handleNewWebsocketWorkerConnection(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("error: recovered worker client from (%v)", r)
		}
	}()

	// !to add TLS stuff

	con, err := ipc.NewClientServer(w, r)
	if err != nil {
		log.Fatalf("error: couldn't init worker connection")
	}
	backend := NewWorker(con)
	backend.Printf("Connect")

	address := util.GetRemoteAddress(con.GetRemoteAddr())
	public := util.IsPublicIP(address)
	zone := r.URL.Query().Get("zone")
	pingServer := h.cfg.Coordinator.GetPingServer(zone)
	backend.Printf("info - address: %v, zone: %v, public: %v, ping server: %v", address, zone, public, pingServer)

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
	backend.HandleRequests(&h.rooms, &h.crowd)

	// Create a workerClient instance
	backend.Address = address
	backend.Region = zone
	backend.PingServer = pingServer

	h.guild.add(backend)
	defer h.cleanWorker(backend)
	backend.AssignId(backend.Id())

	backend.Listen()
	backend.Printf("Disconnect")
}

// cleanWorker is called when a worker is disconnected
// connection from worker to coordinator is also closed
func (h *Hub) cleanWorker(worker Worker) {
	h.guild.Remove(string(worker.Id()))
	worker.Close()
	h.rooms.RemoveAllWithId(string(worker.Id()))
}
