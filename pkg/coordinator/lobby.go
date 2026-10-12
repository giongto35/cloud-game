package coordinator

import (
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/auth"
	"github.com/giongto35/cloud-game/v3/pkg/games"
)

// Lobby: the waiting queue, worker reservations, room membership (roles) and chat.
//
// Locking rules: all fields below are guarded by Hub.mu, as are the room/slot
// fields of every User. Never hold Hub.mu while iterating the users/workers maps
// (take snapshots first) and never send network messages while holding it.

const (
	MaxPlayers       = 4
	SpectatorSlot    = -1
	queueGoneGrace   = 30 * time.Second
	chatMinInterval  = 500 * time.Millisecond
	chatMaxLen       = 300
	syncDebounce     = 150 * time.Millisecond
	lobbyTickEvery   = time.Second
	defaultReserveIn = 30 * time.Second
)

type queueEntry struct {
	acc    int64
	name   string
	game   string
	joined time.Time
	called *Worker   // the worker reserved for this entry
	until  time.Time // reservation deadline
	goneAt time.Time // when the account lost all its connections
}

type roomMeta struct {
	host    int64 // account id of the host
	created time.Time
	banned  map[int64]bool
	locked  bool // tournament/replay room: fixed slots, no save/load/reset
}

// snapshots (no locks held while iterating)

func (h *Hub) userList() []*User {
	var out []*User
	for u := range h.users.Values() {
		out = append(out, u)
	}
	return out
}

func (h *Hub) workerList() []*Worker {
	var out []*Worker
	for w := range h.workers.Values() {
		out = append(out, w)
	}
	return out
}

func (h *Hub) reserveFor() time.Duration {
	if s := h.conf.Coordinator.Auth.QueueReserveSec; s > 0 {
		return time.Duration(s) * time.Second
	}
	return defaultReserveIn
}

// queue helpers (Hub.mu must be held)

func (h *Hub) entryOfLocked(acc int64) (int, *queueEntry) {
	for i, e := range h.queue {
		if e.acc == acc {
			return i, e
		}
	}
	return -1, nil
}

func (h *Hub) reservationLocked(w *Worker) *queueEntry {
	for _, e := range h.queue {
		if e.called == w {
			return e
		}
	}
	return nil
}

func (h *Hub) removeEntryLocked(acc int64) bool {
	i, _ := h.entryOfLocked(acc)
	if i < 0 {
		return false
	}
	h.queue = slices.Delete(h.queue, i, i+1)
	return true
}

// dispatchLocked hands free (unreserved) workers to the waiting accounts in order.
func (h *Hub) dispatchLocked(workers []*Worker) bool {
	changed := false
	now := time.Now()
	for _, w := range workers {
		if !w.HasSlot() || h.reservationLocked(w) != nil || h.special[w] != nil {
			continue
		}
		for _, e := range h.queue {
			if e.called == nil && e.goneAt.IsZero() {
				e.called, e.until = w, now.Add(h.reserveFor())
				changed = true
				h.log.Info().Int64("acc", e.acc).Str("worker", w.Id().Short()).Msg("queue: server reserved")
				break
			}
		}
	}
	return changed
}

