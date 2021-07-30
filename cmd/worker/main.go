package main

import (
	"context"
	goflag "flag"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/thread"
	"github.com/giongto35/cloud-game/v2/pkg/util/logging"
	"github.com/giongto35/cloud-game/v2/pkg/worker"
	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
)

var Version = ""

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

	glog.Infof("[worker] version: %v", Version)
	glog.V(4).Info("[worker] Initialization")
	glog.V(4).Infof("[worker] Local configuration %+v", conf)
	app := worker.New(ctx, conf)
	app.Run()

	signals := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)

	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-signals
		glog.V(4).Infof("[worker] Shutting down [os:%v]", sig)
		done <- struct{}{}
	}()

	<-done
	app.Shutdown()
	cancelCtx()
}

func main() {
	thread.MainWrapMaybe(run)
}
