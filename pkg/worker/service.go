package worker

import (
	"context"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/service"
	"github.com/giongto35/cloud-game/v2/pkg/storage"
)

type Service struct {
	service.RunnableService

	address string
	conf    worker.Config
	cord    coordinator
	router  Router
	storage storage.CloudStorage
	log     *logger.Logger
}

const retry = 10 * time.Second

func NewHandler(address string, conf worker.Config, log *logger.Logger) *Service {
	return &Service{
		address: address,
		conf:    conf,
		log:     log,
		storage: storage.GetCloudStorage(conf.Storage.Provider, conf.Storage.Key),
		router:  NewRouter(),
	}
}

func (h *Service) Run() {
	remoteAddr := h.conf.Worker.Network.CoordinatorAddress
	for {
		conn, err := connect(remoteAddr, h.conf.Worker, h.address, h.log)
		if err != nil {
			h.log.Error().Err(err).
				Msgf("no connection to the coordinator %v. Retrying in %v", remoteAddr, retry)
			time.Sleep(retry)
			continue
		}
		h.cord = *conn
		h.cord.Log.Info().Msgf("Connected to the coordinator %v", remoteAddr)
		h.cord.HandleRequests(h)
		h.cord.Listen()
		// block
		h.cord.Close()
		h.router.Close()
	}
}

func (h *Service) Shutdown(context.Context) error { return nil }

// removeUser removes the user from the room.
func (h *Service) removeUser(user *Session) {
	room := user.GetRoom()
	if room == nil || room.IsEmpty() {
		return
	}
	room.RemoveUser(user)
	h.log.Info().Msg("Closing peer connection")
	if room.IsEmpty() {
		h.log.Info().Msg("Closing an empty room")
		room.Close()
	}
}
