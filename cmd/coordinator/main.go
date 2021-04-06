package main

import (
	"context"
	goflag "flag"
	"math/rand"
	"os"
	"os/signal"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/util/logging"
	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	conf := config.NewConfig()
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	conf.ParseFlags()

	logging.Init()
	defer logging.Flush()

	ctx, cancelCtx := context.WithCancel(context.Background())

	glog.Infof("Initializing coordinator server")
	glog.V(4).Infof("Coordinator configs %v", conf)
	o := coordinator.New(ctx, conf)
	if err := o.Run(); err != nil {
		glog.Errorf("Failed to run coordinator server, reason %v", err)
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
		glog.Infoln("Received SIGTERM, Quiting Coordinator")
		o.Shutdown()
		cancelCtx()
	}
}
