package main

import (
	"context"
	"math/rand"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/os"
	"github.com/giongto35/cloud-game/v2/pkg/worker"
	"github.com/giongto35/cloud-game/v2/pkg/worker/thread"
)

var Version = "?"

func run() {
	rand.Seed(time.Now().UTC().UnixNano())
	conf := config.NewConfig()
	conf.ParseFlags()

	log := logger.NewConsole(conf.Worker.Debug, "w", true)
	log.Info().Msgf("version %s", Version)
	if log.GetLevel() < logger.InfoLevel {
		log.Debug().Msgf("config: %+v", conf)
	}

	ctx, cancelCtx := context.WithCancel(context.Background())
	wrk := worker.New(ctx, conf, log)
	wrk.Start()
	defer func() {
		if err := wrk.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("service shutdown errors")
		}
	}()
	<-os.ExpectTermination()
	cancelCtx()
}

func main() {
	thread.Wrap(run)
}
