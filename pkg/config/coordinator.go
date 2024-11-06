package config

import "flag"

type CoordinatorConfig struct {
	Coordinator Coordinator
	Emulator    Emulator
	Library     Library
	Recording   Recording
	Version     Version
	Webrtc      Webrtc
}

type Coordinator struct {
	Analytics  Analytics
	Debug      bool
	Library    Library
	Monitoring Monitoring
	Origin     struct {
		UserWs   string
		WorkerWs string
	}
	Selector string
	Server   Server
}

// Analytics is optional Google Analytics
type Analytics struct {
	Inject bool
	Gtag   string
}

const SelectByPing = "ping"

// allows custom config path
var coordinatorConfigPath string

func NewCoordinatorConfig() (conf CoordinatorConfig, paths []string) {
	paths, err := LoadConfig(&conf, coordinatorConfigPath)
	if err != nil {
		panic(err)
	}
	return
}

func (c *CoordinatorConfig) ParseFlags() {
	c.Coordinator.Server.WithFlags()
	flag.IntVar(&c.Coordinator.Monitoring.Port, "monitoring.port", c.Coordinator.Monitoring.Port, "Monitoring server port")
	flag.StringVar(&coordinatorConfigPath, "c-conf", coordinatorConfigPath, "Set custom configuration file path")
	flag.Parse()
}
