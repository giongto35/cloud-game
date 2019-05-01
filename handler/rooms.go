package handler

import (
	"image"
	"log"
	"math/rand"
	"strconv"
	"sync"

	emulator "github.com/giongto35/cloud-game/emulator"
	"github.com/giongto35/cloud-game/webrtc"
)

// Room is a game session. multi webRTC sessions can connect to a same game.
// A room stores all the channel for interaction between all webRTCs session and emulator
type Room struct {
	ID string

	imageChannel chan *image.RGBA
	audioChannel chan float32
	inputChannel chan int
	// Done channel is to fire exit event when there is no webRTC session running
	Done chan struct{}

	rtcSessions  []*webrtc.WebRTC
	sessionsLock *sync.Mutex

	director *emulator.Director
}

var rooms = map[string]*Room{}

// generateRoomID generate a unique room ID containing 16 digits
func generateRoomID() string {
	roomID := strconv.FormatInt(rand.Int63(), 16)
	//roomID := uuid.Must(uuid.NewV4()).String()
	return roomID
}

// init initilizes a room returns roomID
func (h *Handler) initRoom(roomID, gameName string) *Room {
	// if no roomID is given, generate it
	if roomID == "" {
		roomID = generateRoomID()
	}
	log.Println("Init new room", roomID, gameName)
	imageChannel := make(chan *image.RGBA, 100)
	audioChannel := make(chan float32, emulator.SampleRate)
	inputChannel := make(chan int, 100)

	// create director
	director := emulator.NewDirector(roomID, imageChannel, audioChannel, inputChannel)

	room := &Room{
		ID: roomID,

		imageChannel: imageChannel,
		audioChannel: audioChannel,
		inputChannel: inputChannel,
		rtcSessions:  []*webrtc.WebRTC{},
		sessionsLock: &sync.Mutex{},
		director:     director,
		Done:         make(chan struct{}),
	}

	go room.startVideo()
	go room.startAudio()
	go director.Start([]string{"../games/" + gameName})

	return room
}

// isRoomRunning check if there is any running sessions.
// TODO: If we remove sessions from room anytime a session is closed, we can check if the sessions list is empty or not.
func (h *Handler) isRoomRunning(roomID string) bool {
	// If no roomID is registered
	if _, ok := rooms[roomID]; !ok {
		return false
	}

	// If there is running session
	for _, s := range rooms[roomID].rtcSessions {
		if !s.IsClosed() {
			return true
		}
	}
	return false
}

func (r *Room) addConnectionToRoom(peerconnection *webrtc.WebRTC, playerIndex int) {
	r.cleanSession(peerconnection)
	peerconnection.AttachRoomID(r.ID)
	go r.startWebRTCSession(peerconnection, playerIndex)

	r.rtcSessions = append(r.rtcSessions, peerconnection)
}

// startWebRTCSession fan-in of the same room to inputChannel
func (r *Room) startWebRTCSession(peerconnection *webrtc.WebRTC, playerIndex int) {
	inputChannel := r.inputChannel
	log.Println("room, inputChannel", r, inputChannel)
	for {
		select {
		case <-peerconnection.Done:
			r.removeSession(peerconnection)
		default:
		}
		// Client stopped
		if peerconnection.IsClosed() {
			return
		}

		// encode frame
		if peerconnection.IsConnected() {
			input := <-peerconnection.InputChannel
			// the first 8 bits belong to player 1
			// the next 8 belongs to player 2 ...
			// We standardize and put it to inputChannel (16 bits)
			input = input << ((uint(playerIndex) - 1) * emulator.NumKeys)
			inputChannel <- input
		}
	}
}

func (r *Room) cleanSession(peerconnection *webrtc.WebRTC) {
	r.removeSession(peerconnection)
}

func (r *Room) removeSession(w *webrtc.WebRTC) {
	r.sessionsLock.Lock()
	defer r.sessionsLock.Unlock()
	for i, s := range r.rtcSessions {
		if s == w {
			r.rtcSessions = append(r.rtcSessions[:i], r.rtcSessions[i+1:]...)
			break
		}
	}
	// If room has no sessions, close room
	if len(r.rtcSessions) == 0 {
		r.Done <- struct{}{}
	}
}

func (r *Room) remove() {
	log.Println("Closing room", r)
	r.director.Done <- struct{}{}
}
