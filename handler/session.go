package handler

import (
	"github.com/giongto35/cloud-game/webrtc"
)

// Session represents a session connected from the browser to the current server
// It involves one connection to browser and one connection to the overlord
// Peerconnection can be from other server to ensure better latency
type Session struct {
	ID             string
	BrowserClient  *BrowserClient
	OverlordClient *OverlordClient
	peerconnection *webrtc.WebRTC

	ServerID    string
	GameName    string
	RoomID      string
	PlayerIndex int
}

// startSession handles one session call
func startSession(webRTC *webrtc.WebRTC, gameName string, roomID string, playerIndex int) (rRoomID string, isNewRoom bool) {
	isNewRoom = false
	cleanSession(webRTC)
	// If the roomID is empty,
	// or the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if roomID == "" || !isRoomRunning(roomID) {
		roomID = initRoom(roomID, gameName)
		isNewRoom = true
	}

	// TODO: Might have race condition
	rooms[roomID].rtcSessions = append(rooms[roomID].rtcSessions, webRTC)
	room := rooms[roomID]

	webRTC.AttachRoomID(roomID)
	go startWebRTCSession(room, webRTC, playerIndex)

	return roomID, isNewRoom
}
