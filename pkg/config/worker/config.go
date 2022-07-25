package worker

import (
	"log"
	"net"
	"net/url"
	"strings"

	"github.com/giongto35/cloud-game/v2/pkg/config"
	"github.com/giongto35/cloud-game/v2/pkg/config/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/config/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/config/monitoring"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	"github.com/giongto35/cloud-game/v2/pkg/config/storage"
	webrtcConfig "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/os"
	flag "github.com/spf13/pflag"
)

type Config struct {
	Encoder   encoder.Encoder
	Emulator  emulator.Emulator
	Recording shared.Recording
	Storage   storage.Storage
	Worker    Worker
	Webrtc    webrtcConfig.Webrtc
}

type Worker struct {
	Monitoring monitoring.Config
	Network    struct {
		CoordinatorAddress string
		Endpoint           string
		PingEndpoint       string
		PublicAddress      string
		Secure             bool
		Zone               string
	}
	AutoSave struct {
		Enabled bool
		Period  int
	}
	Server shared.Server
	Tag    string
}

// allows custom config path
var configPath string

func NewConfig() (conf Config) {
	_ = config.LoadConfig(&conf, configPath)
	conf.expandSpecialTags()
	conf.fixValues()
	return
}

// ParseFlags updates config values from passed runtime flags.
// Define own flags with default value set to the current config param.
// Don't forget to call flag.Parse().
func (c *Config) ParseFlags() {
	c.Worker.Server.WithFlags()
	flag.IntVar(&c.Worker.Monitoring.Port, "monitoring.port", c.Worker.Monitoring.Port, "Monitoring server port")
	flag.StringVar(&c.Worker.Network.CoordinatorAddress, "coordinatorhost", c.Worker.Network.CoordinatorAddress, "Worker URL to connect")
	flag.StringVar(&c.Worker.Network.Zone, "zone", c.Worker.Network.Zone, "Worker network zone (us, eu, etc.)")
	flag.StringVarP(&configPath, "conf", "c", configPath, "Set custom configuration file path")
	flag.Parse()
}

// expandSpecialTags replaces all the special tags in the config.
func (c *Config) expandSpecialTags() {
	tag := "{user}"
	for _, dir := range []*string{&c.Emulator.Storage, &c.Emulator.Libretro.Cores.Repo.ExtLock} {
		if *dir == "" || !strings.Contains(*dir, tag) {
			continue
		}
		userHomeDir, err := os.GetUserHome()
		if err != nil {
			log.Fatalln("couldn't read user home directory", err)
		}
		*dir = strings.Replace(*dir, tag, userHomeDir, -1)
	}
}

// fixValues tries to fix some values otherwise hard to set externally.
func (c *Config) fixValues() {
	// with ICE lite we clear ICE servers
	if c.Webrtc.IceLite {
		c.Webrtc.IceServers = []webrtcConfig.IceServer{}
	}
}

// GetAddr returns defined in the config server address.
func (w *Worker) GetAddr() string { return w.Server.GetAddr() }

// GetPingAddr returns exposed to clients server ping endpoint address.
func (w *Worker) GetPingAddr(address string) url.URL {
	_, srcPort, _ := net.SplitHostPort(w.GetAddr())
	dstHost, _, _ := net.SplitHostPort(address)
	address = net.JoinHostPort(dstHost, srcPort)

	if w.Network.PublicAddress != "" {
		address = w.Network.PublicAddress
		if w.Network.Zone != "" {
			address = w.Network.Zone + "." + address
		}
		port := srcPort
		if port != "" && port != "80" && port != "443" && port != "0" {
			address += ":" + port
		}
	}

	pingURL := url.URL{Scheme: "http", Host: address, Path: w.Network.PingEndpoint}
	if w.Server.Https {
		pingURL.Scheme = "https"
	}
	return pingURL
}

func (w *Worker) GetPort(address string) string {
	_, port, _ := net.SplitHostPort(address)
	return port
}
