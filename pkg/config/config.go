package config

import (
	"flag"
	"github.com/giongto35/cloud-game/pkg/emulator/libretro/image"
	"time"
)

const DefaultSTUNTURN = `[{"urls":"stun:stun-turn.webgame2d.com:3478"},{"urls":"turn:stun-turn.webgame2d.com:3478","username":"root","credential":"root"}]`
const CODEC_VP8 = "VP8"
const CODEC_H264 = "H264"

const AUDIO_RATE = 48000
const AUDIO_CHANNELS = 2
const AUDIO_MS = 20
const AUDIO_FRAME = AUDIO_RATE * AUDIO_MS / 1000 * AUDIO_CHANNELS

var FrontendSTUNTURN = flag.String("stunturn", DefaultSTUNTURN, "Frontend STUN TURN servers")
var Mode = flag.String("mode", "dev", "Environment")
var StunTurnTemplate = `[{"urls":"stun:stun.l.google.com:19302"},{"urls":"stun:%s:3478"},{"urls":"turn:%s:3478","username":"root","credential":"root"}]`
var HttpPort = flag.String("httpPort", "8000", "User agent port of the app")
var HttpsPort = flag.Int("httpsPort", 443, "Https Port")
var HttpsKey = flag.String("httpsKey", "", "Https Key")
var HttpsChain = flag.String("httpsChain", "", "Https Chain")

var WSWait = 20 * time.Second
var MatchWorkerRandom = false
var ProdEnv = "prod"
var StagingEnv = "staging"

const NumKeys = 10

var FileTypeToEmulator = map[string]string{
	"gba": "gba",
	"gbc": "gba",
	"cue": "pcsx",
	"zip": "mame",
	"nes": "nes",
	"smc": "snes",
	"sfc": "snes",
	"swc": "snes",
	"fig": "snes",
	"bs":  "snes",
	"n64": "n64",
	"v64": "n64",
	"z64": "n64",
}

// There is no good way to determine main width and height of the emulator.
// When game run, frame width and height can scale abnormally.
type EmulatorMeta struct {
	Path            string
	Config          string
	Width           int
	Height          int
	AudioSampleRate int
	Fps             float64
	BaseWidth       int
	BaseHeight      int
	Ratio           float64
	Rotation        image.Rotate
	IsGlAllowed     bool
	UsesLibCo       bool
	HasMultitap     bool
}

var EmulatorConfig = map[string]EmulatorMeta{
	"gba": {
		Path:   "assets/emulator/libretro/cores/mgba_libretro",
		Width:  240,
		Height: 160,
	},
	"pcsx": {
		Path:   "assets/emulator/libretro/cores/pcsx_rearmed_libretro",
		Width:  350,
		Height: 240,
	},
	"nes": {
		Path:   "assets/emulator/libretro/cores/nestopia_libretro",
		Width:  256,
		Height: 240,
	},
	"snes": {
		Path:   "assets/emulator/libretro/cores/snes9x_libretro",
		Width:  256,
		Height: 224,
		HasMultitap: true,
	},
	"mame": {
		Path:   "assets/emulator/libretro/cores/fbneo_libretro",
		Width:  240,
		Height: 160,
	},
	"n64": {
		Path:   "assets/emulator/libretro/cores/mupen64plus_next_libretro",
		Config:   "assets/emulator/libretro/cores/mupen64plus_next_libretro.cfg",
		Width:  320,
		Height: 240,
		IsGlAllowed: true,
		UsesLibCo: true,
	},
}

var EmulatorExtension = []string{".so", ".armv7-neon-hf.so", ".dylib", ".dll"}
