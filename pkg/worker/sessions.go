package worker

import (
	"sync"

	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	"github.com/pion/webrtc/v3/pkg/media"
)

// Session represents a user session.
// A WebRTC connection form a browser to the current server.
//
// todo rephrase
// It requires one connection to browser and one connection to the coordinator
// connection to browser is 1-1. connection to coordinator is n - 1
// Peerconnection can be from other server to ensure better latency
type Session struct {
	id         network.Uid
	connection *webrtc.WebRTC
	playerNo   int
	room       *Room
}

type Sessions struct {
	mu    sync.Mutex
	store map[network.Uid]*Session
}

func NewSession(connection *webrtc.WebRTC, id network.Uid) *Session {
	return &Session{id: id, connection: connection}
}

func (s *Session) GetId() string { return s.id.String() }

func (s *Session) GetShortId() string { return s.id.Short() }

func (s *Session) GetRoom() *Room { return s.room }

func (s *Session) GetPeerConn() *webrtc.WebRTC { return s.connection }

func (s *Session) GetPlayerIndex() int { return s.playerNo }

func (s *Session) IsConnected() bool { return s.connection.IsConnected() }

func (s *Session) SendVideo(sample media.Sample) error { return s.connection.WriteVideo(sample) }

func (s *Session) SendAudio(sample media.Sample) error { return s.connection.WriteAudio(sample) }

func (s *Session) SetRoom(room *Room) { s.room = room }

func (s *Session) SetPlayerIndex(index int) { s.playerNo = index }

func (s *Session) Close() {
	// TODO: Use event base
	s.connection.Disconnect()
}

func NewSessions() Sessions {
	return Sessions{store: make(map[network.Uid]*Session, 10)}
}

func (ss *Sessions) Get(id network.Uid) *Session {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	return ss.store[id]
}

func (ss *Sessions) Add(id network.Uid, value *Session) {
	ss.mu.Lock()
	ss.store[id] = value
	ss.mu.Unlock()
}

func (ss *Sessions) Remove(s *Session) {
	ss.mu.Lock()
	delete(ss.store, s.id)
	ss.mu.Unlock()
}

func (ss *Sessions) IsEmpty() bool { return len(ss.store) == 0 }

func (ss *Sessions) ForEach(do func(*Session)) {
	ss.mu.Lock()
	for _, v := range ss.store {
		do(v)
	}
	ss.mu.Unlock()
}