// isAvailableFor reports whether the worker can start a new game for the account.
func (h *Hub) isAvailableFor(w *Worker, acc int64) bool {
	if !w.HasSlot() {
		return false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.special[w] != nil {
		return false
	}
	e := h.reservationLocked(w)
	return e == nil || e.acc == acc
}

// reservedWorkerFor returns a worker reserved for the account, if any.
func (h *Hub) reservedWorkerFor(acc int64) *Worker {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, e := h.entryOfLocked(acc); e != nil && e.called != nil && e.called.HasSlot() {
		return e.called
	}
	return nil
}

// claimWorker reserves the worker slot for a new game of the user.
// It respects queue reservations and removes the user's queue entry when served.
func (h *Hub) claimWorker(u *User) bool {
	h.mu.Lock()
	e := h.reservationLocked(u.w)
	if e != nil && e.acc != u.Account().Id {
		h.mu.Unlock()
		return false
	}
	if !u.w.TryReserve() {
		h.mu.Unlock()
		return false
	}
	served := h.removeEntryLocked(u.Account().Id)
	h.mu.Unlock()
	if served || e != nil {
		h.scheduleSync()
	}
	return true
}

func (h *Hub) QueueJoin(u *User, game string) {
	game = strings.TrimSpace(game)
	if game == "" || len(game) > 256 {
		return
	}
	h.mu.Lock()
	if _, e := h.entryOfLocked(u.Account().Id); e != nil {
		e.game = game
	} else {
		h.queue = append(h.queue, &queueEntry{acc: u.Account().Id, name: u.Account().Name, game: game, joined: time.Now()})
		h.log.Info().Int64("acc", u.Account().Id).Str("game", game).Msgf("queue: joined (%d)", len(h.queue))
	}
	h.mu.Unlock()
	h.dispatch()
}

func (h *Hub) QueueLeave(u *User) {
	h.mu.Lock()
	removed := h.removeEntryLocked(u.Account().Id)
	h.mu.Unlock()
	if removed {
		h.dispatch()
	}
}

func (h *Hub) dispatch() {
	// tournament matches get the free servers first
	if h.tm != nil {
		h.tm.Kick()
	}
	workers := h.workerList()
	h.mu.Lock()
	h.dispatchLocked(workers)
	h.mu.Unlock()
	h.scheduleSync()
}

// attach marks the account as present again (keeps its queue position across reloads).
func (h *Hub) attach(u *User) {
	h.mu.Lock()
	if _, e := h.entryOfLocked(u.Account().Id); e != nil {
		e.goneAt = time.Time{}
	}
	h.mu.Unlock()
}

// detach is called after a user connection is closed.
func (h *Hub) detach(u *User) {
	others := false
	for _, x := range h.userList() {
		if x != u && x.Account().Id == u.Account().Id {
			others = true
			break
		}
	}
	h.leaveRoom(u, "saiu da sala")
	if !others {
		h.mu.Lock()
		if _, e := h.entryOfLocked(u.Account().Id); e != nil {
			e.goneAt = time.Now()
		}
		h.mu.Unlock()
	}
	h.scheduleSync()
}

// lobbyLoop expires reservations and absent accounts.
func (h *Hub) lobbyLoop() {
	t := time.NewTicker(lobbyTickEvery)
	defer t.Stop()
	for {
		select {
		case <-h.done:
			return
		case <-t.C:
		}
		h.lobbyTick(time.Now())
		h.expireSpecials(time.Now())
	}
}

func (h *Hub) lobbyTick(now time.Time) {
	workers := h.workerList()
	alive := map[*Worker]bool{}
	for _, w := range workers {
		alive[w] = true
	}
	var expired []int64
	changed := false
	h.mu.Lock()
	kept := h.queue[:0]
	for _, e := range h.queue {
		switch {
		case e.called != nil && now.After(e.until):
			expired = append(expired, e.acc)
			changed = true
			continue
		case !e.goneAt.IsZero() && now.Sub(e.goneAt) > queueGoneGrace:
			changed = true
			continue
		case e.called != nil && !alive[e.called]:
			e.called = nil
			changed = true
		}
		kept = append(kept, e)
	}
	clear(h.queue[len(kept):])
	h.queue = kept
	if h.dispatchLocked(workers) {
		changed = true
	}
	h.mu.Unlock()

	for _, acc := range expired {
		h.log.Info().Int64("acc", acc).Msg("queue: reservation expired")
		for _, u := range h.userList() {
			if u.Account().Id == acc {
				u.Notify(api.QueueState, api.QueueStateResponse{Status: "expired"})
			}
		}
	}
	if changed {
		h.scheduleSync()
	}
}

// Rooms

func (h *Hub) metaLocked(rid string) *roomMeta {
	m := h.rooms[rid]
	if m == nil {
		m = &roomMeta{created: time.Now(), banned: map[int64]bool{}}
		h.rooms[rid] = m
	}
	return m
}

func (h *Hub) membersLocked(rid string, users []*User) []*User {
	var out []*User
	for _, u := range users {
		if u.room == rid && rid != "" {
			out = append(out, u)
		}
	}
	slices.SortFunc(out, func(a, b *User) int { return a.joined.Compare(b.joined) })
	return out
}

func slotTaken(members []*User, slot int, except *User) bool {
	for _, m := range members {
		if m != except && m.slot == slot {
			return true
		}
	}
	return false
}

// pickSlot chooses a slot for a user joining an existing room.
func (h *Hub) pickSlot(rid string, want int, spectator bool) int {
	if spectator {
		return SpectatorSlot
	}
	users := h.userList()
	h.mu.Lock()
	defer h.mu.Unlock()
	members := h.membersLocked(rid, users)
	if want >= 0 && want < MaxPlayers && !slotTaken(members, want, nil) {
		return want
	}
	for s := range MaxPlayers {
		if !slotTaken(members, s, nil) {
			return s
		}
	}
	return SpectatorSlot
}

func (h *Hub) isBanned(rid string, acc int64) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.rooms[rid]
	return m != nil && m.banned[acc]
}

