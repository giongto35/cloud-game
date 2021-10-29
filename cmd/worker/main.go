package main

import (
	"context"
	goflag "flag"

	config "github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/os"
	"github.com/giongto35/cloud-game/v2/pkg/thread"
	"github.com/giongto35/cloud-game/v2/pkg/worker"
	flag "github.com/spf13/pflag"
)

var Version = "?"

func run() {
	conf := config.NewConfig()
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	conf.ParseFlags()

	log := logger.NewConsole(conf.Worker.Debug, "w", false)
	log.Info().Msgf("version %s", Version)
	if log.GetLevel() < logger.InfoLevel {
		log.Debug().Msgf("config: %+v", conf)
	}

	wrk := worker.New(conf, log)
	wrk.Start()

	ctx, cancelCtx := context.WithCancel(context.Background())
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
