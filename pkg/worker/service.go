package worker

import (
	"context"
	"os"
	"time"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/manager/remotehttp"
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

func (h *Service) Prepare() {
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