func (h *Hub) joinRoom(u *User, rid string, slot int, created bool) {
	h.mu.Lock()
	m := h.metaLocked(rid)
	if created || m.host == 0 {
		m.host = u.Account().Id
	}
	u.room, u.slot, u.joined = rid, slot, time.Now()
	h.mu.Unlock()
	if slot == SpectatorSlot {
		h.systemChat(rid, u.Account().Name+" está assistindo")
	} else {
		h.systemChat(rid, u.Account().Name+" entrou como Jogador "+itoa(slot+1))
	}
	h.scheduleSync()
}

// leaveRoom removes the user from its room and passes the host role if needed.
func (h *Hub) leaveRoom(u *User, why string) {
	users := h.userList()
	h.mu.Lock()
	rid := u.room
	if rid == "" {
		h.mu.Unlock()
		return
	}
	u.room, u.slot = "", SpectatorSlot
	var newHost *User
	if m := h.rooms[rid]; m != nil && m.host == u.Account().Id {
		members := h.membersLocked(rid, users)
		stillHere := false
		for _, x := range members {
			if x.Account().Id == u.Account().Id {
				stillHere = true
			}
		}
		if !stillHere {
			m.host = 0
			// prefer the oldest player, then the oldest spectator
			for _, x := range members {
				if x.slot >= 0 {
					newHost = x
					break
				}
			}
			if newHost == nil && len(members) > 0 {
				newHost = members[0]
			}
			if newHost != nil {
				m.host = newHost.Account().Id
			}
		}
	}
	// a room without players ends: spectators alone don't keep a server busy
	var orphans []*User
	if members := h.membersLocked(rid, users); len(members) > 0 {
		players := 0
		for _, x := range members {
			if x.slot >= 0 {
				players++
			}
		}
		if players == 0 {
			orphans = members
			for _, x := range members {
				x.room, x.slot = "", SpectatorSlot
			}
			newHost = nil
		}
	}
	h.mu.Unlock()
	if len(orphans) > 0 {
		for _, x := range orphans {
			x.notice("room_gone", "A partida terminou: todos os jogadores saíram.")
			if x.w != nil {
				x.w.TerminateSession(x.Id().String())
			}
		}
		h.scheduleSync()
		return
	}
	h.systemChat(rid, u.Account().Name+" "+why)
	if newHost != nil {
		h.systemChat(rid, newHost.Account().Name+" agora é o anfitrião")
	}
	h.scheduleSync()
}

