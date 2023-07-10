package room

import (
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/network/webrtc"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro"
)

func TestGoodWithRecorder(t *testing.T) {
	WithRecorder(&libretro.Caged{Emulator: &libretro.RecordingFrontend{}})
}

func TestBadWithRecorder(t *testing.T) {
	defer func() { _ = recover() }()
	WithEmulator(libretro.Caged{})
	t.Errorf("no panic")
}

func TestGoodWithEmulator(t *testing.T) { WithEmulator(&libretro.Caged{}) }

func TestBadWithEmulator(t *testing.T) {
	defer func() { _ = recover() }()
	WithEmulator(libretro.Caged{}) // not a pointer
	t.Errorf("no panic")
}

func TestGoodWithWebRTCCast(t *testing.T) {
	WithWebRTC(GameSession{AppSession: AppSession{Session: &webrtc.Peer{}}}.Session)
}

func TestBadWithWebRTCCast(t *testing.T) {
	defer func() { _ = recover() }()
	WithWebRTC(GameSession{}) // not a Session due to deep nesting
	t.Errorf("no panic")
}
