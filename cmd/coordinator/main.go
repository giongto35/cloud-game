package main

import (
	"context"
	goflag "flag"
	"math/rand"
	"time"

	config "github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/os"
	"github.com/giongto35/cloud-game/v2/pkg/util/logging"
	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
)

func init() {
	rand.Seed(time.Now().UTC().UnixNano())
}

func main() {
	conf := config.NewConfig()
	flag.CommandLine.AddGoFlagSet(goflag.CommandLine)
	conf.ParseFlags()

	logging.Init()
	defer logging.Flush()

	ctx, cancelCtx := context.WithCancel(context.Background())

	glog.V(4).Infof("Coordinator configs %v", conf)
	c := coordinator.New(conf)
	c.Start()
	defer c.Shutdown(ctx)

	<-os.ExpectTermination()
	cancelCtx()
}
