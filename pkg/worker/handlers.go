package worker

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/manager/remotehttp"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/service"
	"github.com/giongto35/cloud-game/v2/pkg/storage"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/worker/room"
)

type Handler struct {
	service.RunnableService

	address       string
	cfg           worker.Config
	cord          Coordinator
	onlineStorage storage.CloudStorage
	rooms         Rooms
	sessions      Sessions
}

// NewHandler returns a new server
func NewHandler(conf worker.Config, address string) *Handler {
	createOfflineStorage(conf.Emulator.Storage)
	onlineStorage := initCloudStorage(conf)
	return &Handler{
		address:       address,
		cfg:           conf,
		onlineStorage: onlineStorage,
		rooms:         NewRooms(),
		sessions:      NewSessions(),
	}
}

func (h *Handler) Run() {
	coordinatorAddress := h.cfg.Worker.Network.CoordinatorAddress
	for {
		conn, err := newCoordinatorConnection(coordinatorAddress, h.cfg.Worker, h.address)
		if err != nil {
			log.Printf("Cannot connect to coordinator. %v Retrying...", err)
			time.Sleep(time.Second)
			continue
		}
		conn.Printf("Connected at %v", coordinatorAddress)
		h.cord = conn
		h.cord.HandleRequests(h)
		h.cord.Listen()

		h.cord.Close()
		h.rooms.CloseAll()
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

// detachPeerConn detaches a peerconnection from the current room.
func (h *Handler) detachPeerConn(pc *webrtc.WebRTC) {
	log.Printf("[worker] closing peer connection")
	rm := h.rooms.Get(pc.RoomID)
	if rm == nil || rm.IsEmpty() {
		return
	}
	rm.RemoveSession(pc)
	if rm.IsEmpty() {
		log.Printf("[worker] closing an empty room")
		rm.Close()
		pc.InputChannel <- []byte{0xFF, 0xFF}
		close(pc.InputChannel)
	}
}

// createRoom creates a new room or returns nil for existing.
func (h *Handler) createRoom(id string, game games.GameMetadata) *room.Room {
	// If the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if h.rooms.noSessions(id) {
		newRoom := room.NewRoom(id, game, h.onlineStorage, h.cfg)
		h.rooms.Add(newRoom)
		return newRoom
	}
	return nil
}

func createOfflineStorage(path string) {
	log.Printf("Set storage: %v", path)
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Println("Failed to create offline storage, err: ", err)
	}
}
