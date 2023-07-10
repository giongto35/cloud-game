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
	conf, paths := config.NewWorkerConfig()
	conf.ParseFlags()

	log := logger.NewConsole(conf.Worker.Debug, "w", false)
	log.Info().Msgf("version %s", Version)
	log.Info().Msgf("conf: v%v, loaded: %v", conf.Version, paths)
	if log.GetLevel() < logger.InfoLevel {
		log.Debug().Msgf("conf: %+v", conf)
	}

	done := os.ExpectTermination()
	w, err := worker.New(conf, log)
	if err != nil {
		log.Error().Err(err).Msgf("init fail")
		return
	}
	w.Start(done)
	<-done
	time.Sleep(100 * time.Millisecond) // hack
	if err := w.Stop(); err != nil {
		log.Error().Err(err).Msg("shutdown fail")
	}
}

func main() { thread.Wrap(run) }
