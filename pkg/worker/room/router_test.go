package room

import (
	"testing"

	"github.com/giongto35/cloud-game/v3/pkg/com"
)

type tSession struct {
	id        string
	connected bool
}

func (t *tSession) SendAudio([]byte, int32) {}
func (t *tSession) SendVideo([]byte, int32) {}
func (t *tSession) SendData([]byte)         {}
func (t *tSession) Connect()                { t.connected = true }
func (t *tSession) Disconnect()             { t.connected = false }
func (t *tSession) Id() string              { return t.id }

type lookMap struct {
	com.NetMap[string, *tSession]
	prev com.NetMap[string, *tSession] // we could use pointers in the original :3
}

func (l *lookMap) Reset() {
	l.prev = com.NewNetMap[string, *tSession]()
	for s := range l.Map.Values() {
		l.prev.Add(s)
	}
	l.NetMap.Reset()
}

func TestRouter(t *testing.T) {
	router := newTestRouter()

	var r *Room[*tSession]

	router.SetRoom(&Room[*tSession]{id: "test001"})
	room := router.FindRoom("test001")
	if room == nil {
		t.Errorf("no room, but should be")
	}
	router.SetRoom(r)
	room = router.FindRoom("x")
	if room != nil {
		t.Errorf("a room, but should not be")
	}
	router.SetRoom(nil)
	router.Close()
}

func TestRouterReset(t *testing.T) {
	u := lookMap{NetMap: com.NewNetMap[string, *tSession]()}
	router := Router[*tSession]{users: &u}

	router.AddUser(&tSession{id: "1", connected: true})
	router.AddUser(&tSession{id: "2", connected: false})
	router.AddUser(&tSession{id: "3", connected: true})

	router.Reset()

	disconnected := true
	for u := range u.prev.Values() {
		disconnected = disconnected && !u.connected
	}
	if !disconnected {
		t.Errorf("not all users were disconnected, but should")
	}
	if !router.Users().Empty() {
		t.Errorf("has users after reset, but should not")
	}
}

func newTestRouter() *Router[*tSession] {
	u := com.NewNetMap[string, *tSession]()
	return &Router[*tSession]{users: &u}
}
