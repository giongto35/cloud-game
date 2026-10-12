//go:build ramlab

package libretro

import (
	"crypto/sha256"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/nanoarch"
	"github.com/giongto35/cloud-game/v3/pkg/tourney"
)

// TestReplayDeterminism records a full UMK3 match driven by live input,
// plays it back from the frame zero state using only the input log and checks
// that the emulation (winner, detection frame, final RAM) is identical.
//
//	go test -tags ramlab -run TestReplayDeterminism ./pkg/worker/caged/libretro
func TestReplayDeterminism(t *testing.T) {
	st, err := os.ReadFile(filepath.Join("lab", "states", "fight.state"))
	if err != nil {
		t.Skip("lab/states/fight.state is missing (run lab/umk3-setup.txt first)")
	}
	emu := DefaultFrontend("replay", "snes", "nes/Ultimate Mortal Kombat 3.smc")
	defer emu.Shutdown()
	emu.Tick()
	if err := emu.RestoreSnapshot(st); err != nil {
		t.Fatal(err)
	}

	rule := tourney.ByKey("umk3")
	frame0, err := emu.SnapshotState()
	if err != nil {
		t.Fatal(err)
	}

	type outcome struct {
		res   tourney.Result
		ok    bool
		hash  [32]byte
		frame uint64
	}

	// 1. live match: P1 walks and attacks, P2 fights back in round 2
	rec := tourney.NewRecorder()
	watch := tourney.NewWatcher(rule)
	var live outcome
	emu.ResetFrames()
	emu.SetFrameHook(func(frame uint64, input [nanoarch.MaxPort]uint16, ram []byte) {
		rec.Observe(frame, input)
		if r, ok := watch.Observe(frame, ram); ok {
			live.res, live.ok = r, true
		}
	})
	press := func(port int, mask uint16) {
		var b [2]byte
		binary.LittleEndian.PutUint16(b[:], mask)
		emu.Input(port, byte(RetroPad), b[:])
	}
	const right, left, y = 1 << 7, 1 << 6, 1 << 1
	for f := 0; f < 9000 && !live.ok; f++ {
		phase := (f / 40) % 2
		p1, p2 := uint16(0), uint16(0)
		if phase == 0 {
			p1 = right
		} else if (f/6)%2 == 0 {
			p1 = y
		}
		if f > 300 && f < 1100 { // P2 attacks for a while: a realistic, mixed input log
			if phase == 0 {
				p2 = left
			} else if (f/6)%2 == 0 {
				p2 = y
			}
		}
		press(0, p1)
		press(1, p2)
		emu.Tick()
	}
	press(0, 0)
	press(1, 0)
	if !live.ok {
		t.Fatal("the live match had no winner")
	}
	live.frame = emu.Frame()
	live.hash = sha256.Sum256(nanoarch.SystemRAM())
	emu.SetFrameHook(nil)

	replay := &tourney.Replay{
		Header: tourney.Header{Game: "Ultimate Mortal Kombat 3", System: "snes", Rule: "umk3", Frames: rec.Frames(), Winner: live.res.Winner},
		State:  frame0, Events: rec.Events(),
	}
	data, err := replay.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("live: winner P%d score %v at frame %d; replay %d bytes, %d input events, %d frames",
		live.res.Winner+1, live.res.Score, live.res.Frame, len(data), len(replay.Events), replay.Header.Frames)

	// 2. playback from the decoded file
	back, err := tourney.Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if err := emu.RestoreSnapshot(back.State); err != nil {
		t.Fatal(err)
	}
	player := tourney.NewPlayer(back)
	watch2 := tourney.NewWatcher(rule)
	var played outcome
	emu.ResetFrames()
	emu.SetInputSource(func(frame uint64) ([nanoarch.MaxPort]uint16, bool) { return player.Input(frame) })
	emu.SetFrameHook(func(frame uint64, _ [nanoarch.MaxPort]uint16, ram []byte) {
		if r, ok := watch2.Observe(frame, ram); ok {
			played.res, played.ok = r, true
		}
	})
	// live input must be ignored during playback
	press(0, right|y)
	for emu.Frame() < live.frame {
		emu.Tick()
	}
	played.hash = sha256.Sum256(nanoarch.SystemRAM())
	emu.SetInputSource(nil)
	emu.SetFrameHook(nil)

	if !played.ok || played.res != live.res {
		t.Fatalf("playback result %+v differs from live %+v", played.res, live.res)
	}
	if played.hash != live.hash {
		t.Fatal("playback RAM differs from the live match: emulation is not deterministic")
	}
	t.Logf("playback: identical winner, detection frame and RAM hash after %d frames", live.frame)
}
