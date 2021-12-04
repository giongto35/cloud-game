package worker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/manager/remotehttp"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/network/websocket"
	"github.com/giongto35/cloud-game/v2/pkg/service"
	"github.com/giongto35/cloud-game/v2/pkg/storage"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/worker/room"
)

type Handler struct {
	service.RunnableService

	address string
	// Client that connects to coordinator
	oClient *CoordinatorClient
	cfg     worker.Config
	// Rooms map : RoomID -> Room
	rooms map[string]*room.Room
	// global ID of the current server
	serverID string
	// onlineStorage is client accessing to online storage (GCP)
	onlineStorage storage.CloudStorage
	// sessions handles all sessions server is handler (key is sessionID)
	sessions map[string]*Session
}

func NewHandler(conf worker.Config, address string) *Handler {
	createOfflineStorage(conf.Emulator.Storage)
	onlineStorage := initCloudStorage(conf)
	return &Handler{
		address:       address,
		cfg:           conf,
		onlineStorage: onlineStorage,
		rooms:         map[string]*room.Room{},
		sessions:      map[string]*Session{},
	}
}

// Run starts a Handler running logic
func (h *Handler) Run() {
	coordinatorAddress := h.cfg.Worker.Network.CoordinatorAddress
	for {
		conn, err := newCoordinatorConnection(coordinatorAddress, h.cfg.Worker, h.address)
		if err != nil {
			log.Printf("Cannot connect to coordinator. %v Retrying...", err)
			time.Sleep(time.Second)
			continue
		}
		log.Printf("[worker] connected to: %v", coordinatorAddress)

		h.oClient = conn
		go h.oClient.Heartbeat()
		h.routes()
		h.oClient.Listen()
		// If cannot listen, reconnect to coordinator
	}
}

func (h *Handler) Shutdown(context.Context) error { return nil }

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

func initCloudStorage(conf worker.Config) storage.CloudStorage {
	var st storage.CloudStorage
	var err error
	switch conf.Storage.Provider {
	case "google":
		st, err = storage.NewGoogleCloudClient()
	case "oracle":
		st, err = storage.NewOracleDataStorageClient(conf.Storage.Key)
	case "coordinator":
	default:
		st, _ = storage.NewNoopCloudStorage()
	}
	if err != nil {
		log.Printf("Switching to noop cloud save")
		st, _ = storage.NewNoopCloudStorage()
	}
	return st
}

func newCoordinatorConnection(host string, conf worker.Worker, addr string) (*CoordinatorClient, error) {
	scheme := "ws"
	if conf.Network.Secure {
		scheme = "wss"
	}
	address := url.URL{Scheme: scheme, Host: host, Path: conf.Network.Endpoint}

	req, err := MakeConnectionRequest(conf, addr)
	if req != "" && err == nil {
		address.RawQuery = "data=" + req
	}

	conn, err := websocket.Connect(address)
	if err != nil {
		return nil, err
	}
	return NewCoordinatorClient(conn), nil
}

func MakeConnectionRequest(conf worker.Worker, address string) (string, error) {
	req := api.ConnectionRequest{
		Zone:     conf.Network.Zone,
		PingAddr: conf.GetPingAddr(address),
		IsHTTPS:  conf.Server.Https,
	}
	rez, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(rez), nil
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

func (h *Handler) getRoom(roomID string) (r *room.Room) {
	r, ok := h.rooms[roomID]
	if !ok {
		return nil
	}
	return
}

// getRoom returns session from sessionID
func (h *Handler) getSession(sessionID string) *Session {
	session, ok := h.sessions[sessionID]
	if !ok {
		return nil
	}

	return session
}

// detachRoom detach room from Handler
func (h *Handler) detachRoom(roomID string) {
	delete(h.rooms, roomID)
}

// createNewRoom creates a new room
// Return nil in case of room is existed
func (h *Handler) createNewRoom(game games.GameMetadata, recUser string, rec bool, roomID string) *room.Room {
	// If the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if !h.isRoomBusy(roomID) {
		newRoom := room.NewRoom(roomID, game, recUser, rec, h.onlineStorage, h.cfg)
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
