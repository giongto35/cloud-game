package main

import (
	"math/rand"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/os"
)

var Version = "?"

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	conf := config.NewConfig()
	conf.ParseFlags()

	log := logger.NewConsole(conf.Coordinator.Debug, "c", false)

	log.Info().Msgf("version %s", Version)
	log.Info().Msgf("conf version: %v", conf.Version)
	if log.GetLevel() < logger.InfoLevel {
		log.Debug().Msgf("config: %+v", conf)
	}
	c := coordinator.New(conf, log)
	c.Start()
	<-os.ExpectTermination()
	if err := c.Stop(); err != nil {
		log.Error().Err(err).Msg("service shutdown errors")
	}
}
