package main

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/thread"
	"github.com/giongto35/cloud-game/v2/pkg/util/logging"
	"github.com/giongto35/cloud-game/v2/pkg/worker"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

func run() {
	rand.Seed(time.Now().UTC().UnixNano())

	cfg := config.NewDefaultConfig()
	cfg.AddFlags(pflag.CommandLine)

	logging.Init()
	defer logging.Flush()

	ctx, cancelCtx := context.WithCancel(context.Background())

	glog.Infof("Initializing worker server")
	glog.V(4).Infof("Worker configs %v", cfg)
	o := worker.New(ctx, cfg)
	if err := o.Run(); err != nil {
		glog.Errorf("Failed to run worker, reason %v", err)
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
		glog.Infoln("Received SIGTERM, Quiting Worker")
		o.Shutdown()
		cancelCtx()
	}
}

func main() {
	thread.MainWrapMaybe(run)
}
