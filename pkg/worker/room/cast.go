package room

import (
	"github.com/giongto35/cloud-game/v3/pkg/com"
	"github.com/giongto35/cloud-game/v3/pkg/network/webrtc"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro"
)

type GameRouter struct {
	Router[*GameSession]
}

func NewGameRouter() *GameRouter {
	u := com.NewNetMap[SessionKey, *GameSession]()
	return &GameRouter{Router: Router[*GameSession]{users: &u}}
}

func WithEmulator(wtf any) *libretro.Caged { return wtf.(*libretro.Caged) }
func WithRecorder(wtf any) *libretro.RecordingFrontend {
	return (WithEmulator(wtf).Emulator).(*libretro.RecordingFrontend)
}
func WithWebRTC(wtf Session) *webrtc.Peer { return wtf.(*webrtc.Peer) }
