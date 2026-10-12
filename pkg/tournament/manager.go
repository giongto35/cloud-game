package tournament

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/store"
	"github.com/giongto35/cloud-game/v3/pkg/tourney"
)

var (
	ErrNotFound     = errors.New("não encontrado")
	ErrState        = errors.New("o torneio não está nesta fase")
	ErrFewPlayers   = errors.New("são necessários pelo menos 2 participantes")
	ErrNotRunning   = errors.New("a partida não está em andamento")
	ErrBadWinner    = errors.New("o vencedor precisa ser um dos jogadores da partida")
	ErrUnknownGame  = errors.New("este jogo não tem detecção automática de resultado")
	ErrAlreadyJoind = errors.New("você já está inscrito")
)

// Assignment is a match that needs a server.
type Assignment struct {
	Match      int64
	Tournament int64
	Name       string // tournament name
	Round      string
	Game       string
	Rule       string
	Players    [2]int64
	Names      [2]string
}

// Rooms is implemented by the coordinator hub: servers and players.
type Rooms interface {
	// ReserveMatch reserves a free server for the match; false if none is free.
	ReserveMatch(a Assignment) bool
	// EndMatch announces the result to the room and closes it after a while
	// (or just drops the reservation when the room never started).
	EndMatch(match int64, text string)
	// Present returns the participants currently in the match room.
	Present(match int64) []int64
	// Notify sends a notice to the given accounts (nil = everyone online).
	Notify(users []int64, kind, text string, match int64)
}

type Config struct {
	MaxConcurrent int
	JoinTimeout   time.Duration
	ReplayDir     string
}

// semaphore limits the number of matches played at the same time.
type semaphore chan struct{}

func (s semaphore) TryAcquire() bool {
	select {
	case s <- struct{}{}:
		return true
	default:
		return false
	}
}

func (s semaphore) Release() {
	select {
	case <-s:
	default:
	}
}

type active struct {
	assign   Assignment
	deadline time.Time // players must join before it
}

// Manager owns the tournament state machine. All transitions are serialized by mu;
// calls into Rooms that may call back (Notify, EndMatch) happen outside of it.
type Manager struct {
	st    *store.Store
	conf  Config
	log   *logger.Logger
	rooms Rooms

	mu     sync.Mutex
	sem    semaphore
	queue  []int64           // ready matches, FIFO
	active map[int64]*active // matches holding a semaphore slot
	done   chan struct{}
}

func NewManager(st *store.Store, conf Config, log *logger.Logger) *Manager {
	if conf.MaxConcurrent <= 0 {
		conf.MaxConcurrent = 2
	}
	if conf.JoinTimeout <= 0 {
		conf.JoinTimeout = 3 * time.Minute
	}
	if conf.ReplayDir == "" {
		conf.ReplayDir = "./data/replays"
	}
	return &Manager{st: st, conf: conf, log: log, sem: make(semaphore, conf.MaxConcurrent),
		active: map[int64]*active{}, done: make(chan struct{})}
}

func (m *Manager) MaxConcurrent() int { return m.conf.MaxConcurrent }

// Start recovers unfinished matches (rooms don't survive a restart) and runs the timers.
func (m *Manager) Start(rooms Rooms) error {
	m.rooms = rooms
	ctx := context.Background()
	open, err := m.st.OpenMatches(ctx)
	if err != nil {
		return err
	}
	m.mu.Lock()
	for i := range open {
		mt := &open[i]
		if mt.Status != StatusReady {
			mt.Status, mt.RoomId = StatusReady, ""
			_ = m.st.UpdateMatch(ctx, mt)
		}
		m.queue = append(m.queue, mt.Id)
	}
	m.mu.Unlock()
	go m.loop()
	m.Kick()
	return nil
}

func (m *Manager) Stop() { close(m.done) }

func (m *Manager) loop() {
	t := time.NewTicker(5 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-m.done:
			return
		case <-t.C:
			m.checkDeadlines(time.Now())
			m.Kick()
		}
	}
}

type notice struct {
	users []int64
	kind  string
	text  string
	match int64
}

