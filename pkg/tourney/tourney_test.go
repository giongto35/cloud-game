package tourney

import (
	"bytes"
	"testing"
)

func ram() []byte { return make([]byte, 0x20000) }

func TestFindRule(t *testing.T) {
	cases := map[string]string{
		"Ultimate Mortal Kombat 3":        "umk3",
		"Super Bomberman 2 (USA) (PT-BR)": "sb2",
		"Super Mario Kart (USA) (PT-BR)":  "smk",
		"Mortal Kombat 3 (USA)":           "",
		"Super Bomberman 3 (Europe)":      "",
	}
	for game, key := range cases {
		r := Find(game)
		got := ""
		if r != nil {
			got = r.Key
		}
		if got != key {
			t.Errorf("%q: got %q, want %q", game, got, key)
		}
	}
}

func run(w *Watcher, m []byte, from, n uint64) (Result, bool) {
	for f := from; f < from+n; f++ {
		if r, ok := w.Observe(f, m); ok {
			return r, true
		}
	}
	return Result{}, false
}

func TestWatcherUMK3(t *testing.T) {
	r := ByKey("umk3")
	w := NewWatcher(r)
	m := ram()
	// leftover values from a previous match must not count before arming
	m[0x3AA6] = 2
	if _, ok := run(w, m, 0, 100); ok {
		t.Fatal("must not report before arming")
	}
	m[0x3AA6] = 0
	run(w, m, 100, 10)
	m[0x3AA6] = 1 // P1 won round 1
	m[0x3AA8] = 1 // P2 won round 2
	if _, ok := run(w, m, 110, 100); ok {
		t.Fatal("1-1 is not a result")
	}
	m[0x3AA8] = 2 // P2 wins round 3
	// a transient value shorter than Stable is ignored
	if _, ok := run(w, m, 210, 10); ok {
		t.Fatal("must wait for a stable value")
	}
	res, ok := run(w, m, 220, 100)
	if !ok || res.Winner != 1 || res.Score != [2]byte{1, 2} {
		t.Fatalf("unexpected result %+v %v", res, ok)
	}
	if _, ok := w.Observe(1000, m); ok || !w.Done() {
		t.Fatal("a watcher reports once")
	}
}

func TestWatcherBombermanTargetFromRAM(t *testing.T) {
	w := NewWatcher(ByKey("sb2"))
	m := ram()
	m[0x1CA7] = 4 // "Troféus 5"
	run(w, m, 0, 5)
	m[0x5E66] = 3
	if _, ok := run(w, m, 5, 100); ok {
		t.Fatal("3 of 5 trophies is not a win")
	}
	m[0x5E66] = 5
	res, ok := run(w, m, 105, 100)
	if !ok || res.Winner != 0 {
		t.Fatalf("P1 should win with 5 trophies: %+v", res)
	}
}

func TestWatcherMarioKartAndTie(t *testing.T) {
	w := NewWatcher(ByKey("smk"))
	m := ram()
	m[0x10C1], m[0x11C1] = 128, 127
	run(w, m, 0, 5)
	m[0x10C1], m[0x11C1] = 133, 133 // impossible photo finish: undecided
	if _, ok := run(w, m, 5, 100); ok {
		t.Fatal("a tie must not produce a winner")
	}
	w2 := NewWatcher(ByKey("smk"))
	m[0x10C1], m[0x11C1] = 132, 132
	run(w2, m, 0, 5)
	m[0x11C1] = 133
	res, ok := run(w2, m, 5, 10)
	if !ok || res.Winner != 1 {
		t.Fatalf("P2 crossed the line first: %+v", res)
	}
}

func TestReplayRoundTrip(t *testing.T) {
	rec := NewRecorder()
	var in [MaxPorts]uint16
	for f := uint64(0); f < 10000; f++ {
		switch {
		case f%37 == 0:
			in[0] ^= 1 << 8 // A
		case f%53 == 0:
			in[1] ^= 1 << 7 // right
		}
		rec.Observe(f, in)
	}
	state := bytes.Repeat([]byte{1, 2, 3, 4}, 50000)
	rp := &Replay{Header: Header{Game: "Super Mario Kart", System: "snes", Fps: 60.0988, Frames: rec.Frames(), Winner: 1},
		State: state, Events: rec.Events()}
	data, err := rp.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 20000 {
		t.Fatalf("replay too big: %d bytes", len(data))
	}
	back, err := Decode(data)
	if err != nil {
		t.Fatal(err)
	}
	if back.Header != rp.Header || !bytes.Equal(back.State, state) || len(back.Events) != len(rp.Events) {
		t.Fatalf("round trip mismatch: %+v", back.Header)
	}

	// the player reproduces the exact per-frame input
	rec2 := NewRecorder()
	p := NewPlayer(back)
	for f := uint64(0); ; f++ {
		keys, ok := p.Input(f)
		if !ok {
			break
		}
		rec2.Observe(f, keys)
	}
	if rec2.Frames() != rec.Frames() || len(rec2.Events()) != len(rec.Events()) {
		t.Fatalf("playback differs: %d/%d frames, %d/%d events", rec2.Frames(), rec.Frames(), len(rec2.Events()), len(rec.Events()))
	}
	for i, e := range rec.Events() {
		if rec2.Events()[i] != e {
			t.Fatalf("event %d differs: %+v vs %+v", i, rec2.Events()[i], e)
		}
	}

	if _, err := Decode([]byte("BRPL\x01garbage")); err == nil {
		t.Fatal("garbage must not decode")
	}
}
