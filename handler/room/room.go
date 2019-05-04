package room

import (
	"fmt"
	"image"
	"io/ioutil"
	"log"
	"math/rand"
	"strconv"
	"sync"

	emulator "github.com/giongto35/cloud-game/emulator"
	storage "github.com/giongto35/cloud-game/handler/cloud-storage"
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

	// Cloud storage to store room state online
	onlineStorage *storage.Client
}

// NewRoom creates a new room
func NewRoom(roomID, gamepath, gameName string, onlineStorage *storage.Client) *Room {
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

		imageChannel:  imageChannel,
		audioChannel:  audioChannel,
		inputChannel:  inputChannel,
		rtcSessions:   []*webrtc.WebRTC{},
		sessionsLock:  &sync.Mutex{},
		director:      director,
		Done:          make(chan struct{}),
		onlineStorage: onlineStorage,
	}

	go room.startVideo()
	go room.startAudio()
	go director.Start([]string{gamepath + "/" + gameName})

	return room
}

// generateRoomID generate a unique room ID containing 16 digits
func generateRoomID() string {
	roomID := strconv.FormatInt(rand.Int63(), 16)
	log.Println("Generate Room ID", roomID)
	//roomID := uuid.Must(uuid.NewV4()).String()
	return roomID
}

func (r *Room) AddConnectionToRoom(peerconnection *webrtc.WebRTC, playerIndex int) {
	peerconnection.AttachRoomID(r.ID)
	r.rtcSessions = append(r.rtcSessions, peerconnection)

	go r.startWebRTCSession(peerconnection, playerIndex)
}

// startWebRTCSession fan-in of the same room to inputChannel
func (r *Room) startWebRTCSession(peerconnection *webrtc.WebRTC, playerIndex int) {
	inputChannel := r.inputChannel
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

func (r *Room) CleanSession(peerconnection *webrtc.WebRTC) {
	r.removeSession(peerconnection)
	// TODO: Clean all channels
}

func (r *Room) removeSession(w *webrtc.WebRTC) {
	fmt.Println("Cleaning session: ", w)
	r.sessionsLock.Lock()
	defer r.sessionsLock.Unlock()
	fmt.Println("Sessions list", r.rtcSessions)
	for i, s := range r.rtcSessions {
		fmt.Println("found session: ", s, w)
		if s.ID == w.ID {
			r.rtcSessions = append(r.rtcSessions[:i], r.rtcSessions[i+1:]...)
			fmt.Println("found session: ", len(r.rtcSessions))

			// If room has no sessions, close room
			if len(r.rtcSessions) == 0 {
				log.Println("No session in room")
				r.Done <- struct{}{}
			}
			break
		}
	}
}

func (r *Room) Close() {
	log.Println("Closing room", r)
	r.director.Done <- struct{}{}
}

func (r *Room) SaveGame() error {
	onlineSaveFunc := func() error {
		// Try to save the game to gCloud
		if err := r.onlineStorage.SaveFile(r.director.GetHash(), r.director.GetHashPath()); err != nil {
			return err
		}

		return nil
	}

	// TODO: Move to game view
	if err := r.director.SaveGame(onlineSaveFunc); err != nil {
		return err
	}

	return nil
}

func (r *Room) LoadGame() error {
	// TODO: Fix, because load game always come to local, this logic is unnecessary. Move to load game
	onlineLoadFunc := func() error {
		log.Println("Loading game from cloud storage")
		// If the game is not on local server
		// Try to load from gcloud
		data, err := r.onlineStorage.LoadFile(r.director.GetHash())
		if err != nil {
			return err
		}
		// Save the data fetched from gcloud to local server
		ioutil.WriteFile(r.director.GetHashPath(), data, 0644)
		// Reload game again
		//err = r.director.LoadGame(nil)
		//if err != nil {
		//return err
		//}
		return nil
	}

	err := r.director.LoadGame(onlineLoadFunc)

	return err
}

func (r *Room) IsRunning() bool {
	// If there is running session
	for _, s := range r.rtcSessions {
		if !s.IsClosed() {
			return true
		}
	}

	return false
}
