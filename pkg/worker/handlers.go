package worker

import (
	"context"
	"errors"
	"log"
	"net/http"
	"net/url"
	"os"
	"sync"
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
	cord Coordinator
	// Raw address of coordinator
	coordinatorHost string
	cfg             worker.Config
	// rooms map : RoomID -> Room
	rooms map[string]*room.Room
	// global ID of the current server
	serverID string
	// onlineStorage is client accessing to online storage (GCP)
	onlineStorage *storage.Client
	// sessions handles all sessions server is handler (key is sessionID)
	sessions map[network.Uid]*Session

	mu sync.Mutex

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
func (h *Handler) Run(ctx context.Context) {
	conf := h.cfg.Worker.Network

	h.Prepare()

	for {
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}

		conn, err := newCoordinatorConnection(conf.CoordinatorAddress, conf.Zone, h.cfg)
		if err != nil {
			log.Printf("Cannot connect to coordinator. %v Retrying...", err)
			time.Sleep(time.Second)
			continue
		}
		log.Printf("[worker] connected to: %v", conf.CoordinatorAddress)

		h.cord = conn
		h.cord.HandleRequests(h)

		h.cord.Listen()
		h.Close()
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}
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

func newCoordinatorConnection(host string, zone string, conf worker.Config) (Coordinator, error) {
	scheme := "ws"
	if conf.Worker.Network.Secure {
		scheme = "wss"
	}
	address := url.URL{Scheme: scheme, Host: host, Path: conf.Worker.Network.Endpoint, RawQuery: "zone=" + zone}
	log.Printf("[worker] connect to %v", address.String())

	conn, err := ipc.NewClient(address)
	if err != nil {
		return Coordinator{}, err
	}
	return NewCoordinator(conn), nil
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

func (h *Handler) getSession(id network.Uid) *Session {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.sessions[id]
}

func (h *Handler) addSession(id network.Uid, value *Session) {
	h.mu.Lock()
	h.sessions[id] = value
	h.mu.Unlock()
}

func (h *Handler) removeSession(id network.Uid) {
	h.mu.Lock()
	delete(h.sessions, id)
	h.mu.Unlock()
}

func (h *Handler) getRoom(id string) *room.Room {
	//h.mu.Lock()
	//defer h.mu.Unlock()
	return h.rooms[id]
}

// detachRoom detach room from Handler
func (h *Handler) detachRoom(id string) {
	//h.mu.Lock()
	delete(h.rooms, id)
	//h.mu.Unlock()
}

// createRoom creates a new room or returns nil for existing.
func (h *Handler) createRoom(id string, game games.GameMetadata) *room.Room {
	// If the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if !h.isRoomBusy(id) {
		newRoom := room.NewRoom(id, game, h.onlineStorage, h.cfg)
		//h.mu.Lock()
		h.rooms[newRoom.ID] = newRoom
		//h.mu.Unlock()
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
	h.mu.Lock()
	r, ok := h.rooms[roomID]
	h.mu.Unlock()
	if !ok {
		return false
	}
	return r.IsRunningSessions()
}

func (h *Handler) Close() {
	h.cord.Close()
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
