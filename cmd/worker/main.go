package main

import (
	"math/rand"
	"time"

	config "github.com/giongto35/cloud-game/v3/pkg/config/worker"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/os"
	"github.com/giongto35/cloud-game/v3/pkg/worker"
	"github.com/giongto35/cloud-game/v3/pkg/worker/thread"
)

var Version = "?"

func run() {
	rand.New(rand.NewSource(time.Now().UnixNano())) // !to remove when bumped to 1.20
	conf := config.NewConfig()
	conf.ParseFlags()

	log := logger.NewConsole(conf.Worker.Debug, "w", false)
	log.Info().Msgf("version %s", Version)
	log.Info().Msgf("conf version: %v", conf.Version)
	if log.GetLevel() < logger.InfoLevel {
		log.Debug().Msgf("config: %+v", conf)
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
