package tournament

import (
	"context"
	"fmt"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/store"
	"github.com/giongto35/cloud-game/v3/pkg/tourney"
)

type fakeRooms struct {
	mu       sync.Mutex
	free     int
	reserved map[int64]Assignment
	present  map[int64][]int64
	notices  []string
	ended    []int64
	maxBusy  int
}

func newFakeRooms(free int) *fakeRooms {
	return &fakeRooms{free: free, reserved: map[int64]Assignment{}, present: map[int64][]int64{}}
}

func (f *fakeRooms) ReserveMatch(a Assignment) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.free == 0 {
		return false
	}
	f.free--
	f.reserved[a.Match] = a
	f.maxBusy = max(f.maxBusy, len(f.reserved))
	return true
}

func (f *fakeRooms) EndMatch(match int64, _ string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.reserved[match]; ok {
		delete(f.reserved, match)
		f.free++
	}
	f.ended = append(f.ended, match)
}

func (f *fakeRooms) Present(match int64) []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.present[match]
}

func (f *fakeRooms) Notify(users []int64, kind, text string, match int64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.notices = append(f.notices, fmt.Sprintf("%v:%s:%d", users, kind, match))
}

func (f *fakeRooms) running() []int64 {
	f.mu.Lock()
	defer f.mu.Unlock()
	var out []int64
	for id := range f.reserved {
		out = append(out, id)
	}
	slices.Sort(out)
	return out
}

func setup(t *testing.T, players, servers int) (*Manager, *fakeRooms, *store.Store, int64, []int64) {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ctx := context.Background()
	var ids []int64
	for i := range players + 1 {
		u := &store.User{Email: fmt.Sprintf("u%d@t.com", i), Name: fmt.Sprintf("Jogador %d", i), PasswordHash: "x",
			Role: store.RoleUser, Status: store.StatusApproved}
		if err := st.CreateUser(ctx, u); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, u.Id)
	}
	m := NewManager(st, Config{MaxConcurrent: 2, JoinTimeout: time.Minute, ReplayDir: t.TempDir()}, logger.Default())
	rooms := newFakeRooms(servers)
	if err := m.Start(rooms); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(m.Stop)
	tr, err := m.Create(ctx, "Copa Belko", "Ultimate Mortal Kombat 3", "snes", ids[0])
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range ids[1:] {
		if err := m.Join(ctx, tr.Id, id); err != nil {
			t.Fatal(err)
		}
	}
	return m, rooms, st, tr.Id, ids[1:]
}

// play makes the room report a result for every running match (P1 or P2 wins).
func play(t *testing.T, m *Manager, rooms *fakeRooms, st *store.Store, slot int) int {
	t.Helper()
	ctx := context.Background()
	n := 0
	for _, id := range rooms.running() {
		m.MatchStarted(id, fmt.Sprintf("room-%d", id))
		if err := m.Result(id, slot, [2]int{2, 1}, nil); err != nil {
			t.Fatalf("result of match %d: %v", id, err)
		}
		mt, _ := st.Match(ctx, id)
		if mt.Status != StatusFinished || mt.Source != SourceRAM {
			t.Fatalf("match %d not finished: %+v", id, mt)
		}
		n++
	}
	return n
}

func TestBracketBuild(t *testing.T) {
	players := []int64{1, 2, 3, 4, 5}
	ms := Build(players)
	if len(ms) != 7 || Rounds(5) != 3 {
		t.Fatalf("5 players need an 8 slot bracket: %d matches, %d rounds", len(ms), Rounds(5))
	}
	byes := 0
	for _, m := range ms[:4] {
		if m.P1 == nil {
			t.Fatal("no first round match may be empty")
		}
		if m.P2 == nil {
			byes++
			if m.Status != StatusFinished || *m.WinnerId != *m.P1 {
				t.Fatal("a bye is a walkover")
			}
		}
	}
	if byes != 3 {
		t.Fatalf("expected 3 byes, got %d", byes)
	}
	names := []string{RoundName(1, 4), RoundName(2, 4), RoundName(3, 4), RoundName(4, 4), RoundName(1, 5)}
	want := []string{"Oitavas de final", "Quartas de final", "Semifinal", "Final", "Rodada de 32"}
	if !slices.Equal(names, want) {
		t.Fatalf("round names %v", names)
	}
	sh := Shuffle(players)
	slices.Sort(sh)
	if !slices.Equal(sh, players) {
		t.Fatal("shuffle must keep every player")
	}
}