func (m *Manager) flush(ns []notice) {
	for _, n := range ns {
		m.rooms.Notify(n.users, n.kind, n.text, n.match)
	}
}

// Kick tries to start queued matches (call it when a server becomes free).
func (m *Manager) Kick() {
	if m.rooms == nil {
		return
	}
	m.mu.Lock()
	ns := m.dispatchLocked()
	m.mu.Unlock()
	m.flush(ns)
}

func (m *Manager) dispatchLocked() (ns []notice) {
	ctx := context.Background()
	for len(m.queue) > 0 {
		if !m.sem.TryAcquire() {
			return
		}
		id := m.queue[0]
		a, mt, err := m.assignment(ctx, id)
		if err != nil || mt.Status != StatusReady {
			m.sem.Release()
			m.queue = m.queue[1:]
			continue
		}
		if !m.rooms.ReserveMatch(a) {
			m.sem.Release() // no free server: wait for the next Kick
			return
		}
		m.queue = m.queue[1:]
		mt.Status, mt.Attempts = StatusStarting, mt.Attempts+1
		_ = m.st.UpdateMatch(ctx, mt)
		m.active[id] = &active{assign: a, deadline: time.Now().Add(m.conf.JoinTimeout)}
		m.log.Info().Msgf("tournament %d: match %d (%s) called: %s x %s", a.Tournament, id, a.Round, a.Names[0], a.Names[1])
		for i, p := range a.Players {
			ns = append(ns, notice{users: []int64{p}, kind: "match_ready", match: id, text: callText(a, i)})
		}
		ns = append(ns, notice{kind: "tournaments", match: id})
	}
	return
}

func (m *Manager) assignment(ctx context.Context, id int64) (Assignment, *store.Match, error) {
	mt, err := m.st.Match(ctx, id)
	if err != nil {
		return Assignment{}, nil, err
	}
	t, err := m.st.Tournament(ctx, mt.TournamentId)
	if err != nil {
		return Assignment{}, nil, err
	}
	if mt.P1 == nil || mt.P2 == nil {
		return Assignment{}, nil, ErrState
	}
	players, _ := m.st.TournamentPlayers(ctx, t.Id)
	names := map[int64]string{}
	for _, p := range players {
		names[p.UserId] = p.Name
	}
	all, _ := m.st.Matches(ctx, t.Id)
	rounds := 0
	for _, x := range all {
		rounds = max(rounds, x.Round)
	}
	a := Assignment{Match: id, Tournament: t.Id, Name: t.Name, Round: RoundName(mt.Round, rounds), Game: t.Game, Rule: t.Rule,
		Players: [2]int64{*mt.P1, *mt.P2}, Names: [2]string{names[*mt.P1], names[*mt.P2]}}
	return a, mt, nil
}

func callText(a Assignment, i int) string {
	return fmt.Sprintf("%s · %s: sua partida contra %s está pronta!", a.Name, a.Round, a.Names[1-i])
}

// CallFor returns the match the account was called to (starting or running),
// so the call can be repeated when the player (re)connects.
func (m *Manager) CallFor(acc int64) (int64, string, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id, a := range m.active {
		for i, p := range a.assign.Players {
			if p == acc {
				return id, callText(a.assign, i), true
			}
		}
	}
	return 0, "", false
}

// MatchStarted is called when the match room is live.
func (m *Manager) MatchStarted(id int64, room string) {
	ctx := context.Background()
	m.mu.Lock()
	defer m.mu.Unlock()
	mt, err := m.st.Match(ctx, id)
	if err != nil || mt.Status != StatusStarting {
		return
	}
	now := time.Now()
	mt.Status, mt.RoomId, mt.StartedAt = StatusRunning, room, &now
	_ = m.st.UpdateMatch(ctx, mt)
	go m.rooms.Notify(nil, "tournaments", "", id)
}

// Assignment returns the live assignment of a match holding a server.
func (m *Manager) Active(id int64) (Assignment, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	a, ok := m.active[id]
	if !ok {
		return Assignment{}, false
	}
	return a.assign, true
}

