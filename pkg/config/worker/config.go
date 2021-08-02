package worker

import (
	"encoding/json"
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
	Worker      struct {
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
	Webrtc webrtcConfig.Webrtc
	Loaded bool
}

// allows custom config path
var configPath string

func NewConfig() (conf Config) {
	if err := config.LoadConfig(&conf, configPath); err == nil {
		conf.Loaded = true
	}
	conf.expandSpecialTags()
	return
}

func EmptyConfig() (conf Config) {
	conf.Loaded = false
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

func (c *Config) Serialize() []byte {
	res, _ := json.Marshal(c)
	return res
}

func (c *Config) Deserialize(data []byte) {
	if err := json.Unmarshal(data, c); err == nil {
		c.Loaded = true
	}
	c.expandSpecialTags()
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

// GetPingAddr returns the server for latency check of a zone.
func (c *Config) GetPingAddr(address string) string {
	scheme := "http"
	//host := c.Worker.Server.Address
	if c.Worker.Server.Https {
		scheme = "https"
	//	host = c.Worker.Server.Tls.Address
	}
	//if strings.HasPrefix(host, ":") {
	//	host = "localhost" + host
	//}
	//if c.Worker.Network.Zone != "" {
	//	host = c.Worker.Network.Zone + "." + host
	//}
	u := url.URL{Scheme: scheme, Host: address, Path: c.Worker.Network.PingEndpoint}
	return u.String()
}
