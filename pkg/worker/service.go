package worker

import (
	"context"
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
	ctx     context.Context
	log     *logger.Logger
	router  Router
	storage storage.CloudStorage
}

const retry = 10 * time.Second

func NewWorkerService(ctx context.Context, address string, conf worker.Config, log *logger.Logger) *Service {
	return &Service{
		address: address,
		conf:    conf,
		ctx:     ctx,
		log:     log,
		storage: storage.GetCloudStorage(conf.Storage.Provider, conf.Storage.Key),
		router:  NewRouter(),
	}
}

func (s *Service) Run() {
	if err := remotehttp.CheckCores(s.conf.Emulator, s.log); err != nil {
		s.log.Error().Err(err).Msg("cores sync error")
	}
	remoteAddr := s.conf.Worker.Network.CoordinatorAddress
	go func() {
		for {
			conn, err := connect(remoteAddr, s.conf.Worker, s.address, s.log)
			if err != nil {
				s.log.Error().Err(err).
					Msgf("no connection to the coordinator %v. Retrying in %v", remoteAddr, retry)
				time.Sleep(retry)
				continue
			}
			s.cord = *conn
			s.cord.Log.Info().Msgf("Connected to the coordinator %v", remoteAddr)
			s.cord.HandleRequests(s)
			select {
			case <-s.ctx.Done():
				s.cord.Close()
				s.router.Close()
				return
			case <-s.cord.Done():
				s.cord.Close()
				s.router.Close()
			}
		}
	}()
}

func (s *Service) Shutdown(context.Context) error { return nil }
