package main

import (
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/coordinator"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/os"
)

var Version = "?"

func main() {
	conf, paths := config.NewCoordinatorConfig()
	conf.ParseFlags()

	log := logger.NewConsole(conf.Coordinator.Debug, "c", false)
	log.Info().Msgf("version %s", Version)
	log.Info().Msgf("conf: v%v, loaded: %v", conf.Version, paths)
	if log.GetLevel() < logger.InfoLevel {
		printable := conf
		if printable.Coordinator.Auth.AdminPassword != "" {
			printable.Coordinator.Auth.AdminPassword = "***"
		}
		log.Debug().Msgf("conf: %+v", printable)
	}
	c, err := coordinator.New(conf, log)
	if err != nil {
		log.Error().Err(err).Msgf("init fail")
		return
	}
	c.Start()
	<-os.ExpectTermination()
	if err := c.Stop(); err != nil {
		log.Error().Err(err).Msg("shutdown fail")
	}
}
