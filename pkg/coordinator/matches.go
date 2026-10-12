package coordinator

import (
	"context"
	"fmt"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/tournament"
	"github.com/giongto35/cloud-game/v3/pkg/tourney"
)

// Tournament matches and replays run on servers reserved for them ("special"
// rooms): only the two participants play (P1 = slot 0, P2 = slot 1), everyone
// else watches, and nobody can save, load, reset or move between slots, so the
// recorded input replays stay deterministic.
//
// Lock order: tournament.Manager.mu -> Hub.mu. The hub never calls the
// manager while holding Hub.mu.

const (
	specialMatch  = "match"
	specialReplay = "replay"

	matchEndDelay    = 8 * time.Second  // the result stays on screen before the room closes
	replayIdleExpiry = 90 * time.Second // an unused replay reservation is dropped
)

type special struct {
	kind   string
	assign tournament.Assignment // matches
	replay int64                 // replays
	game   string
	room   string // room id once the game is running
	since  time.Time
	ending bool
}

// specialOfLocked returns the reservation of a worker (Hub.mu held).
func (h *Hub) specialOfLocked(w *Worker) *special { return h.special[w] }

func (h *Hub) specialWorker(kind string, id int64) *Worker {
	h.mu.Lock()
	defer h.mu.Unlock()
	for w, sp := range h.special {
		if sp.kind == kind && ((kind == specialMatch && sp.assign.Match == id) || (kind == specialReplay && sp.replay == id)) {
			return w
		}
	}
	return nil
}

// freeWorkerLocked returns a free worker with no reservation of any kind.
func (h *Hub) freeWorkerLocked(workers []*Worker) *Worker {
	for _, w := range workers {
		if w.HasSlot() && h.reservationLocked(w) == nil && h.special[w] == nil {
			return w
		}
	}
	return nil
}

