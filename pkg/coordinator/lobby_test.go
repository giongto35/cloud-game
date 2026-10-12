package coordinator

import (
	"sync"
	"testing"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/auth"
	"github.com/giongto35/cloud-game/v3/pkg/com"
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/store"
)

// fakeConn is an in-memory Connection that records notifications.
type fakeConn struct {
	id  com.Uid
	mu  sync.Mutex
	out []api.PT
}

func (f *fakeConn) Disconnect()                                              {}
func (f *fakeConn) Id() com.Uid                                              { return f.id }
func (f *fakeConn) ProcessPackets(func(api.In[com.Uid]) error) chan struct{} { return nil }
func (f *fakeConn) Send(api.PT, any) ([]byte, error)                         { return nil, nil }
func (f *fakeConn) Notify(t api.PT, _ any) {
	f.mu.Lock()
	f.out = append(f.out, t)
	f.mu.Unlock()
}

func newTestHub(t *testing.T) *Hub {
	t.Helper()
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	svc := auth.NewService(st, config.Auth{AdminEmail: "wenderson@belko.com.br", AdminPassword: "x-admin-pass"}, logger.Default())
	var conf config.CoordinatorConfig
	conf.Coordinator.Auth.QueueReserveSec = 30
	h := NewHub(conf, svc, logger.Default())
	t.Cleanup(h.Stop)
	return h
}

func newTestUser(h *Hub, id int64, name string) *User {
	u := &User{Connection: &fakeConn{id: com.NewUid()}, slot: SpectatorSlot, log: logger.Default()}
	u.SetAccount(&auth.Account{Id: id, Name: name, Status: store.StatusApproved,
		Perms: auth.PermsFor(store.RoleUser, false)})
	h.users.Add(u)
	return u
}

func newTestWorker(h *Hub) *Worker {
	w := &Worker{Connection: &fakeConn{id: com.NewUid()}, log: logger.Default()}
	h.workers.Add(w)
	return w
}

func queuePos(h *Hub, acc int64) (int, *queueEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.entryOfLocked(acc)
}

func TestQueueReservation(t *testing.T) {
	h := newTestHub(t)
	a := newTestUser(h, 1, "Ana")
	b := newTestUser(h, 2, "Bruno")

	h.QueueJoin(a, "Sushi The Cat")
	h.QueueJoin(b, "anguna")
	if i, e := queuePos(h, 1); i != 0 || e.called != nil {
		t.Fatalf("Ana should wait first, got %d %+v", i, e)
	}
	if i, _ := queuePos(h, 2); i != 1 {
		t.Fatalf("Bruno should be second, got %d", i)
	}

	w := newTestWorker(h)
	h.dispatch()

	if _, e := queuePos(h, 1); e == nil || e.called != w {
		t.Fatal("the free worker must be reserved for Ana")
	}
	if h.isAvailableFor(w, 2) {
		t.Fatal("a reserved worker is not available to others")
	}
	if !h.isAvailableFor(w, 1) || h.reservedWorkerFor(1) != w {
		t.Fatal("a reserved worker is available to its owner")
	}
	if h.find1stFreeWorker("", 2) != nil {
		t.Fatal("Bruno must not get Ana's reserved worker")
	}

	b.w = w
	if h.claimWorker(b) {
		t.Fatal("Bruno must not claim Ana's reservation")
	}
	a.w = w
	if !h.claimWorker(a) {
		t.Fatal("Ana should claim her reservation")
	}
	if _, e := queuePos(h, 1); e != nil {
		t.Fatal("Ana must leave the queue once served")
	}
	if i, _ := queuePos(h, 2); i != 0 {
		t.Fatalf("Bruno should be first now, got %d", i)
	}
}

func TestQueueExpiryAndAbsence(t *testing.T) {
	h := newTestHub(t)
	a := newTestUser(h, 1, "Ana")
	b := newTestUser(h, 2, "Bruno")
	h.QueueJoin(a, "game")
	h.QueueJoin(b, "game")
	newTestWorker(h)
	h.dispatch()

	// Ana does not answer: after the deadline Bruno is called
	h.lobbyTick(time.Now().Add(31 * time.Second))
	if _, e := queuePos(h, 1); e != nil {
		t.Fatal("expired entry must be removed")
	}
	if _, e := queuePos(h, 2); e == nil || e.called == nil {
		t.Fatal("Bruno should be called after Ana's reservation expired")
	}

	// Bruno closes the tab and does not come back
	h.users.Remove(b)
	h.detach(b)
	h.mu.Lock()
	_, e := h.entryOfLocked(2)
	gone := !e.goneAt.IsZero()
	h.mu.Unlock()
	if !gone {
		t.Fatal("absent account should be marked")
	}
	h.lobbyTick(time.Now().Add(time.Minute))
	if _, e := queuePos(h, 2); e != nil {
		t.Fatal("absent account must be removed after the grace period")
	}
}

