package main

import (
	"context"
	goflag "flag"
	"math/rand"
	"os"
	"os/signal"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/thread"
	"github.com/giongto35/cloud-game/v2/pkg/util/logging"
	"github.com/giongto35/cloud-game/v2/pkg/worker"
	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func run() {
	conf := config.NewConfig()
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	conf.ParseFlags()

	logging.Init()
	defer logging.Flush()

	ctx, cancelCtx := context.WithCancel(context.Background())

	glog.V(4).Info("[worker] Initialization")
	glog.V(4).Infof("[worker] Local configuration %+v", conf)
	wrk := worker.New(ctx, conf)
	wrk.Run()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)
	select {
	case <-stop:
		glog.V(4).Info("[worker] Shutting down")
		wrk.Shutdown()
		cancelCtx()
	}
}

func main() {
	thread.MainWrapMaybe(run)
}
