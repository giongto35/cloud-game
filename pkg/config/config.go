package config

import (
	"flag"
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
var ProdEnv = "prod"
var StagingEnv = "staging"

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

var SupportedRomExtensions = listSupportedRomExtensions()

var EmulatorExtension = []string{".so", ".armv7-neon-hf.so", ".dylib", ".dll"}

func listSupportedRomExtensions() []string {
	m := make([]string, 0, len(FileTypeToEmulator))
	for k := range FileTypeToEmulator {
		m = append(m, k)
	}
	return m
}