func TestQueueSurvivesReconnect(t *testing.T) {
	h := newTestHub(t)
	a := newTestUser(h, 1, "Ana")
	h.QueueJoin(a, "game")
	h.users.Remove(a)
	h.detach(a)
	a2 := newTestUser(h, 1, "Ana")
	h.attach(a2)
	h.lobbyTick(time.Now().Add(time.Minute))
	if i, _ := queuePos(h, 1); i != 0 {
		t.Fatal("reconnected account keeps its position")
	}
}

func TestRoomSlotsAndHost(t *testing.T) {
	h := newTestHub(t)
	const rid = "abc___game"
	host := newTestUser(h, 1, "Ana")
	p2 := newTestUser(h, 2, "Bruno")
	viewer := newTestUser(h, 3, "Carla")

	h.joinRoom(host, rid, 0, true)
	if !h.isHost(host) || !h.canControl(host) {
		t.Fatal("creator is the host")
	}
	if s := h.pickSlot(rid, 0, false); s != 1 {
		t.Fatalf("taken slot must be skipped, got %d", s)
	}
	h.joinRoom(p2, rid, h.pickSlot(rid, 3, false), false)
	if p2.slot != 3 {
		t.Fatalf("free requested slot should be granted, got %d", p2.slot)
	}
	h.joinRoom(viewer, rid, h.pickSlot(rid, 0, true), false)
	if viewer.slot != SpectatorSlot {
		t.Fatal("spectators have no slot")
	}
	if h.canControl(p2) || h.canControl(viewer) {
		t.Fatal("only the host controls the room")
	}
	if _, ok := h.setSlot(viewer, 3); ok {
		t.Fatal("cannot take an occupied slot")
	}
	if _, ok := h.setSlot(viewer, 2); !ok || viewer.slot != 2 {
		t.Fatal("can take a free slot")
	}
	if _, ok := h.setSlot(viewer, 7); ok {
		t.Fatal("invalid slot must be rejected")
	}

	// host leaves: the oldest player becomes the host
	h.leaveRoom(host, "saiu da sala")
	if !h.isHost(p2) {
		t.Fatal("the next player should become the host")
	}

	// full room: everyone else watches
	h.joinRoom(newTestUser(h, 4, "Davi"), rid, h.pickSlot(rid, -1, false), false)
	h.joinRoom(newTestUser(h, 5, "Eva"), rid, h.pickSlot(rid, -1, false), false)
	if s := h.pickSlot(rid, -1, false); s != SpectatorSlot {
		t.Fatalf("a full room only accepts spectators, got %d", s)
	}

	h.onRoomClosed(rid)
	if h.roomOf(p2) != "" || h.isHost(p2) {
		t.Fatal("closing the room clears memberships")
	}
}

func TestRoomEndsWithoutPlayers(t *testing.T) {
	h := newTestHub(t)
	const rid = "abc___game"
	player := newTestUser(h, 1, "Ana")
	viewer := newTestUser(h, 2, "Bruno")
	h.joinRoom(player, rid, 0, true)
	h.joinRoom(viewer, rid, SpectatorSlot, false)
	h.leaveRoom(player, "saiu da sala")
	if h.roomOf(viewer) != "" {
		t.Fatal("spectators must leave a room without players")
	}
	got := false
	fc := viewer.Connection.(*fakeConn)
	fc.mu.Lock()
	for _, p := range fc.out {
		if p == api.Notice {
			got = true
		}
	}
	fc.mu.Unlock()
	if !got {
		t.Fatal("spectators must be told that the match is over")
	}
}

func TestLobbyState(t *testing.T) {
	h := newTestHub(t)
	w := newTestWorker(h)
	w.RoomId = "abc___Sushi The Cat"
	w.TryReserve()
	w.SetLib([]api.GameInfo{{Name: "Sushi The Cat", System: "gba"}})
	newTestWorker(h)
	u := newTestUser(h, 1, "Ana")
	h.joinRoom(u, w.RoomId, 0, true)

	users := h.userList()
	h.mu.Lock()
	st := h.lobbyStateLocked(users, []*Worker{w})
	h.mu.Unlock()
	if len(st.Rooms) != 1 || st.Rooms[0].Game != "Sushi The Cat" || st.Rooms[0].System != "gba" ||
		st.Rooms[0].Host != "Ana" || st.Rooms[0].SlotsFree != 3 {
		t.Fatalf("unexpected lobby: %+v", st)
	}
}

func TestChatSanitize(t *testing.T) {
	if got := sanitizeChat("  oi\x00 pessoal\n "); got != "oi  pessoal" {
		t.Fatalf("got %q", got)
	}
	long := make([]rune, 400)
	for i := range long {
		long[i] = 'á'
	}
	if got := []rune(sanitizeChat(string(long))); len(got) != chatMaxLen {
		t.Fatalf("expected %d runes, got %d", chatMaxLen, len(got))
	}
}
