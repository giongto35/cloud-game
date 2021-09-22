package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/network"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	"sync"
)

// Session represents a session connected from the browser to the current server
// It requires one connection to browser and one connection to the coordinator
// connection to browser is 1-1. connection to coordinator is n - 1
// Peerconnection can be from other server to ensure better latency
type Session struct {
	ID             string
	peerconnection *webrtc.WebRTC

	// Should I make direct reference
	RoomID string
}

type Sessions struct {
	mu    sync.Mutex
	store map[network.Uid]*Session
}

// Close close a session
func (s *Session) Close() {
	// TODO: Use event base
	s.peerconnection.StopClient()
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

func (ss *Sessions) Remove(id network.Uid) {
	ss.mu.Lock()
	delete(ss.store, id)
	ss.mu.Unlock()
}
