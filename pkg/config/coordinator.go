package config

import "flag"

type CoordinatorConfig struct {
	Coordinator Coordinator
	Emulator    Emulator
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

type Library struct {
	// some directory which is going to be
	// the root folder for the library
	BasePath string
	// a list of supported file extensions
	Supported []string
	// a list of ignored words in the files
	Ignored []string
	// print some additional info
	Verbose bool
	// enable directory changes watch
	WatchMode bool
}

func (l Library) GetSupportedExtensions() []string { return l.Supported }

// Analytics is optional Google Analytics
type Analytics struct {
	Inject bool
	Gtag   string
}

const SelectByPing = "ping"

// allows custom config path
var coordinatorConfigPath string

func NewCoordinatorConfig() (conf CoordinatorConfig) {
	err := LoadConfig(&conf, coordinatorConfigPath)
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