// onRoomClosed is called when a worker reports that its room was closed.
func (h *Hub) onRoomClosed(rid string) {
	users := h.userList()
	h.mu.Lock()
	delete(h.rooms, rid)
	for _, u := range users {
		if u.room == rid {
			u.room, u.slot = "", SpectatorSlot
		}
	}
	h.mu.Unlock()
	h.dispatch()
}

func (h *Hub) isHost(u *User) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.rooms[u.room]
	return u.room != "" && m != nil && m.host == u.Account().Id
}

// canControl reports whether the user may save, load, reset or manage the room.
func (h *Hub) canControl(u *User) bool {
	if h.isLocked(u) {
		return false
	}
	if u.Account().Can(auth.PermRoomsModerate) {
		h.mu.Lock()
		defer h.mu.Unlock()
		return u.room != ""
	}
	return h.isHost(u)
}

// isLocked reports whether the user is in a tournament or replay room.
func (h *Hub) isLocked(u *User) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	m := h.rooms[u.room]
	return u.room != "" && m != nil && m.locked
}

// roomOf returns the room id the user is in.
func (h *Hub) roomOf(u *User) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return u.room
}

// setSlot validates and records a slot change; it returns the previous slot.
func (h *Hub) setSlot(target *User, slot int) (prev int, ok bool) {
	users := h.userList()
	h.mu.Lock()
	defer h.mu.Unlock()
	if target.room == "" || slot < SpectatorSlot || slot >= MaxPlayers {
		return 0, false
	}
	if m := h.rooms[target.room]; m != nil && m.locked {
		return 0, false
	}
	if slot >= 0 && slotTaken(h.membersLocked(target.room, users), slot, target) {
		return 0, false
	}
	prev, target.slot = target.slot, slot
	return prev, true
}

func (h *Hub) restoreSlot(target *User, slot int) {
	h.mu.Lock()
	target.slot = slot
	h.mu.Unlock()
}

func (h *Hub) ban(rid string, acc int64) {
	h.mu.Lock()
	h.metaLocked(rid).banned[acc] = true
	h.mu.Unlock()
}

// Kick disconnects a user with a reason shown to them.
func (h *Hub) Kick(u *User, reason string) {
	u.Notify(api.Kicked, api.KickedResponse{Reason: reason})
	time.AfterFunc(300*time.Millisecond, u.Disconnect)
}

// Chat

func sanitizeChat(text string) string {
	text = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, text)
	text = strings.TrimSpace(text)
	if utf8.RuneCountInString(text) > chatMaxLen {
		text = string([]rune(text)[:chatMaxLen])
	}
	return text
}

func (h *Hub) Chat(u *User, text string) {
	text = sanitizeChat(text)
	if text == "" {
		return
	}
	now := time.Now()
	h.mu.Lock()
	rid := u.room
	if rid == "" || now.Sub(u.lastChat) < chatMinInterval {
		h.mu.Unlock()
		return
	}
	u.lastChat = now
	h.mu.Unlock()
	h.sendRoom(rid, api.ChatMessageResponse{From: u.Id().String(), Name: u.Account().Name, Text: text, Ts: now.UnixMilli()})
}

func (h *Hub) systemChat(rid, text string) {
	if rid == "" {
		return
	}
	h.sendRoom(rid, api.ChatMessageResponse{Text: text, Ts: time.Now().UnixMilli(), System: true})
}

func (h *Hub) sendRoom(rid string, msg api.ChatMessageResponse) {
	users := h.userList()
	h.mu.Lock()
	members := h.membersLocked(rid, users)
	h.mu.Unlock()
	for _, m := range members {
		m.Notify(api.ChatMessage, msg)
	}
}

// State broadcasting

func (h *Hub) scheduleSync() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.syncPending {
		return
	}
	h.syncPending = true
	time.AfterFunc(syncDebounce, func() {
		h.mu.Lock()
		h.syncPending = false
		h.mu.Unlock()
		h.sync()
	})
}

