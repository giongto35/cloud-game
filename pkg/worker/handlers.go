package worker

import (
	"context"
	"os"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/manager/remotehttp"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/service"
	"github.com/giongto35/cloud-game/v2/pkg/storage"
)

type Handler struct {
	service.RunnableService

	address string
	conf    worker.Config
	cord    Coordinator
	log     *logger.Logger
	storage storage.CloudStorage
	router  Router
}

func NewHandler(address string, conf worker.Config, log *logger.Logger) *Handler {
	createLocalStorage(conf.Emulator.Storage, log)
	return &Handler{
		address: address,
		conf:    conf,
		log:     log,
		storage: initCloudStorage(conf),
		router:  NewRouter(),
	}
}

func (h *Handler) Run() {
	var err error
	coordinatorAddress := h.conf.Worker.Network.CoordinatorAddress
	for {
		if h.cord, err = newCoordinatorConnection(coordinatorAddress, h.conf.Worker, h.address, h.log); err != nil {
			h.log.Error().Err(err).Msg("Cannot connect to coordinator. %v Retrying...")
			time.Sleep(time.Second)
			continue
		}
		h.cord.GetLogger().Info().Msgf("Connected at %v", coordinatorAddress)
		h.cord.HandleRequests(h)
		h.cord.Listen()

		h.cord.Close()
		h.router.Close()
	}
}

func (h *Handler) Shutdown(context.Context) error { return nil }

func (h *Handler) Prepare() {
	if !h.conf.Emulator.Libretro.Cores.Repo.Sync {
		return
	}

	h.log.Info().Msg("Starting Libretro cores sync...")
	coreManager := remotehttp.NewRemoteHttpManager(h.conf.Emulator.Libretro, h.log)
	// make a dir for cores
	dir := coreManager.Conf.GetCoresStorePath()
	if err := os.MkdirAll(dir, os.ModeDir); err != nil {
		h.log.Error().Err(err).Msgf("couldn't make directory: %v", dir)
		return
	}
	if err := coreManager.Sync(); err != nil {
		h.log.Error().Err(err).Msg("cores sync has failed")
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
		st, _ = storage.NewNoopCloudStorage()
	}
	return st
}

// removeUser removes the user from the room.
func (h *Handler) removeUser(user *Session) {
	room := user.GetRoom()
	if room == nil || room.IsEmpty() {
		return
	}
	room.RemoveUser(user)
	h.log.Info().Msg("Closing peer connection")
	if room.IsEmpty() {
		h.log.Info().Msg("Closing an empty room")
		room.Close()
		user.GetPeerConn().SendMessage([]byte{0xFF, 0xFF})
	}
}

// CreateRoom creates a new room or returns nil for existing.
func (h *Handler) CreateRoom(id string, game games.GameMetadata, onClose func(*Room)) *Room {
	// If the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	old := h.router.GetRoom(id)
	exists := old != nil && old.HasRunningSessions()
	if exists {
		return nil
	}
	return NewRoom(id, game, h.storage, onClose, h.conf, h.log)
}

func createLocalStorage(path string, log *logger.Logger) {
	log.Info().Msgf("Local storage path: %v", path)
	if err := os.MkdirAll(path, 0755); err != nil {
		log.Error().Err(err).Msgf("failed to create local storage path: %v", path)
	}
}

func (h *Handler) TerminateSession(session *Session) {
	session.Close()
	h.router.RemoveUser(session)
	h.removeUser(session)
}
