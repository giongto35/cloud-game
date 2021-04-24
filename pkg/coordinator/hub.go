package coordinator

import (
	"fmt"
	"log"
	"net/http"
	"strconv"

	api2 "github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
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

	guild Guild
	rooms map[string]*WorkerClient
	users map[network.Uid]*User
}

func NewRouter(cfg coordinator.Config, library games.GameLibrary) *Hub {
	return &Hub{
		cfg:     cfg,
		library: library,

		guild: NewGuild(),
		rooms: map[string]*WorkerClient{},
		users: map[network.Uid]*User{},
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
		log.Fatalf("error: couldn't start user handler")
	}
	user := NewUser(conn)
	log.Printf("new user: %v", user.id)

	// Server will pair the frontend with the server running the room.
	// It only happens when we are trying to access a running room over share link.
	// TODO: Update link to the wiki
	roomId := r.URL.Query().Get("room_id")
	region := r.URL.Query().Get("zone")

	// O_o
	user.Printf("Trying to find some worker")
	var worker *WorkerClient
	if worker = h.findWorkerByRoom(roomId, region); worker != nil {
		user.Printf("An existing worker has been found for room [%v]", roomId)
		goto connection
	}
	if worker = h.findWorkerByIp(h.cfg.Coordinator.DebugHost); worker != nil {
		user.Printf("The worker has been found with provided address: %v", h.cfg.Coordinator.DebugHost)
		goto connection
	}
	if h.cfg.Coordinator.RoundRobin {
		if worker = h.findAnyFreeWorker(region); worker != nil {
			user.Printf("A free worker has been found right away")
			goto connection
		}
	}
	if worker = h.findFastestWorker(region, func(addresses []string) (error, map[string]int64) {
		// send this address to user and get back latency
		user.Printf("Ping addresses: %v", addresses)
		data, err := user.Send(api2.P_latencyCheck, addresses)
		if err != nil {
			log.Printf("can't get a response with latencies %v", err)
			return err, map[string]int64{}
		}
		ll := api2.Latencies{}
		err = ll.FromResponse(data)
		if err != nil {
			log.Printf("can't convert user latencies, %v", err)
			return err, map[string]int64{}
		}
		return nil, ll
	}); worker != nil {
		user.Printf("The fastest worker has been found")
		goto connection
	}

	user.Printf("error: THERE ARE NO FREE WORKERS")
	return

connection:
	user.Printf("Assigned worker: %v", worker.Id)

	user.AssignWorker(worker)
	h.users[user.id] = user
	defer h.cleanUser(user)

	h.useragentRoutes(user)

	_, _ = user.SendAndForget(api2.P_init, initPacket(h.cfg.Webrtc.IceServers, h.library.GetAll()))

	user.WaitDisconnect()
	user.RetainWorker()

	// Notify worker to clean session
	worker.SendPacket(api.TerminateSessionPacket(user.id))
}

// WSO handles all connections from a new worker to coordinator
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

	worker := NewWorkerClient(conn, network.NewUid())
	log.Printf("new worker: %v", worker.Id)

	// Register to workersClients map the client connection
	address := util.GetRemoteAddress(conn)
	worker.Println("Address:", address)
	// Region of the worker
	zone := r.URL.Query().Get("zone")
	worker.Printf("Is public: %v zone: %v", util.IsPublicIP(address), zone)

	pingServer := h.getPingServer(zone)

	worker.Printf("Set ping server address: %s", pingServer)

	// In case worker and coordinator in the same host
	if !util.IsPublicIP(address) && h.cfg.Environment.Get() == environment.Production {
		// Don't accept private IP for worker's address in prod mode
		// However, if the worker in the same host with coordinator, we can get public IP of worker
		worker.Printf("[!] Address %s is invalid", address)

		address = util.GetHostPublicIP()
		worker.Printf("Find public address: %s", address)

		if address == "" || !util.IsPublicIP(address) {
			// Skip this worker because we cannot find public IP
			worker.Println("[!] Unable to find public address, reject worker")
			return
		}
	}

	// Create a workerClient instance
	worker.Address = address
	//worker.StunTurnServer = ice.ToJson(h.cfg.Webrtc.IceServers, ice.Replacement{From: "server-ip", To: address})
	worker.Region = zone
	worker.PingServer = pingServer

	// Attach to Server instance with workerID, add defer
	h.guild.add(worker)
	defer h.cleanWorker(worker)

	worker.SendPacket(api.ServerIdPacket(worker.Id))

	h.workerRoutes(worker)
	worker.Listen()
}

func (h *Hub) cleanUser(user *User) {
	user.Println("Disconnect from coordinator")
	delete(h.users, user.id)
	user.Clean()
}

// cleanWorker is called when a worker is disconnected
// connection from worker to coordinator is also closed
func (h *Hub) cleanWorker(worker *WorkerClient) {
	h.guild.remove(worker)
	// Clean all rooms connecting to that server
	for roomID, roomServer := range h.rooms {
		if roomServer == worker {
			worker.Printf("Remove room %s", roomID)
			delete(h.rooms, roomID)
		}
	}
}

// getPingServer returns the server for latency check of a zone.
// In latency check to find best worker step, we use this server to find the closest worker.
func (h *Hub) getPingServer(zone string) string {
	if h.cfg.Coordinator.PingServer != "" {
		return fmt.Sprintf("%s/echo", h.cfg.Coordinator.PingServer)
	}

	if h.cfg.Coordinator.Server.Https && h.cfg.Coordinator.Server.Tls.Domain != "" {
		return fmt.Sprintf(pingServerTemp, zone, h.cfg.Coordinator.Server.Tls.Domain)
	}
	return devPingServer
}