// waitSpecialRoom waits until the reserved room of a worker is running.
func (h *Hub) waitSpecialRoom(w *Worker, max time.Duration) string {
	for deadline := time.Now().Add(max); ; {
		h.mu.Lock()
		sp := h.special[w]
		room := ""
		if sp != nil {
			room = sp.room
		}
		h.mu.Unlock()
		if sp == nil || (room != "" && w.RoomId == room) || time.Now().After(deadline) {
			return room
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// tournament.Rooms implementation

func (h *Hub) ReserveMatch(a tournament.Assignment) bool {
	workers := h.workerList()
	h.mu.Lock()
	w := h.freeWorkerLocked(workers)
	if w != nil {
		h.special[w] = &special{kind: specialMatch, assign: a, game: a.Game, since: time.Now()}
	}
	h.mu.Unlock()
	if w == nil {
		return false
	}
	h.log.Info().Str("worker", w.Id().Short()).Msgf("match %d reserved", a.Match)
	h.scheduleSync()
	return true
}

func (h *Hub) EndMatch(match int64, text string) {
	w := h.specialWorker(specialMatch, match)
	if w == nil {
		return
	}
	h.mu.Lock()
	sp := h.special[w]
	rid := sp.room
	sp.ending = true
	if rid == "" || w.RoomId != rid { // never started: just free the server
		delete(h.special, w)
	}
	h.mu.Unlock()
	if rid == "" || w.RoomId != rid {
		h.dispatch()
		return
	}
	h.systemChat(rid, text)
	for _, u := range h.userList() {
		if u.w == w {
			u.Notify(api.Notice, api.NoticeResponse{Kind: "match_over", Text: text, Match: match})
		}
	}
	// players and viewers go back to the tournament page (and reconnect at once,
	// so the next call reaches them); stale connections are dropped afterwards
	time.AfterFunc(matchEndDelay, func() {
		for _, u := range h.userList() {
			if u.w == w {
				u.Notify(api.Notice, api.NoticeResponse{Kind: "match_closed", Text: text, Match: match})
			}
		}
		time.AfterFunc(5*time.Second, func() {
			for _, u := range h.userList() {
				if u.w == w && h.roomOf(u) == rid {
					h.Kick(u, text)
				}
			}
		})
	})
}

func (h *Hub) Present(match int64) []int64 {
	w := h.specialWorker(specialMatch, match)
	if w == nil {
		return nil
	}
	var out []int64
	for _, u := range h.userList() {
		if u.w == w && !slices.Contains(out, u.Account().Id) {
			out = append(out, u.Account().Id)
		}
	}
	return out
}

func (h *Hub) Notify(users []int64, kind, text string, match int64) {
	n := api.NoticeResponse{Kind: kind, Text: text, Match: match}
	for _, u := range h.userList() {
		if users == nil || slices.Contains(users, u.Account().Id) {
			u.Notify(api.Notice, n)
		}
	}
}

// ReserveReplay reserves a server to play back a replay (or reuses the running room).
func (h *Hub) ReserveReplay(id int64, game string) bool {
	if h.specialWorker(specialReplay, id) != nil {
		return true
	}
	workers := h.workerList()
	h.mu.Lock()
	w := h.freeWorkerLocked(workers)
	if w != nil {
		h.special[w] = &special{kind: specialReplay, replay: id, game: game, since: time.Now()}
	}
	h.mu.Unlock()
	if w != nil {
		h.scheduleSync()
	}
	return w != nil
}

// releaseSpecial drops the reservation of a worker whose room closed; a match
// that ended without a result goes back to the tournament queue.
func (h *Hub) releaseSpecial(w *Worker, rid string) {
	h.mu.Lock()
	sp := h.special[w]
	if sp == nil || (rid != "" && sp.room != "" && sp.room != rid) {
		h.mu.Unlock()
		return
	}
	delete(h.special, w)
	h.mu.Unlock()
	if sp.kind == specialMatch && !sp.ending && h.tm != nil {
		h.tm.RoomLost(sp.assign.Match)
	}
}

// expireSpecials drops replay reservations nobody used.
func (h *Hub) expireSpecials(now time.Time) {
	h.mu.Lock()
	for w, sp := range h.special {
		if sp.kind == specialReplay && sp.room == "" && now.Sub(sp.since) > replayIdleExpiry {
			delete(h.special, w)
		}
	}
	h.mu.Unlock()
}

// onMatchResult receives the result detected by the worker.
func (h *Hub) onMatchResult(w *Worker, rq api.MatchResultRequest) {
	h.mu.Lock()
	sp := h.special[w]
	ok := sp != nil && sp.kind == specialMatch && sp.assign.Match == rq.Match && sp.room == rq.Room
	h.mu.Unlock()
	if !ok || h.tm == nil {
		h.log.Warn().Msgf("unexpected match result %d from %s", rq.Match, w.Id().Short())
		return
	}
	h.log.Info().Msgf("match %d: P%d wins %v (frame %d, replay %d bytes)", rq.Match, rq.Winner+1, rq.Score, rq.Frame, len(rq.Replay))
	if err := h.tm.Result(rq.Match, rq.Winner, rq.Score, rq.Replay); err != nil {
		h.log.Error().Err(err).Msgf("match %d result rejected", rq.Match)
	}
}

// startSpecial starts or joins a tournament match or a replay room.
func (u *User) startSpecial(rq api.GameStartUserRequest, h *Hub) {
	acc := u.Account()
	h.mu.Lock()
	sp := h.special[u.w]
	ok := sp != nil && !sp.ending &&
		((rq.MatchId != 0 && sp.kind == specialMatch && sp.assign.Match == rq.MatchId) ||
			(rq.ReplayId != 0 && sp.kind == specialReplay && sp.replay == rq.ReplayId))
	slot := SpectatorSlot
	var assign tournament.Assignment
	var room, game string
	if ok {
		assign, room, game = sp.assign, sp.room, sp.game
		if sp.kind == specialMatch {
			for i, p := range assign.Players {
				if p == acc.Id {
					slot = i
				}
			}
		}
	}
	h.mu.Unlock()
	if !ok {
		u.notice("room_gone", "Esta partida não está mais disponível.")
		return
	}

	created := false
	var resp *api.StartGameResponse
	var err error
	if u.w.HasSlot() {
		// the room isn't running yet: a participant (or any replay viewer) starts it
		if rq.MatchId != 0 && slot < 0 {
			u.notice("room_gone", "A partida ainda não começou. Tente assistir daqui a pouco.")
			return
		}
		if !u.w.TryReserve() {
			u.Notify(api.ErrNoFreeSlots, "")
			return
		}
		var tr *api.TournamentRoom
		var replay []byte
		if rq.MatchId != 0 {
			room = fmt.Sprintf("m%d%04x___%s", rq.MatchId, rand.IntN(0x10000), game)
			tr = &api.TournamentRoom{Match: rq.MatchId, Rule: assign.Rule}
		} else {
			room = fmt.Sprintf("r%d%04x___%s", rq.ReplayId, rand.IntN(0x10000), game)
			_, data, rerr := h.tm.ReplayData(context.Background(), rq.ReplayId)
			if rerr != nil {
				u.w.FreeSlots()
				u.notice("error", "Não foi possível abrir este replay.")
				return
			}
			replay = data
		}
		resp, err = u.w.StartSpecial(u.Id().String(), api.GameStartUserRequest{GameName: game, RoomId: room, PlayerIndex: slot}, tr, replay)
		if err != nil || resp == nil || resp.Rid == "" {
			u.w.FreeSlots()
			u.log.Error().Err(err).Msg("special room start failed")
			u.notice("error", "Não foi possível iniciar a partida. Tente novamente.")
			return
		}
		room, created = resp.Rid, true
		h.mu.Lock()
		if cur := h.special[u.w]; cur != nil {
			cur.room = room
		}
		h.mu.Unlock()
		if rq.MatchId != 0 {
			h.tm.MatchStarted(rq.MatchId, room)
		}
	} else {
		// the other player may be starting the room right now: wait for it
		room = h.waitSpecialRoom(u.w, 20*time.Second)
		if room == "" || u.w.RoomId != room {
			u.Notify(api.ErrNoFreeSlots, "")
			return
		}
		resp, err = u.w.StartGame(u.Id().String(), api.GameStartUserRequest{GameName: game, RoomId: room, PlayerIndex: slot})
		if err != nil || resp == nil || resp.Rid == "" {
			u.notice("error", "Não foi possível entrar na partida.")
			return
		}
	}

	h.mu.Lock()
	h.metaLocked(room).locked = true
	h.mu.Unlock()
	h.joinRoom(u, room, slot, created && rq.MatchId != 0)
	u.StartGame(room, resp.AV, resp.KbMouse, slot, false)
	if rq.MatchId != 0 && slot >= 0 {
		if r := tourney.ByKey(assign.Rule); r != nil {
			u.notice("match_rules", r.Hint)
		}
	}
}