// Result records the winner detected by the worker (slot 0/1) and the replay.
func (m *Manager) Result(id int64, slot int, score [2]int, replay []byte) error {
	ctx := context.Background()
	m.mu.Lock()
	act, ok := m.active[id]
	if !ok {
		m.mu.Unlock()
		return ErrNotRunning
	}
	mt, err := m.st.Match(ctx, id)
	if err != nil || mt.Status != StatusRunning || slot < 0 || slot > 1 {
		m.mu.Unlock()
		return ErrNotRunning
	}
	if len(replay) > 0 {
		if rid, err := m.saveReplay(ctx, mt, act.assign.Game, replay); err == nil {
			mt.ReplayId = &rid
		} else {
			m.log.Error().Err(err).Msg("replay save failed")
		}
	}
	winner := act.assign.Players[slot]
	mt.Score = fmt.Sprintf("%d-%d", score[0], score[1])
	ns := m.finishLocked(ctx, mt, winner, SourceRAM)
	text := fmt.Sprintf("Fim de partida! %s venceu (%s).", act.assign.Names[slot], mt.Score)
	m.mu.Unlock()

	m.rooms.EndMatch(id, text)
	m.flush(ns)
	m.Kick()
	return nil
}

// SetWinner lets the admin decide a match (stalled, disputed or not detected).
func (m *Manager) SetWinner(id, winner int64) error {
	ctx := context.Background()
	m.mu.Lock()
	mt, err := m.st.Match(ctx, id)
	if err != nil {
		m.mu.Unlock()
		return ErrNotFound
	}
	if mt.Status == StatusFinished || mt.Status == StatusPending {
		m.mu.Unlock()
		return ErrState
	}
	if (mt.P1 == nil || *mt.P1 != winner) && (mt.P2 == nil || *mt.P2 != winner) {
		m.mu.Unlock()
		return ErrBadWinner
	}
	_, wasActive := m.active[id]
	ns := m.finishLocked(ctx, mt, winner, SourceAdmin)
	m.mu.Unlock()
	if wasActive {
		m.rooms.EndMatch(id, "A administração definiu o resultado desta partida.")
	}
	m.flush(ns)
	m.Kick()
	return nil
}

// finishLocked closes a match, frees its slot and advances the winner.
func (m *Manager) finishLocked(ctx context.Context, mt *store.Match, winner int64, source string) []notice {
	now := time.Now()
	mt.Status, mt.WinnerId, mt.Source, mt.FinishedAt = StatusFinished, &winner, source, &now
	_ = m.st.UpdateMatch(ctx, mt)
	if _, ok := m.active[mt.Id]; ok {
		delete(m.active, mt.Id)
		m.sem.Release()
	}
	m.queue = slices.DeleteFunc(m.queue, func(x int64) bool { return x == mt.Id })
	return append(m.advanceLocked(ctx, mt), notice{kind: "tournaments", match: mt.Id})
}

// advanceLocked moves the winner to the next round or ends the tournament.
func (m *Manager) advanceLocked(ctx context.Context, mt *store.Match) []notice {
	next, err := m.st.MatchAt(ctx, mt.TournamentId, mt.Round+1, mt.Position/2)
	if errors.Is(err, store.ErrNotFound) { // that was the final
		_ = m.st.SetTournamentStatus(ctx, mt.TournamentId, TFinished, mt.WinnerId)
		t, _ := m.st.Tournament(ctx, mt.TournamentId)
		name := ""
		if t != nil {
			name = t.Name
		}
		m.log.Info().Msgf("tournament %d finished", mt.TournamentId)
		return []notice{{kind: "tournament_over", text: name, match: mt.Id}}
	}
	if err != nil {
		return nil
	}
	w := *mt.WinnerId
	if mt.Position%2 == 0 {
		next.P1 = &w
	} else {
		next.P2 = &w
	}
	if next.P1 != nil && next.P2 != nil && next.Status == StatusPending {
		now := time.Now()
		next.Status, next.ReadyAt = StatusReady, &now
		m.queue = append(m.queue, next.Id)
	}
	_ = m.st.UpdateMatch(ctx, next)
	return nil
}

