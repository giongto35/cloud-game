package main

import (
	"context"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/giongto35/cloud-game/pkg/overlord"
	"github.com/giongto35/cloud-game/pkg/util/logging"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
)

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	cfg := overlord.NewDefaultConfig()
	cfg.AddFlags(pflag.CommandLine)

	logging.Init()
	defer logging.Flush()

	ctx, cancelCtx := context.WithCancel(context.Background())

	glog.Infof("Initializing overlord server")
	glog.V(4).Infof("Overlord configs %v", cfg)
	o := overlord.New(ctx, cfg)
	if err := o.Run(); err != nil {
		glog.Errorf("Failed to run overlord server, reason %v", err)
		os.Exit(1)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
		glog.Infoln("Received SIGTERM, Quiting Overlord")
		o.Shutdown()
		cancelCtx()
	}
}
