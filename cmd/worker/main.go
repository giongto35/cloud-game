package main

import (
	"context"
	goflag "flag"
	"math/rand"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/os"
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

	glog.Infof("[worker] version: %v", Version)
	glog.V(4).Infof("[worker] Local configuration %+v", conf)
	wrk := worker.New(conf)
	wrk.Start()

	ctx, cancelCtx := context.WithCancel(context.Background())
	defer wrk.Shutdown(ctx)
	<-os.ExpectTermination()
	cancelCtx()
}

func main() {
	thread.MainWrapMaybe(run)
}
