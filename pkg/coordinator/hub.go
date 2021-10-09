package coordinator

import (
	"log"
	"net/http"

	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
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
}

func NewHub(conf coordinator.Config, lib games.GameLibrary) *Hub {
	// scan the lib right away
	lib.Scan()

	return &Hub{
		conf:     conf,
		launcher: launcher.NewGameLauncher(lib),
		crowd:    client.NewNetMap(make(map[string]client.NetClient, 42)),
		guild:    NewGuild(),
		rooms:    client.NewNetMap(make(map[string]client.NetClient, 10)),
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
	usr := NewUserClient(conn)
	defer usr.Logf("Disconnected")
	defer usr.Close()
	usr.Logf("Connected")

	q := r.URL.Query()
	roomId := q.Get("room_id")
	region := q.Get("zone")

	usr.Logf("Searching for a free worker")
	var wkr *Worker
	if wkr = h.findWorkerByRoom(roomId, region); wkr != nil {
		usr.Logf("An existing worker has been found for room [%v]", roomId)
	} else if wkr = h.findWorkerByIp(h.conf.Coordinator.DebugHost); wkr != nil {
		usr.Logf("The worker has been found with provided address: %v", h.conf.Coordinator.DebugHost)
		if wkr = h.findAnyFreeWorker(region); wkr != nil {
			usr.Logf("A free worker has been found right away")
		}
	} else if wkr = h.findFastestWorker(region,
		func(servers []string) (map[string]int64, error) { return usr.CheckLatency(servers) }); wkr != nil {
		usr.Logf("The fastest worker has been found")
	} else {
		usr.Logf("error: no workers")
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

	if h.conf.Coordinator.Server.Https && !connRt.IsHTTPS {
		log.Printf("Warning! Unsecure connection. The worker may not work properly without HTTPS on its side!")
	}

	conn, err := ipc.NewClientServer(w, r)
	if err != nil {
		log.Fatalf("error: couldn't init worker connection")
	}
	backend := NewWorkerClient(conn)
	backend.Logf("Connect")
	defer backend.Logf("Disconnect")
	defer backend.Close()

	address := network.GetRemoteAddress(conn.GetRemoteAddr())
	public := network.IsPublicIP(address)
	backend.Zone = connRt.Zone
	backend.PingServer = connRt.PingAddr
	backend.Logf("addr: %v | zone: %v | pub: %v | ping: %v", address, backend.Zone, public, backend.PingServer)

	// !to rewrite
	// In case wkr and coordinator in the same host
	if !public && h.conf.Environment.Get() == environment.Production {
		// Don't accept private IP for wkr's address in prod mode
		// However, if the wkr in the same host with coordinator, we can get public IP of wkr
		backend.Logf("[!] Address %s is invalid", address)

		address = network.GetHostPublicIP()
		backend.Logf("Find public address: %s", address)

		if address == "" || !network.IsPublicIP(address) {
			// Skip this wkr because we cannot find public IP
			backend.Logf("[!] Unable to find public address, reject wkr")
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
