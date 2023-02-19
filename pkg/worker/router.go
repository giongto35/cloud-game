package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/network/webrtc"
	"github.com/pion/webrtc/v3/pkg/media"
)

// Router tracks and routes freshly connected users to a game room.
// Basically, it holds user connection data until some user makes (connects to)
// a new room (game), then it manages all the cross-references between room and users.
// Rooms and users has 1-to-n relationship.
type Router struct {
	room  GamingRoom
	users com.NetMap[*Session]
}

// Session represents WebRTC connection of the user.
type Session struct {
	id   api.Uid
	conn *webrtc.Peer
	pi   int
	room GamingRoom // back reference
}

func NewRouter() Router { return Router{users: com.NewNetMap[*Session]()} }

func (r *Router) SetRoom(room GamingRoom) { r.room = room }
func (r *Router) AddUser(user *Session)   { r.users.Add(user) }
func (r *Router) Close() {
	if r.room != nil {
		r.room.Close()
	}
}
func (r *Router) GetRoom(id string) GamingRoom {
	if r.room != nil && r.room.GetId() == id {
		return r.room
	}
	return nil
}
func (r *Router) GetUser(uid api.Uid) *Session   { sess, _ := r.users.Find(uid.String()); return sess }
func (r *Router) RemoveRoom()                    { r.room = nil }
func (r *Router) RemoveDisconnect(user *Session) { r.users.Remove(user); user.Disconnect() }

func NewSession(rtc *webrtc.Peer, id api.Uid) *Session { return &Session{id: id, conn: rtc} }

func (s *Session) Disconnect()                          { s.conn.Disconnect() }
func (s *Session) GetPeerConn() *webrtc.Peer            { return s.conn }
func (s *Session) GetPlayerIndex() int                  { return s.pi }
func (s *Session) GetSetRoom(v GamingRoom) GamingRoom   { vv := s.room; s.room = v; return vv }
func (s *Session) Id() api.Uid                          { return s.id }
func (s *Session) IsConnected() bool                    { return s.conn.IsConnected() }
func (s *Session) SendAudio(sample *media.Sample) error { return s.conn.WriteAudio(sample) }
func (s *Session) SendVideo(sample *media.Sample) error { return s.conn.WriteVideo(sample) }
func (s *Session) SetPlayerIndex(index int)             { s.pi = index }
func (s *Session) SetRoom(room GamingRoom)              { s.room = room }
