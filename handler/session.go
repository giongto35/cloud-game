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

	// TODO: Decouple this
	handler *Handler

	ServerID    string
	GameName    string
	RoomID      string
	PlayerIndex int
}

// getRoom returns room from roomID
func (h *Handler) getRoom(roomID string) *Room {
	room, ok := h.rooms[roomID]
	if !ok {
		return nil
	}

	return room
}

// createNewRoom creates a new room
// Return nil in case of room is existed
func (h *Handler) createNewRoom(gameName string, roomID string, playerIndex int) *Room {
	//cleanSession(peerconnection)
	// If the roomID is empty,
	// or the roomID doesn't have any running sessions (room was closed)
	// we spawn a new room
	if roomID == "" || !h.isRoomRunning(roomID) {
		room := h.initRoom(roomID, gameName)
		h.rooms[roomID] = room
		return room
	}

	// TODO: Might have race condition
	//rooms[roomID].rtcSessions = append(rooms[roomID].rtcSessions, peerconnection)
	//room := rooms[roomID]

	//peerconnection.AttachRoomID(roomID)
	//go startWebRTCSession(room, peerconnection, playerIndex)

	return nil
}

// startPeerConnection handles one peerconnection call
//func (s *Handler) startPeerConnection( gameName string, roomID string, playerIndex int) (rRoomID string, isNewRoom bool) {
//isNewRoom = false
//cleanSession(peerconnection)
//// If the roomID is empty,
//// or the roomID doesn't have any running sessions (room was closed)
//// we spawn a new room
//if roomID == "" || !isRoomRunning(roomID) {
//roomID = initRoom(roomID, gameName)
//isNewRoom = true
//}

//// TODO: Might have race condition
//rooms[roomID].rtcSessions = append(rooms[roomID].rtcSessions, peerconnection)
//room := rooms[roomID]

//peerconnection.AttachRoomID(roomID)
//go startWebRTCSession(room, peerconnection, playerIndex)

//return roomID, isNewRoom
//}