// useragentRoutes adds all useragent (browser) request routes.
func (h *Hub) useragentRoutes(user *User) {
	if user == nil {
		return
	}

	user.Handle(func(p ipc.Packet) {

		switch p.T {
		case ipc.PacketType(api2.P_webrtc_init):
			if user.Worker == nil {
				//return cws.EmptyPacket
				return
			}

			// initWebrtc now only sends signal to worker, asks it to createOffer
			user.Printf("Received init_webrtc request -> relay to worker: %s", user.Worker)
			// relay request to target worker
			// worker creates a PeerConnection, and createOffer
			// send SDP back to browser

			defer user.Println("Received SDP from worker -> sending back to browser")
			resp := user.Worker.SyncSend(cws.WSPacket{ID: api.InitWebrtc, SessionID: user.id})

			if resp != cws.EmptyPacket && resp.ID == api.Offer {
				rez, err := user.SendAndForget(api2.P_webrtc_offer, resp.Data)
				log.Printf("OFFER REZ %+v, err: %v", rez, err)
			}
		case ipc.PacketType(api2.P_webrtc_answer):
			user.Println("Received browser answered SDP -> relay to worker")
			user.Worker.SendPacket(cws.WSPacket{ID: api.Answer, SessionID: user.id, Data: p.Payload.(string)})
		case ipc.PacketType(api2.P_webrtc_ice_candidate):
			user.Println("Received IceCandidate from browser -> relay to worker")
			pack := cws.WSPacket{ID: api.IceCandidate, SessionID: user.id, Data: p.Payload.(string)}
			user.Worker.SendPacket(pack)
		case ipc.PacketType(api2.P_game_start):
			user.Println("Received start request from a browser -> relay to worker")
			// +injects game data into the original game request
			request := api.GameStartRequest{}
			if err := request.From(p.Payload.(string)); err != nil {
				user.Printf("err: %v", err)
				return
			}
			gameStartCall, err := newNewGameStartCall(request, h.library)
			if err != nil {
				user.Printf("err: %v", err)
				return
			}
			packet, err := gameStartCall.To()
			if err != nil {
				user.Printf("err: %v", err)
				return
			}

			workerResp := user.Worker.SyncSend(cws.WSPacket{
				ID: api.Start, SessionID: user.id, RoomID: request.RoomId, Data: packet})

			// Response from worker contains initialized roomID. Set roomID to the session
			user.RoomID = workerResp.RoomID
			user.Println("Received room response from browser: ", workerResp.RoomID)

			_, err = user.SendAndForget(api2.P_game_start, user.RoomID)
			if err != nil {
				user.Printf("can't send back start request")
				return
			}
		case ipc.PacketType(api2.P_game_quit):
			user.Println("Received quit request from a browser -> relay to worker")
			request := api.GameQuitRequest{}
			if err := request.From(p.Payload.(string)); err != nil {
				user.Printf("err: %v", err)
				return
			}
			user.Worker.SyncSend(cws.WSPacket{ID: api.GameQuit, SessionID: user.id, RoomID: request.RoomId})
		case ipc.PacketType(api2.P_game_save):
			user.Println("Received save request from a browser -> relay to worker")
			// TODO: Async
			response := user.Worker.SyncSend(cws.WSPacket{ID: api.GameSave, SessionID: user.id, RoomID: user.RoomID})
			user.Printf("SAVE result: %v", response.Data)
		case ipc.PacketType(api2.P_game_load):
			user.Println("Received load request from a browser -> relay to worker")
			// TODO: Async
			response := user.Worker.SyncSend(cws.WSPacket{ID: api.GameLoad, SessionID: user.id, RoomID: user.RoomID})
			user.Printf("LOAD result: %v", response.Data)
		case ipc.PacketType(api2.P_game_set_player_index):
			user.Println("Received update player index request from a browser -> relay to worker")
			user.Printf("val is %v", p.Payload)
			v, ok := p.Payload.(string)
			if !ok {
				user.Printf("can't convert %v", v)
				return
			}
			// TODO: Async
			response := user.Worker.SyncSend(cws.WSPacket{
				ID:        api.GamePlayerSelect,
				SessionID: user.id,
				RoomID:    user.RoomID,
				Data:      v,
			})
			user.Printf("Player index result: %v", response.Data)

			if response.Data == "error" {
				user.Printf("Player switch failed for some reason")
			}

			idx, _ := strconv.Atoi(response.Data)

			_, _ = user.SendAndForget(api2.P_game_set_player_index, idx)
		case ipc.PacketType(api2.P_game_toggle_multitap):
			user.Println("Received multitap request from a browser -> relay to worker")
			// TODO: Async
			response := user.Worker.SyncSend(cws.WSPacket{ID: api.GameMultitap, SessionID: user.id, RoomID: user.RoomID})
			user.Printf("MULTITAP result: %v", response.Data)
		}
	})
}

// workerRoutes adds all worker request routes.
func (h *Hub) workerRoutes(wc *WorkerClient) {
	if wc == nil {
		return
	}
	wc.Receive(api.Heartbeat, wc.handleHeartbeat())
	wc.Receive(api.RegisterRoom, wc.handleRegisterRoom2(h))
	wc.Receive(api.GetRoom, wc.handleGetRoom2(h))
	wc.Receive(api.CloseRoom, wc.handleCloseRoom2(h))
	wc.Receive(api.IceCandidate, wc.handleIceCandidate2(h))
}

func initPacket(servers []webrtc.IceServer, games []games.GameMetadata) api2.InitPack {
	var gameName []string
	for _, game := range games {
		gameName = append(gameName, game.Name)
	}
	return api2.InitPack{Ice: servers, Games: gameName}
}
