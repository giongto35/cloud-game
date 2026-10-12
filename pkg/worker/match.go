package worker

import (
	"fmt"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/tourney"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged/libretro/nanoarch"
)

// resultTail keeps recording a few seconds after the result (the victory screen).
const resultTail = 240

// matchRuntime records a tournament match and detects its result from the RAM.
// hook runs on the emulation thread; everything else happens in finish().
type matchRuntime struct {
	rule   *tourney.Rule
	header tourney.Header
	room   string
	state0 []byte

	rec      *tourney.Recorder
	watch    *tourney.Watcher
	result   *tourney.Result
	finished bool

	report func(api.MatchResultRequest)
	log    *logger.Logger
}

func (m *matchRuntime) hook(frame uint64, input [nanoarch.MaxPort]uint16, ram []byte) {
	if m.finished {
		return
	}
	m.rec.Observe(frame, input)
	if m.result == nil {
		if r, ok := m.watch.Observe(frame, ram); ok {
			m.result = &r
			m.log.Info().Int("winner", r.Winner+1).Msgf("match %d: result detected at frame %d %v", m.header.Match, r.Frame, r.Score)
		}
		return
	}
	if frame >= m.result.Frame+resultTail {
		m.finished = true
		go m.finish() // the recorder isn't touched by the hook anymore
	}
}

func (m *matchRuntime) finish() {
	res := *m.result
	m.header.Frames = m.rec.Frames()
	m.header.Winner = res.Winner
	m.header.CreatedAt = time.Now().Unix()
	rp := &tourney.Replay{Header: m.header, State: m.state0, Events: m.rec.Events()}
	data, err := rp.Bytes()
	if err != nil {
		m.log.Error().Err(err).Msg("replay encoding failed")
		data = nil
	}
	m.report(api.MatchResultRequest{
		Room: m.room, Match: rp.Header.Match, Winner: res.Winner,
		Score: [2]int{int(res.Score[0]), int(res.Score[1])}, Frame: res.Frame, Replay: data,
	})
}

// setupMatchRoom prepares a freshly loaded game for a tournament match or a replay.
// It must run before the emulation loop starts.
func (c *coordinator) setupMatchRoom(fe *libretro.Frontend, rq api.StartGameRequest, rid, game, system string) error {
	// the first frame initialises the core; the recording starts right after it
	fe.Tick()

	if len(rq.Replay) > 0 {
		rp, err := tourney.Decode(rq.Replay)
		if err != nil {
			return err
		}
		if err := fe.RestoreSnapshot(rp.State); err != nil {
			return fmt.Errorf("replay state: %w", err)
		}
		fe.ResetFrames()
		player := tourney.NewPlayer(rp)
		fe.SetInputSource(func(frame uint64) ([nanoarch.MaxPort]uint16, bool) { return player.Input(frame) })
		c.log.Info().Msgf("replay room %s: %d frames, %d input events", rid, rp.Header.Frames, len(rp.Events))
		return nil
	}

	t := rq.Tournament
	rule := tourney.ByKey(t.Rule)
	if rule == nil {
		return fmt.Errorf("unknown tournament rule %q", t.Rule)
	}
	state0, err := fe.SnapshotState()
	if err != nil {
		return fmt.Errorf("frame zero state: %w", err)
	}
	fe.ResetFrames()
	m := &matchRuntime{
		rule: rule, room: rid, state0: state0,
		header: tourney.Header{Game: game, System: system, Rule: rule.Key, Match: t.Match, Fps: fe.FPS()},
		rec:    tourney.NewRecorder(), watch: tourney.NewWatcher(rule),
		report: func(r api.MatchResultRequest) { c.Notify(api.MatchResult, r) },
		log:    c.log,
	}
	fe.SetFrameHook(m.hook)
	c.log.Info().Msgf("tournament room %s: match %d, rule %s", rid, t.Match, rule.Key)
	return nil
}
