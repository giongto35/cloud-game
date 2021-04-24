package worker

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/manager/remotehttp"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	storage "github.com/giongto35/cloud-game/v2/pkg/worker/cloud-storage"
	"github.com/giongto35/cloud-game/v2/pkg/worker/room"
)

type Handler struct {
	// Client that connects to coordinator
	oClient *CoordinatorClient
	// Raw address of coordinator
	coordinatorHost string
	cfg             worker.Config
	// Rooms map : RoomID -> Room
	rooms map[string]*room.Room
	// global ID of the current server
	serverID string
	// onlineStorage is client accessing to online storage (GCP)
	onlineStorage *storage.Client
	// sessions handles all sessions server is handler (key is sessionID)
	sessions map[network.Uid]*Session

	w *Worker
}

// NewHandler returns a new server
func NewHandler(cfg worker.Config, wrk *Worker) *Handler {
	// Create offline storage folder
	createOfflineStorage(cfg.Emulator.Storage)

	// Init online storage
	onlineStorage := storage.NewInitClient()
	return &Handler{
		rooms:           map[string]*room.Room{},
		sessions:        map[network.Uid]*Session{},
		coordinatorHost: cfg.Worker.Network.CoordinatorAddress,
		cfg:             cfg,
		onlineStorage:   onlineStorage,
		w:               wrk,
	}
}

// Run starts a Handler running logic
func (h *Handler) Run() {
	conf := h.cfg.Worker.Network
	for {
		conn, err := newCoordinatorConnection(conf.CoordinatorAddress, conf.Zone, h.cfg)
		if err != nil {
			log.Printf("Cannot connect to coordinator. %v Retrying...", err)
			time.Sleep(time.Second)
			continue
		}
		log.Printf("[worker] connected to: %v", conf.CoordinatorAddress)

		h.oClient = conn
		go h.oClient.Heartbeat()
		h.routes()
		h.oClient.Listen()
		// If cannot listen, reconnect to coordinator
	}
}

func (h *Handler) Prepare() {
	if !h.cfg.Emulator.Libretro.Cores.Repo.Sync {
		return
	}

	log.Printf("Starting Libretro cores sync...")
	coreManager := remotehttp.NewRemoteHttpManager(h.cfg.Emulator.Libretro)
	// make a dir for cores
	dir := coreManager.Conf.GetCoresStorePath()
	if err := os.MkdirAll(dir, os.ModeDir); err != nil {
		log.Printf("error: couldn't make %v directory", dir)
		return
	}
	if err := coreManager.Sync(); err != nil {
		log.Printf("error: cores sync has failed, %v", err)
	}
}

func newCoordinatorConnection(host string, zone string, conf worker.Config) (*CoordinatorClient, error) {
	scheme := "ws"
	if conf.Worker.Network.Secure {
		scheme = "wss"
	}
	address := url.URL{Scheme: scheme, Host: host, Path: conf.Worker.Network.Endpoint, RawQuery: "zone=" + zone}
	log.Printf("[worker] connect to %v", address.String())

	conn, err := ipc.Connect(address)
	if err != nil {
		return nil, err
	}
	return NewCoordinatorClient(conn), nil
}

func (h *Handler) GetCoordinatorClient() *CoordinatorClient {
	return h.oClient
}

// detachPeerConn detaches a peerconnection from the current room.
func (h *Handler) detachPeerConn(pc *webrtc.WebRTC) {
	log.Printf("[worker] closing peer connection")
	gameRoom := h.getRoom(pc.RoomID)
	if gameRoom == nil || gameRoom.IsEmpty() {
		return
	}
	gameRoom.RemoveSession(pc)
	if gameRoom.IsEmpty() {
		log.Printf("[worker] closing an empty room")
		gameRoom.Close()
		pc.InputChannel <- []byte{0xFF, 0xFF}
		close(pc.InputChannel)
	}
}

func (h *Handler) getSession(id network.Uid) *Session { return h.sessions[id] }

func (h *Handler) getRoom(id string) *room.Room { return h.rooms[id] }

// detachRoom detach room from Handler
func (h *Handler) detachRoom(id string) {
	delete(h.rooms, id)
}

// createRoom creates a new room or returns nil for existing.
func (h *Handler) createRoom(id string, game games.GameMetadata) *room.Room {
	// If the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if !h.isRoomBusy(id) {
		newRoom := room.NewRoom(id, game, h.onlineStorage, h.cfg)
		// TODO: Might have race condition (and it has (:)
		h.rooms[newRoom.ID] = newRoom
		return newRoom
	}
	return nil
}

// isRoomBusy check if there is any running sessions.
// TODO: If we remove sessions from room anytime a session is closed,
// we can check if the sessions list is empty or not.
func (h *Handler) isRoomBusy(roomID string) bool {
	if roomID == "" {
		return false
	}
	// If no roomID is registered
	r, ok := h.rooms[roomID]
	if !ok {
		return false
	}
	return r.IsRunningSessions()
}

func (h *Handler) Close() {
	if h.oClient != nil {
		h.oClient.Close()
	}
	for _, r := range h.rooms {
		r.Close()
	}
}

func createOfflineStorage(path string) {
	log.Printf("Set storage: %v", path)
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Println("Failed to create offline storage, err: ", err)
	}
}

func echo(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	_, _ = w.Write([]byte{0x65, 0x63, 0x68, 0x6f}) // echo
}