func TestTournamentRunsWithTwoConcurrentMatches(t *testing.T) {
	ctx := context.Background()
	m, rooms, st, tid, _ := setup(t, 8, 4) // 4 free servers, but only 2 concurrent matches
	if err := m.Begin(ctx, tid); err != nil {
		t.Fatal(err)
	}
	if got := len(rooms.running()); got != 2 {
		t.Fatalf("expected 2 concurrent matches, got %d", got)
	}
	running, queued := m.Snapshot()
	if len(running) != 2 || len(queued) != 2 {
		t.Fatalf("expected 2 running + 2 queued, got %v %v", running, queued)
	}
	// play everything: each result frees a slot and starts the next match
	total := 0
	for i := 0; i < 20 && len(rooms.running()) > 0; i++ {
		total += play(t, m, rooms, st, i%2)
	}
	if total != 7 {
		t.Fatalf("8 players play 7 matches, got %d", total)
	}
	if rooms.maxBusy > 2 {
		t.Fatalf("never more than 2 matches at once, saw %d", rooms.maxBusy)
	}
	tr, _ := st.Tournament(ctx, tid)
	if tr.Status != TFinished || tr.WinnerId == nil {
		t.Fatalf("tournament should be finished with a winner: %+v", tr)
	}
	final, _ := st.MatchAt(ctx, tid, 3, 0)
	if *final.WinnerId != *tr.WinnerId {
		t.Fatal("the final winner is the champion")
	}
}

func TestQueueWaitsForFreeServers(t *testing.T) {
	ctx := context.Background()
	m, rooms, st, tid, _ := setup(t, 4, 1) // a single free server
	if err := m.Begin(ctx, tid); err != nil {
		t.Fatal(err)
	}
	if got := len(rooms.running()); got != 1 {
		t.Fatalf("one server, one match: got %d", got)
	}
	play(t, m, rooms, st, 0)
	if got := len(rooms.running()); got != 1 {
		t.Fatalf("the next match must start on the freed server, got %d", got)
	}
}

func TestByesAndWalkovers(t *testing.T) {
	ctx := context.Background()
	m, rooms, st, tid, players := setup(t, 3, 4)
	if err := m.Begin(ctx, tid); err != nil {
		t.Fatal(err)
	}
	// 3 players: one bye in round 1, a single real first round match
	run := rooms.running()
	if len(run) != 1 {
		t.Fatalf("expected one playable first round match, got %v", run)
	}
	a, _ := m.Active(run[0])
	// only player 1 shows up: walkover after the deadline
	rooms.present[run[0]] = []int64{a.Players[0]}
	m.checkDeadlines(time.Now().Add(2 * time.Minute))
	mt, _ := st.Match(ctx, run[0])
	if mt.Status != StatusFinished || mt.Source != SourceWalkover || *mt.WinnerId != a.Players[0] {
		t.Fatalf("expected a walkover: %+v", mt)
	}
	// the final (bye winner x walkover winner) starts
	run = rooms.running()
	if len(run) != 1 {
		t.Fatalf("the final should be running, got %v", run)
	}
	// nobody shows up twice: the admin decides
	m.checkDeadlines(time.Now().Add(2 * time.Minute))
	m.Kick()
	run = rooms.running()
	m.checkDeadlines(time.Now().Add(2 * time.Minute))
	final, _ := st.MatchAt(ctx, tid, 2, 0)
	if final.Status != StatusStalled {
		t.Fatalf("expected a stalled final, got %s (%v)", final.Status, run)
	}
	if err := m.SetWinner(final.Id, players[0]+999); err != ErrBadWinner {
		t.Fatalf("the winner must be a player of the match: %v", err)
	}
	if err := m.SetWinner(final.Id, *final.P2); err != nil {
		t.Fatal(err)
	}
	tr, _ := st.Tournament(ctx, tid)
	if tr.Status != TFinished || *tr.WinnerId != *final.P2 {
		t.Fatalf("admin decision should end the tournament: %+v", tr)
	}
}

func TestReplayIsStored(t *testing.T) {
	ctx := context.Background()
	m, rooms, st, tid, _ := setup(t, 2, 1)
	if err := m.Begin(ctx, tid); err != nil {
		t.Fatal(err)
	}
	id := rooms.running()[0]
	m.MatchStarted(id, "room")
	rp := &tourney.Replay{Header: tourney.Header{Game: "Ultimate Mortal Kombat 3", Frames: 10}, State: []byte{1, 2, 3},
		Events: []tourney.Event{{Frame: 1, Port: 0, Mask: 8}}}
	data, _ := rp.Bytes()
	if err := m.Result(id, 1, [2]int{0, 2}, data); err != nil {
		t.Fatal(err)
	}
	mt, _ := st.Match(ctx, id)
	if mt.ReplayId == nil || mt.Score != "0-2" {
		t.Fatalf("replay not linked: %+v", mt)
	}
	info, back, err := m.ReplayData(ctx, *mt.ReplayId)
	if err != nil || info.Frames != 10 || len(back) != len(data) {
		t.Fatalf("replay not readable: %v %+v", err, info)
	}
	if err := m.Result(id, 0, [2]int{2, 0}, nil); err != ErrNotRunning {
		t.Fatal("a finished match can't get another result")
	}
	if tr, _ := st.Tournament(ctx, tid); tr.Status != TFinished {
		t.Fatal("2 players: one match decides the tournament")
	}
}
