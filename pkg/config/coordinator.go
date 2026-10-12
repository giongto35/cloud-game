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
	Analytics   Analytics
	Auth        Auth
	Covers      Covers
	Tournaments Tournaments
	Debug       bool
	Library     Library
	MaxWsSize   int64
	Monitoring  Monitoring
	Origin      struct {
		UserWs   string
		WorkerWs string
	}
	Selector string
	Server   Server
}

// Auth configures accounts, sessions and access control.
type Auth struct {
	// DbPath is the SQLite database file path.
	DbPath string
	// AdminEmail is the primary (protected) administrator account.
	AdminEmail string
	// AdminPassword is the initial password of the primary admin (first start only).
	// When empty, a random one is generated and printed to the log once.
	AdminPassword string
	// Moderators enables the moderator role (delegated user approval).
	Moderators bool
	// SessionDays is the session lifetime in days.
	SessionDays int
	// SecureCookie forces the Secure cookie flag (set it when behind an HTTPS proxy).
	SecureCookie bool
	// QueueReserveSec is how long a freed server waits for the called user.
	QueueReserveSec int
}

// Covers configures the game cover art cache.
type Covers struct {
	Enabled bool
	// Dir is where downloaded images are stored.
	Dir string
	// Url is the libretro-thumbnails server.
	Url string
	// RetryDays is how long a "not found" result is kept before trying again.
	RetryDays int
}

// Tournaments configures the tournament module.
type Tournaments struct {
	Enabled bool
	// MaxConcurrent is the number of tournament matches played at the same time.
	MaxConcurrent int
	// JoinTimeoutSec is how long the players have to join a called match.
	JoinTimeoutSec int
	// ReplayDir stores the match replays.
	ReplayDir string
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
