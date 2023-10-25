package libretro

import (
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/games"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/manager"
	"github.com/giongto35/cloud-game/v3/pkg/worker/cloud"
)

type Caged struct {
	Emulator

	base *Frontend // maintains the root for mad embedding
	conf CagedConf
	log  *logger.Logger
	w, h int

	OnSysInfoChange func()
}

type CagedConf struct {
	Emulator  config.Emulator
	Recording config.Recording
}

func (c *Caged) Name() string { return "libretro" }

func Cage(conf CagedConf, log *logger.Logger) Caged {
	return Caged{conf: conf, log: log}
}

func (c *Caged) Init() error {
	if err := manager.CheckCores(c.conf.Emulator, c.log); err != nil {
		c.log.Warn().Err(err).Msgf("a Libretro cores sync fail")
	}
	return nil
}

func (c *Caged) ReloadFrontend() {
	frontend, err := NewFrontend(c.conf.Emulator, c.log)
	if err != nil {
		c.log.Fatal().Err(err).Send()
	}
	c.Emulator = frontend
	c.base = frontend
}

// VideoChangeCb adds a callback when video params are changed by the app.
func (c *Caged) VideoChangeCb(fn func()) { c.base.SetVideoChangeCb(fn) }

func (c *Caged) Load(game games.GameMetadata, path string) error {
	c.Emulator.LoadCore(game.System)
	if err := c.Emulator.LoadGame(game.FullPath(path)); err != nil {
		return err
	}
	c.ViewportRecalculate()
	return nil
}

func (c *Caged) EnableRecording(nowait bool, user string, game string) {
	if c.conf.Recording.Enabled {
		// !to fix races with canvas pool when recording
		c.base.DisableCanvasPool = true
		c.Emulator = WithRecording(c.Emulator, nowait, user, game, c.conf.Recording, c.log)
	}
}

func (c *Caged) EnableCloudStorage(uid string, storage cloud.Storage) {
	if storage != nil {
		wc, err := WithCloud(c.Emulator, uid, storage)
		if err != nil {
			c.log.Error().Err(err).Msgf("couldn't init %v", wc.HashPath())
		} else {
			c.log.Info().Msgf("cloud state %v has been initialized", wc.HashPath())
			c.Emulator = wc
		}
	}
}

func (c *Caged) PixFormat() uint32                 { return c.Emulator.PixFormat() }
func (c *Caged) Rotation() uint                    { return c.Emulator.Rotation() }
func (c *Caged) AudioSampleRate() int              { return c.Emulator.AudioSampleRate() }
func (c *Caged) ViewportSize() (int, int)          { return c.base.ViewportSize() }
func (c *Caged) Scale() float64                    { return c.Emulator.Scale() }
func (c *Caged) SendControl(port int, data []byte) { c.base.Input(port, data) }
func (c *Caged) Start()                            { go c.Emulator.Start() }
func (c *Caged) SetSaveOnClose(v bool)             { c.base.SaveOnClose = v }
func (c *Caged) SetSessionId(name string)          { c.base.SetSessionId(name) }
func (c *Caged) Close()                            { c.Emulator.Close() }