// RoomLost is called when a match room closed before a result (everybody left,
// the server went away). The match goes back to the front of the queue;
// after 3 attempts the admin decides.
func (m *Manager) RoomLost(id int64) {
	ctx := context.Background()
	m.mu.Lock()
	if _, ok := m.active[id]; !ok {
		m.mu.Unlock()
		return
	}
	delete(m.active, id)
	m.sem.Release()
	mt, err := m.st.Match(ctx, id)
	if err == nil && mt.Status != StatusFinished {
		mt.RoomId = ""
		if mt.Attempts < 3 {
			mt.Status = StatusReady
			m.queue = append([]int64{id}, m.queue...)
		} else {
			mt.Status = StatusStalled
		}
		_ = m.st.UpdateMatch(ctx, mt)
		m.log.Warn().Msgf("match %d: room lost (attempt %d)", id, mt.Attempts)
	}
	m.mu.Unlock()
	m.rooms.Notify(nil, "tournaments", "", id)
	m.Kick()
}

// checkDeadlines handles players that didn't show up.
func (m *Manager) checkDeadlines(now time.Time) {
	ctx := context.Background()
	var ns []notice
	var ended []int64
	m.mu.Lock()
	for id, a := range m.active {
		if now.Before(a.deadline) {
			continue
		}
		mt, err := m.st.Match(ctx, id)
		if err != nil || mt.Status != StatusStarting {
			continue
		}
		present := m.rooms.Present(id)
		var here []int64
		for _, p := range a.assign.Players {
			if slices.Contains(present, p) {
				here = append(here, p)
			}
		}
		ended = append(ended, id)
		if len(here) == 1 { // walkover
			ns = append(ns, m.finishLocked(ctx, mt, here[0], SourceWalkover)...)
			continue
		}
		// nobody (or a stuck room): try once more at the end of the queue, then ask the admin
		delete(m.active, id)
		m.sem.Release()
		if mt.Attempts < 2 {
			mt.Status = StatusReady
			m.queue = append(m.queue, id)
		} else {
			mt.Status = StatusStalled
		}
		_ = m.st.UpdateMatch(ctx, mt)
		ns = append(ns, notice{kind: "tournaments", match: id})
	}
	m.mu.Unlock()
	for _, id := range ended {
		m.rooms.EndMatch(id, "O tempo para entrar na partida acabou.")
	}
	m.flush(ns)
	if len(ended) > 0 {
		m.Kick()
	}
}

func (m *Manager) saveReplay(ctx context.Context, mt *store.Match, game string, data []byte) (int64, error) {
	if _, err := tourney.Decode(data); err != nil {
		return 0, err
	}
	if err := os.MkdirAll(m.conf.ReplayDir, 0o750); err != nil {
		return 0, err
	}
	name := fmt.Sprintf("t%d-r%d-p%d-m%d.brpl", mt.TournamentId, mt.Round, mt.Position, mt.Id)
	if err := os.WriteFile(filepath.Join(m.conf.ReplayDir, name), data, 0o640); err != nil {
		return 0, err
	}
	rp, _ := tourney.Decode(data)
	info := &store.ReplayInfo{MatchId: &mt.Id, Game: game, Frames: int64(rp.Header.Frames), Size: len(data), File: name}
	if err := m.st.CreateReplay(ctx, info); err != nil {
		return 0, err
	}
	return info.Id, nil
}

// ReplayData loads a stored replay.
func (m *Manager) ReplayData(ctx context.Context, id int64) (*store.ReplayInfo, []byte, error) {
	info, err := m.st.Replay(ctx, id)
	if err != nil {
		return nil, nil, ErrNotFound
	}
	data, err := os.ReadFile(filepath.Join(m.conf.ReplayDir, filepath.Base(info.File)))
	return info, data, err
}

// Tournament lifecycle

func (m *Manager) Create(ctx context.Context, name, game, system string, by int64) (*store.Tournament, error) {
	rule := tourney.Find(game)
	if rule == nil {
		return nil, ErrUnknownGame
	}
	t := &store.Tournament{Name: name, Game: game, System: system, Rule: rule.Key, Status: TRegistering, CreatedBy: &by}
	return t, m.st.CreateTournament(ctx, t)
}

