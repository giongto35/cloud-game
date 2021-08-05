package worker

import (
	"log"
	"net/url"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/config/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/config/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	webrtcConfig "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/environment"
	flag "github.com/spf13/pflag"
)

type Config struct {
	Encoder     encoder.Encoder
	Emulator    emulator.Emulator
	Environment shared.Environment
	Worker      Worker
	Webrtc      webrtcConfig.Webrtc
}

type Worker struct {
	Monitoring monitoring.Config
	Network    struct {
		CoordinatorAddress string
		Endpoint           string
		PingEndpoint       string
		Secure             bool
		Zone               string
	}
	Server shared.Server
}

// allows custom config path
var configPath string

func NewConfig() (conf Config) {
	_ = config.LoadConfig(&conf, configPath)
	conf.expandSpecialTags()
	return
}

// ParseFlags updates config values from passed runtime flags.
// Define own flags with default value set to the current config param.
// Don't forget to call flag.Parse().
func (c *Config) ParseFlags() {
	c.Environment.WithFlags()
	c.Worker.Server.WithFlags()
	flag.IntVar(&c.Worker.Monitoring.Port, "monitoring.port", c.Worker.Monitoring.Port, "Monitoring server port")
	flag.StringVar(&c.Worker.Network.CoordinatorAddress, "coordinatorhost", c.Worker.Network.CoordinatorAddress, "Worker URL to connect")
	flag.StringVar(&c.Worker.Network.Zone, "zone", c.Worker.Network.Zone, "Worker network zone (us, eu, etc.)")
	flag.StringVarP(&configPath, "conf", "c", configPath, "Set custom configuration file path")
	flag.Parse()
}

// expandSpecialTags replaces all the special tags in the config.
func (c *Config) expandSpecialTags() {
	// home dir
	dir := c.Emulator.Storage
	if dir != "" {
		tag := "{user}"
		if strings.Contains(dir, tag) {
			userHomeDir, err := environment.GetUserHome()
			if err != nil {
				log.Fatalln("couldn't read user home directory", err)
			}
			c.Emulator.Storage = strings.Replace(dir, tag, userHomeDir, -1)
		}
	}
}

// GetAddr returns defined in the config server address.
func (w *Worker) GetAddr() string { return w.Server.GetAddr() }

// GetPingAddr returns exposed to clients server ping endpoint address.
func (w *Worker) GetPingAddr(address string) string {
	pingURL := url.URL{Scheme: "http", Host: address, Path: w.Network.PingEndpoint}
	if w.Server.Https {
		pingURL.Scheme = "https"
	}
	return pingURL.String()
}
