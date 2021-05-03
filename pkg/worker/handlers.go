package worker

import (
	"context"
	"errors"
	"log"
	"os"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/manager/remotehttp"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	storage "github.com/giongto35/cloud-game/v2/pkg/worker/cloud-storage"
	"github.com/giongto35/cloud-game/v2/pkg/worker/room"
)

type Handler struct {
	cfg           worker.Config
	cord          Coordinator
	onlineStorage *storage.Client
	rooms         Rooms
	sessions      Sessions
	w             *Worker
}

// NewHandler returns a new server
func NewHandler(cfg worker.Config, wrk *Worker) *Handler {
	if err := cfg.Emulator.CreateOfflineStorage(); err != nil {
		log.Printf("error: couldn't create offline storage at %v", cfg.Emulator.Storage)
	}
	return &Handler{
		cfg:           cfg,
		onlineStorage: storage.NewInitClient(),
		rooms:         NewRooms(),
		sessions:      NewSessions(),
		w:             wrk,
	}
}

func (h *Handler) Run(ctx context.Context) {
	conf := h.cfg.Worker.Network

	h.syncCores()

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
		conn.Printf("Connected at %v", conf.CoordinatorAddress)
		h.cord = conn
		h.cord.HandleRequests(h)
		h.cord.Listen()

		h.cord.Close()
		h.rooms.CloseAll()
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		}
	}
}

func (h *Handler) syncCores() {
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