// sync pushes the lobby, queue and room states to everyone.
func (h *Hub) sync() {
	users := h.userList()
	workers := h.workerList()
	now := time.Now()

	h.mu.Lock()
	lobby := h.lobbyStateLocked(users, workers)

	queueOf := map[int64]api.QueueStateResponse{}
	for i, e := range h.queue {
		q := api.QueueStateResponse{Status: "waiting", Position: i + 1, Size: len(h.queue), Game: e.game}
		if e.called != nil {
			q.Status = "called"
			q.ExpiresIn = max(0, int(e.until.Sub(now).Seconds()))
		}
		queueOf[e.acc] = q
	}

	rooms := map[string]api.RoomStateResponse{}
	memberRooms := map[*User]string{}
	for _, u := range users {
		if u.room == "" {
			continue
		}
		memberRooms[u] = u.room
		if _, ok := rooms[u.room]; ok {
			continue
		}
		m := h.rooms[u.room]
		st := api.RoomStateResponse{RoomId: u.room, Game: games.ExtractGame(u.room)}
		for _, x := range h.membersLocked(u.room, users) {
			st.Members = append(st.Members, api.RoomMember{
				Id: x.Id().String(), Account: x.Account().Id, Name: x.Account().Name, Slot: x.slot,
				Host: m != nil && m.host == x.Account().Id,
			})
		}
		rooms[u.room] = st
	}
	h.mu.Unlock()

	for _, u := range users {
		u.Notify(api.LobbyState, lobby)
		q, ok := queueOf[u.Account().Id]
		if !ok {
			q = api.QueueStateResponse{Status: "none"}
		}
		u.Notify(api.QueueState, q)
		if rid, ok := memberRooms[u]; ok {
			st := rooms[rid]
			members := make([]api.RoomMember, len(st.Members))
			copy(members, st.Members)
			for i := range members {
				members[i].You = members[i].Id == u.Id().String()
			}
			st.Members = members
			u.Notify(api.RoomState, st)
		}
	}
}

func systemOf(w *Worker, game string) string {
	for _, g := range w.AppNames() {
		if g.Name == game {
			return g.System
		}
	}
	return ""
}

func (h *Hub) lobbyStateLocked(users []*User, workers []*Worker) api.LobbyStateResponse {
	st := api.LobbyStateResponse{Rooms: []api.LobbyRoom{}, Queue: api.LobbyQueue{Size: len(h.queue)}}
	st.Workers.Total = len(workers)
	for _, w := range workers {
		if w.HasSlot() && h.reservationLocked(w) == nil && h.special[w] == nil {
			st.Workers.Free++
		}
		rid := w.RoomId
		if rid == "" {
			continue
		}
		game := games.ExtractGame(rid)
		room := api.LobbyRoom{Id: rid, Game: game, System: systemOf(w, game), Players: []api.LobbyMember{}}
		m := h.rooms[rid]
		if m != nil {
			room.Since = m.created.UnixMilli()
		}
		taken := 0
		for _, x := range h.membersLocked(rid, users) {
			if m != nil && m.host == x.Account().Id && room.Host == "" {
				room.Host = x.Account().Name
			}
			if x.slot >= 0 {
				taken++
				room.Players = append(room.Players, api.LobbyMember{Name: x.Account().Name, Slot: x.slot})
			} else {
				room.Spectators++
			}
		}
		room.SlotsFree = MaxPlayers - taken
		st.Rooms = append(st.Rooms, room)
	}
	slices.SortFunc(st.Rooms, func(a, b api.LobbyRoom) int { return int(a.Since - b.Since) })
	return st
}

// Notices

func (h *Hub) onPendingChanged(count int) {
	for _, u := range h.userList() {
		if u.Account().Can(auth.PermUsersApprove) {
			u.Notify(api.Notice, api.NoticeResponse{Kind: "pending_users", Count: count})
		}
	}
}

func itoa(i int) string { return strconv.Itoa(i) }
