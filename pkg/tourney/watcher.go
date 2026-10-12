package tourney

// Result is a detected match result.
type Result struct {
	Winner int     // 0 = player 1, 1 = player 2
	Score  [2]byte // final scores
	Frame  uint64  // frame of the detection
}

// Watcher evaluates a Rule on every frame. It is not safe for concurrent use:
// call Observe from the emulation thread only (frame hook).
type Watcher struct {
	rule *Rule

	armed     bool
	candidate int // player holding a winning score, -1 = none
	since     uint64
	done      bool
}

func NewWatcher(r *Rule) *Watcher { return &Watcher{rule: r, candidate: -1} }

func (w *Watcher) target(ram []byte) byte {
	if w.rule.TargetAddr != 0 && int(w.rule.TargetAddr) < len(ram) {
		if t := ram[w.rule.TargetAddr] + w.rule.TargetAdd; t > 0 {
			return t
		}
	}
	return w.rule.Target
}

// Observe checks the RAM of one frame and returns the result once it is final.
//
// Safety rules:
//   - arming: the watcher only counts after both scores were seen below the
//     target (leftovers from menus or a previous match are ignored);
//   - stability: a winning score must hold for Rule.Stable frames;
//   - a tie (both reach the target) never produces a winner.
func (w *Watcher) Observe(frame uint64, ram []byte) (Result, bool) {
	if w.done || ram == nil {
		return Result{}, false
	}
	a, b := w.rule.Score[0], w.rule.Score[1]
	if int(a) >= len(ram) || int(b) >= len(ram) {
		return Result{}, false
	}
	s := [2]byte{ram[a], ram[b]}
	t := w.target(ram)

	if !w.armed {
		if s[0] < t && s[1] < t {
			w.armed = true
		}
		return Result{}, false
	}

	leader := -1
	switch {
	case s[0] >= t && s[1] >= t:
		leader = -1 // tie: undecided
	case s[0] >= t:
		leader = 0
	case s[1] >= t:
		leader = 1
	}
	if leader != w.candidate {
		w.candidate, w.since = leader, frame
		return Result{}, false
	}
	if leader < 0 || frame-w.since < uint64(max(w.rule.Stable, 1)) {
		return Result{}, false
	}
	w.done = true
	return Result{Winner: leader, Score: s, Frame: frame}, true
}

// Done reports whether a result was already produced.
func (w *Watcher) Done() bool { return w.done }
