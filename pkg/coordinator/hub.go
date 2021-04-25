package coordinator

import (
	"log"
	"net/http"
	"unsafe"

	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/coordinator/user"
	"github.com/giongto35/cloud-game/v2/pkg/coordinator/worker"
	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	ws "github.com/giongto35/cloud-game/v2/pkg/network/websocket"
	"github.com/giongto35/cloud-game/v2/pkg/util"
)

type Hub struct {
	cfg     coordinator.Config
	library games.GameLibrary
	crowd   Crowd
	guild   Guild
	rooms   map[string]worker.WorkerClient
}

func NewHub(cfg coordinator.Config, library games.GameLibrary) *Hub {
	return &Hub{
		cfg:     cfg,
		library: library,
		crowd:   NewCrowd(),
		guild:   NewGuild(),
		rooms:   map[string]worker.WorkerClient{},
	}
}

// handleNewWebsocketUserConnection handles all connections from user/frontend.
func (h *Hub) handleNewWebsocketUserConnection(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Something wrong. Recovered in ", r)
		}
	}()

	conn, err := ipc.NewClientServer(w, r)
	if err != nil {
		log.Fatalf("error: couldn't start usr handler")
	}
	usr := user.New(conn)
	log.Printf("new usr: %v", usr.Id)

	// Server will pair the frontend with the server running the room.
	// It only happens when we are trying to access a running room over share link.
	// TODO: Update link to the wiki
	roomId := r.URL.Query().Get("room_id")
	region := r.URL.Query().Get("zone")

	// O_o
	usr.Printf("Trying to find some wkr")
	var wkr *worker.WorkerClient
	if wkr = h.findWorkerByRoom(roomId, region); wkr != nil {
		usr.Printf("An existing wkr has been found for room [%v]", roomId)
		goto connection
	}
	if wkr = h.findWorkerByIp(h.cfg.Coordinator.DebugHost); wkr != nil {
		usr.Printf("The wkr has been found with provided address: %v", h.cfg.Coordinator.DebugHost)
		goto connection
	}
	if h.cfg.Coordinator.RoundRobin {
		if wkr = h.findAnyFreeWorker(region); wkr != nil {
			usr.Printf("A free wkr has been found right away")
			goto connection
		}
	}
	if wkr = h.findFastestWorker(region,
		func(servers []string) (map[string]int64, error) { return usr.CheckLatency(servers) }); wkr != nil {
		usr.Printf("The fastest wkr has been found")
		goto connection
	}

	usr.Printf("error: THERE ARE NO FREE WORKERS")
	return

connection:
	usr.AssignWorker(wkr)
	h.crowd.add(usr)
	defer h.crowd.finish(usr)

	usr.HandleRequests(h.library)

	usr.InitSession(user.InitSessionRequest{
		// don't do this at home
		Ice:   *(*[]user.IceServer)(unsafe.Pointer(&h.cfg.Webrtc.IceServers)),
		Games: h.getGames(),
	})

	usr.WaitDisconnect()
	usr.RetainWorker()

	// Notify wkr to clean session
	wkr.SendPacket(api.TerminateSessionPacket(usr.Id))
	usr.Println("Disconnect from coordinator")
}

// handleNewWebsocketWorkerConnection handles all connections from a new worker to coordinator.
func (h *Hub) handleNewWebsocketWorkerConnection(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Something wrong. Recovered in ", r)
		}
	}()

	conn, err := ws.Upgrade(w, r)
	if err != nil {
		log.Printf("error: socket upgrade failed because of %v", err)
		return
	}

	wkr := worker.NewWorkerClient(conn, network.NewUid())
	log.Printf("new wkr: %v", wkr.Id)

	// Register to workersClients map the client connection
	address := util.GetRemoteAddress(conn)
	wkr.Println("Address:", address)
	// Region of the wkr
	zone := r.URL.Query().Get("zone")
	wkr.Printf("Is public: %v zone: %v", util.IsPublicIP(address), zone)

	pingServer := h.cfg.Coordinator.GetPingServer(zone)

	wkr.Printf("Set ping server address: %s", pingServer)

	// In case wkr and coordinator in the same host
	if !util.IsPublicIP(address) && h.cfg.Environment.Get() == environment.Production {
		// Don't accept private IP for wkr's address in prod mode
		// However, if the wkr in the same host with coordinator, we can get public IP of wkr
		wkr.Printf("[!] Address %s is invalid", address)

		address = util.GetHostPublicIP()
		wkr.Printf("Find public address: %s", address)

		if address == "" || !util.IsPublicIP(address) {
			// Skip this wkr because we cannot find public IP
			wkr.Println("[!] Unable to find public address, reject wkr")
			return
		}
	}

	// Create a workerClient instance
	wkr.Address = address
	//wkr.StunTurnServer = ice.ToJson(h.cfg.Webrtc.IceServers, ice.Replacement{From: "server-ip", To: address})
	wkr.Region = zone
	wkr.PingServer = pingServer

	// Attach to Server instance with workerID, add defer
	h.guild.add(wkr)
	defer h.cleanWorker(wkr)

	wkr.SendPacket(api.ServerIdPacket(wkr.Id))

	h.workerRoutes(wkr)
	wkr.Listen()
}

// cleanWorker is called when a worker is disconnected
// connection from worker to coordinator is also closed
func (h *Hub) cleanWorker(worker worker.WorkerClient) {
	h.guild.remove(worker)
	// Clean all rooms connecting to that server
	for roomID, roomServer := range h.rooms {
		if roomServer == worker {
			worker.Printf("Remove room %s", roomID)
			delete(h.rooms, roomID)
		}
	}
}

func (h *Hub) getGames() []string {
	var gameList []string
	for _, game := range h.library.GetAll() {
		gameList = append(gameList, game.Name)
	}
	return gameList
}

// workerRoutes adds all worker request routes.
func (h *Hub) workerRoutes(wc worker.WorkerClient) {
	wc.Receive(api.Heartbeat, func(resp cws.WSPacket) (req cws.WSPacket) {
		return resp
	})
	wc.Receive(api.RegisterRoom, func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Printf("Coordinator: Received registerRoom room %s from worker %s", resp.Data, wc.Id)
		h.rooms[resp.Data] = wc
		log.Printf("Coordinator: Current room list is: %+v", h.rooms)
		return api.RegisterRoomPacket(api.NoData)
	})
	wc.Receive(api.GetRoom, func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Coordinator: Received a get room request")
		log.Println("Result: ", h.rooms[resp.Data])
		return api.GetRoomPacket(string(h.rooms[resp.Data].Id))
	})
	wc.Receive(api.CloseRoom, func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Printf("Coordinator: Received closeRoom room %s from worker %s", resp.Data, wc.Id)
		delete(h.rooms, resp.Data)
		log.Printf("Coordinator: Current room list is: %+v", h.rooms)
		return api.CloseRoomPacket(api.NoData)
	})
	wc.Receive(api.IceCandidate, func(resp cws.WSPacket) (req cws.WSPacket) {
		wc.Println("relay IceCandidate to useragent")
		usr := h.crowd.findById(resp.SessionID)
		if usr != nil {
			// Remove SessionID while sending back to browser
			resp.SessionID = ""
			usr.SendWebrtcIceCandidate(resp.Data)
		} else {
			wc.Println("error: unknown SessionID:", resp.SessionID)
		}
		return cws.EmptyPacket
	})
}
