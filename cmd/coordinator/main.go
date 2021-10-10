package main

import (
	"context"
	goflag "flag"
	"math/rand"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/logger"
	"github.com/giongto35/cloud-game/v2/pkg/os"
	"github.com/giongto35/cloud-game/v2/pkg/util/logging"
	flag "github.com/spf13/pflag"
)

var Version = ""

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	conf := config.NewConfig()
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	conf.ParseFlags()

	log := logger.NewConsole(conf.Coordinator.Debug, "c")
	logging.Init()
	defer logging.Flush()

	log.Info().Msgf("version: %s", Version)
	log.Debug().Msgf("config: %+v", conf)
	c := coordinator.New(conf, log)
	c.Start()

	ctx, cancel := context.WithCancel(context.Background())
	defer c.Shutdown(ctx)
	<-os.ExpectTermination()
	cancel()
}
