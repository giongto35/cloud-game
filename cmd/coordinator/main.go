package main

import (
	"context"
	goflag "flag"

	config "github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/os"
	flag "github.com/spf13/pflag"
)

var Version = "?"

func main() {
	conf := config.NewConfig()
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	conf.ParseFlags()

	log := logger.NewConsole(conf.Coordinator.Debug, "c", false)

	log.Info().Msgf("version %s", Version)
	if log.GetLevel() < logger.InfoLevel {
		log.Debug().Msgf("config: %+v", conf)
	}
	c := coordinator.New(conf, log)
	c.Start()

	ctx, cancel := context.WithCancel(context.Background())
	defer func() {
		if err := c.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("service shutdown errors")
		}
	}()
	<-os.ExpectTermination()
	cancel()
}