func (m *Manager) Join(ctx context.Context, tid, uid int64) error {
	t, err := m.st.Tournament(ctx, tid)
	if err != nil {
		return ErrNotFound
	}
	if t.Status != TRegistering {
		return ErrState
	}
	if err := m.st.AddTournamentPlayer(ctx, tid, uid); errors.Is(err, store.ErrDuplicate) {
		return ErrAlreadyJoind
	} else if err != nil {
		return err
	}
	go m.rooms.Notify(nil, "tournaments", "", 0)
	return nil
}

func (m *Manager) Leave(ctx context.Context, tid, uid int64) error {
	t, err := m.st.Tournament(ctx, tid)
	if err != nil {
		return ErrNotFound
	}
	if t.Status != TRegistering {
		return ErrState
	}
	if err := m.st.RemoveTournamentPlayer(ctx, tid, uid); err != nil {
		return err
	}
	go m.rooms.Notify(nil, "tournaments", "", 0)
	return nil
}

// Begin draws the bracket and queues the first round.
func (m *Manager) Begin(ctx context.Context, tid int64) error {
	m.mu.Lock()
	t, err := m.st.Tournament(ctx, tid)
	if err != nil {
		m.mu.Unlock()
		return ErrNotFound
	}
	if t.Status != TRegistering {
		m.mu.Unlock()
		return ErrState
	}
	players, err := m.st.TournamentPlayers(ctx, tid)
	if err != nil {
		m.mu.Unlock()
		return err
	}
	if len(players) < 2 {
		m.mu.Unlock()
		return ErrFewPlayers
	}
	ids := make([]int64, len(players))
	for i, p := range players {
		ids[i] = p.UserId
	}
	matches := Build(Shuffle(ids))
	if err := m.st.CreateBracket(ctx, tid, matches); err != nil {
		m.mu.Unlock()
		return err
	}
	var ns []notice
	for _, mt := range matches {
		switch {
		case mt.Status == StatusReady:
			m.queue = append(m.queue, mt.Id)
		case mt.Round == 1 && mt.Status == StatusFinished: // bye
			ns = append(ns, m.advanceLocked(ctx, mt)...)
		}
	}
	m.mu.Unlock()
	m.log.Info().Msgf("tournament %d started with %d players", tid, len(players))
	m.flush(append(ns, notice{kind: "tournaments", text: t.Name}))
	m.Kick()
	return nil
}

// Cancel stops a tournament and frees its servers.
func (m *Manager) Cancel(ctx context.Context, tid int64) error {
	m.mu.Lock()
	t, err := m.st.Tournament(ctx, tid)
	if err != nil {
		m.mu.Unlock()
		return ErrNotFound
	}
	if t.Status == TFinished || t.Status == TCanceled {
		m.mu.Unlock()
		return ErrState
	}
	_ = m.st.SetTournamentStatus(ctx, tid, TCanceled, nil)
	var ended []int64
	for id, a := range m.active {
		if a.assign.Tournament == tid {
			delete(m.active, id)
			m.sem.Release()
			ended = append(ended, id)
		}
	}
	all, _ := m.st.Matches(ctx, tid)
	for i := range all {
		mt := &all[i]
		m.queue = slices.DeleteFunc(m.queue, func(x int64) bool { return x == mt.Id })
		if mt.Status != StatusFinished {
			mt.Status = StatusStalled
			_ = m.st.UpdateMatch(ctx, mt)
		}
	}
	m.mu.Unlock()
	for _, id := range ended {
		m.rooms.EndMatch(id, "O torneio foi cancelado.")
	}
	m.rooms.Notify(nil, "tournaments", "", 0)
	m.Kick()
	return nil
}

// Snapshot returns the scheduler state (for the admin and the UI).
func (m *Manager) Snapshot() (running []int64, queued []int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for id := range m.active {
		running = append(running, id)
	}
	slices.Sort(running)
	return running, append([]int64(nil), m.queue...)
}
