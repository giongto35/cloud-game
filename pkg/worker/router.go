package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	"github.com/pion/webrtc/v3/pkg/media"
)

type Router struct {
	room  *Room
	users com.NetMap[*Session]
}

// Session represents a user session.
// A WebRTC connection form a browser to the current server.
//
// todo rephrase
// It requires one connection to browser and one connection to the coordinator
// connection to browser is 1-1. connection to coordinator is n - 1
// Peerconnection can be from other server to ensure better latency
type Session struct {
	conn *webrtc.WebRTC
	id   network.Uid
	pi   int
	room *Room
}

func NewRouter() Router { return Router{users: com.NewNetMap[*Session]()} }

func (r *Router) SetRoom(room *Room)    { r.room = room }
func (r *Router) AddUser(user *Session) { r.users.Add(user) }
func (r *Router) Close() {
	if r.room != nil {
		r.room.Close()
	}
}
func (r *Router) GetUser(uid network.Uid) *Session {
	sess, _ := r.users.Find(string(uid))
	return sess
}
func (r *Router) RemoveRoom()              { r.room = nil }
func (r *Router) RemoveUser(user *Session) { r.users.Remove(user) }
func (r *Router) GetRoom(id string) *Room {
	if r.room != nil && r.room.id == id {
		return r.room
	}
	return nil
}

func NewSession(connection *webrtc.WebRTC, id network.Uid) *Session {
	return &Session{id: id, conn: connection}
}

func (s *Session) Id() network.Uid                     { return s.id }
func (s *Session) GetRoom() *Room                      { return s.room }
func (s *Session) GetPeerConn() *webrtc.WebRTC         { return s.conn }
func (s *Session) GetPlayerIndex() int                 { return s.pi }
func (s *Session) IsConnected() bool                   { return s.conn.IsConnected() }
func (s *Session) SendVideo(sample media.Sample) error { return s.conn.WriteVideo(sample) }
func (s *Session) SendAudio(sample media.Sample) error { return s.conn.WriteAudio(sample) }
func (s *Session) SetRoom(room *Room)                  { s.room = room }
func (s *Session) SetPlayerIndex(index int)            { s.pi = index }
func (s *Session) Close()                              { s.conn.Disconnect() } // TODO: Use event base
