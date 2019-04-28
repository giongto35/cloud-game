package handler

import (
	"image"
	"log"
	"math/rand"
	"strconv"
	"sync"

	ui "github.com/giongto35/cloud-game/emulator"
	"github.com/giongto35/cloud-game/webrtc"
)

// Room is a game session. multi webRTC sessions can connect to a same game.
// A room stores all the channel for interaction between all webRTCs session and emulator
type Room struct {
	imageChannel chan *image.RGBA
	audioChannel chan float32
	inputChannel chan int
	// Done channel is to fire exit event when there is no webRTC session running
	Done chan struct{}

	rtcSessions  []*webrtc.WebRTC
	sessionsLock *sync.Mutex

	director *ui.Director
}

var rooms = map[string]*Room{}

// generateRoomID generate a unique room ID containing 16 digits
func generateRoomID() string {
	roomID := strconv.FormatInt(rand.Int63(), 16)
	//roomID := uuid.Must(uuid.NewV4()).String()
	return roomID
}

// init initilizes a room returns roomID
func initRoom(roomID, gameName string) string {
	// if no roomID is given, generate it
	if roomID == "" {
		roomID = generateRoomID()
	}
	log.Println("Init new room", roomID, gameName)
	imageChannel := make(chan *image.RGBA, 100)
	audioChannel := make(chan float32, ui.SampleRate)
	inputChannel := make(chan int, 100)

	// create director
	director := ui.NewDirector(roomID, imageChannel, audioChannel, inputChannel)

	room := &Room{
		imageChannel: imageChannel,
		audioChannel: audioChannel,
		inputChannel: inputChannel,
		rtcSessions:  []*webrtc.WebRTC{},
		sessionsLock: &sync.Mutex{},
		director:     director,
		Done:         make(chan struct{}),
	}
	rooms[roomID] = room

	go room.startVideo()
	go room.startAudio()
	go director.Start([]string{"games/" + gameName})

	return roomID
}

// isRoomRunning check if there is any running sessions.
// TODO: If we remove sessions from room anytime a session is closed, we can check if the sessions list is empty or not.
func isRoomRunning(roomID string) bool {
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

// startWebRTCSession fan-in of the same room to inputChannel
func startWebRTCSession(room *Room, webRTC *webrtc.WebRTC, playerIndex int) {
	inputChannel := room.inputChannel
	log.Println("room, inputChannel", room, inputChannel)
	for {
		select {
		case <-webRTC.Done:
			removeSession(webRTC, room)
		default:
		}
		// Client stopped
		if webRTC.IsClosed() {
			return
		}

		// encode frame
		if webRTC.IsConnected() {
			input := <-webRTC.InputChannel
			// the first 8 bits belong to player 1
			// the next 8 belongs to player 2 ...
			// We standardize and put it to inputChannel (16 bits)
			input = input << ((uint(playerIndex) - 1) * ui.NumKeys)
			inputChannel <- input
		}
	}
}

func cleanSession(w *webrtc.WebRTC) {
	room, ok := rooms[w.RoomID]
	if !ok {
		return
	}
	removeSession(w, room)
}

func removeSession(w *webrtc.WebRTC, room *Room) {
	room.sessionsLock.Lock()
	defer room.sessionsLock.Unlock()
	for i, s := range room.rtcSessions {
		if s == w {
			room.rtcSessions = append(room.rtcSessions[:i], room.rtcSessions[i+1:]...)
			break
		}
	}
	// If room has no sessions, close room
	if len(room.rtcSessions) == 0 {
		room.Done <- struct{}{}
	}
}

func (r *Room) remove() {
	log.Println("Closing room", r)
	r.director.Done <- struct{}{}
}
