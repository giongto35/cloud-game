package worker

import "github.com/giongto35/cloud-game/v2/pkg/webrtc"

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

// Close close a session
func (s *Session) Close() {
	// TODO: Use event base
	s.peerconnection.StopClient()
}
