package main

import (
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/os"
	"github.com/giongto35/cloud-game/v3/pkg/worker"
	"github.com/giongto35/cloud-game/v3/pkg/worker/thread"
)

var Version = "?"

func run() {
	conf := config.NewWorkerConfig()
	conf.ParseFlags()

	log := logger.NewConsole(conf.Worker.Debug, "w", false)
	log.Info().Msgf("version %s", Version)
	log.Info().Msgf("conf: v%v", conf.Version)
	if log.GetLevel() < logger.InfoLevel {
		log.Debug().Msgf("conf: %+v", conf)
	}

	done := os.ExpectTermination()
	wrk := worker.New(conf, log, done)
	wrk.Start()
	<-done
	time.Sleep(100 * time.Millisecond)
	if err := wrk.Stop(); err != nil {
		log.Error().Err(err).Msg("service shutdown errors")
	}
}

func main() {
	thread.Wrap(run)
}
