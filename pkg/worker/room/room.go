package room

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net"
	"os"
	"path/filepath"
	"sync"

	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/emulator"
	"github.com/giongto35/cloud-game/v2/pkg/emulator/libretro/nanoarch"
	"github.com/giongto35/cloud-game/v2/pkg/encoder"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/recorder"
	"github.com/giongto35/cloud-game/v2/pkg/session"
	"github.com/giongto35/cloud-game/v2/pkg/storage"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
)

// Room is a game session. multi webRTC sessions can connect to a same game.
// A room stores all the channel for interaction between all webRTCs session and emulator
type Room struct {
	ID string

	// imageChannel is image stream received from director
	imageChannel <-chan nanoarch.GameFrame
	// audioChannel is audio stream received from director
	audioChannel <-chan []int16
	// inputChannel is input stream send to director. This inputChannel is combined
	// input from webRTC + connection info (player index)
	inputChannel chan<- nanoarch.InputEvent
	// voiceInChannel is voice stream received from users
	//voiceInChannel chan []byte
	// voiceOutChannel is voice stream routed to all users
	//voiceOutChannel chan []byte
	//voiceSample     [][]byte
	// State of room
	IsRunning bool
	// Done channel is to fire exit event when room is closed
	Done chan struct{}
	// List of peer connections in the room
	rtcSessions []*webrtc.WebRTC
	// NOTE: Not in use, lock rtcSessions
	sessionsLock *sync.Mutex
	// Director is emulator
	director emulator.CloudEmulator
	// Cloud storage to store room state online
	onlineStorage storage.CloudStorage

	rec *recorder.Recording

	vPipe *encoder.VideoPipe
}

const (
	bufSize        = 245969
	SocketAddrTmpl = "/tmp/cloudretro-retro-%s.sock"
)

// NewVideoImporter return image Channel from stream
func NewVideoImporter(roomID string) chan nanoarch.GameFrame {
	sockAddr := fmt.Sprintf(SocketAddrTmpl, roomID)
	imgChan := make(chan nanoarch.GameFrame)

	l, err := net.Listen("unix", sockAddr)
	if err != nil {
		log.Fatal("listen error:", err)
	}

	log.Println("Creating uds server", sockAddr)
	go func(l net.Listener) {
		defer l.Close()

		conn, err := l.Accept()
		if err != nil {
			log.Fatal("Accept error: ", err)
		}
		defer conn.Close()

		log.Println("Received new conn")
		log.Println("Spawn Importer")

		fullBuf := make([]byte, bufSize*2)
		fullBuf = fullBuf[:0]

		for {
			// TODO: Not reallocate
			buf := make([]byte, bufSize)
			l, err := conn.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("error: %v", err)
				}
				continue
			}

			buf = buf[:l]
			fullBuf = append(fullBuf, buf...)
			if len(fullBuf) >= bufSize {
				buff := bytes.NewBuffer(fullBuf)
				dec := gob.NewDecoder(buff)

				frame := nanoarch.GameFrame{}
				err := dec.Decode(&frame)
				if err != nil {
					log.Fatalf("%v", err)
				}
				imgChan <- frame
				fullBuf = fullBuf[bufSize:]
			}
		}
	}(l)

	return imgChan
}

// NewRoom creates a new room
func NewRoom(roomID string, game games.GameMetadata, recUser string, rec bool, onlineStorage storage.CloudStorage, cfg worker.Config) *Room {
	if roomID == "" {
		roomID = session.GenerateRoomID(game.Name)
	}

	log.Println("New room: ", roomID, game)
	inputChannel := make(chan nanoarch.InputEvent, 100)

	room := &Room{
		ID: roomID,

		inputChannel: inputChannel,
		imageChannel: nil,
		//voiceInChannel:  make(chan []byte, 1),
		//voiceOutChannel: make(chan []byte, 1),
		rtcSessions:   []*webrtc.WebRTC{},
		sessionsLock:  &sync.Mutex{},
		IsRunning:     true,
		onlineStorage: onlineStorage,

		Done: make(chan struct{}, 1),
	}

	// Check if room is on local storage, if not, pull from GCS to local storage
	go func(game games.GameMetadata, roomID string) {
		store := nanoarch.Storage{
			Path:     cfg.Emulator.Storage,
			MainSave: roomID,
		}

		// Check room is on local or fetch from server
		log.Printf("Check for %s in the online storage", roomID)
		if err := room.saveOnlineRoomToLocal(roomID, store.GetSavePath()); err != nil {
			log.Printf("warn: room %s is not in the online storage, error %s", roomID, err)
		}

		// If not then load room or create room from local.
		log.Printf("Room %s started. GameName: %s, WithGame: %t", roomID, game.Name, cfg.Encoder.WithoutGame)

		// Spawn new emulator and plug-in all channels
		emuName := cfg.Emulator.GetEmulator(game.Type, game.Path)
		libretroConfig := cfg.Emulator.GetLibretroCoreConfig(emuName)

		if cfg.Encoder.WithoutGame {
			// Run without game, image stream is communicated over a unix socket
			imageChannel := NewVideoImporter(roomID)
			director, _, audioChannel := nanoarch.Init(roomID, false, inputChannel, store, libretroConfig)
			room.imageChannel = imageChannel
			room.director = director
			room.audioChannel = audioChannel
		} else {
			// Run without game, image stream is communicated over image channel
			director, imageChannel, audioChannel := nanoarch.Init(roomID, true, inputChannel, store, libretroConfig)
			room.imageChannel = imageChannel
			room.director = director
			room.audioChannel = audioChannel
		}

		gameMeta := room.director.LoadMeta(filepath.Join(game.Base, game.Path))

		// nwidth, nheight are the WebRTC output size
		var nwidth, nheight int
		emu, ar := cfg.Emulator, cfg.Emulator.AspectRatio

		if ar.Keep {
			baseAspectRatio := float64(gameMeta.BaseWidth) / float64(ar.Height)
			nwidth, nheight = resizeToAspect(baseAspectRatio, ar.Width, ar.Height)
			log.Printf("Viewport size will be changed from %dx%d (%f) -> %dx%d", ar.Width, ar.Height,
				baseAspectRatio, nwidth, nheight)
		} else {
			nwidth, nheight = gameMeta.BaseWidth, gameMeta.BaseHeight
			log.Printf("Viewport custom size is disabled, base size will be used instead %dx%d", nwidth, nheight)
		}

		if emu.Scale > 1 {
			nwidth, nheight = nwidth*emu.Scale, nheight*emu.Scale
			log.Printf("Viewport size has scaled to %dx%d", nwidth, nheight)
		}

		// set game frame size considering its orientation
		encoderW, encoderH := nwidth, nheight
		if gameMeta.Rotation.IsEven {
			encoderW, encoderH = nheight, nwidth
		}

		if cfg.Recording.Enabled {
			room.rec = recorder.NewRecording(
				recorder.Meta{UserName: recUser},
				recorder.Options{
					Dir:                   cfg.Recording.Folder,
					Fps:                   gameMeta.Fps,
					Frequency:             gameMeta.AudioSampleRate,
					Game:                  game.Name,
					ImageCompressionLevel: cfg.Recording.CompressLevel,
					Name:                  cfg.Recording.Name,
					Zip:                   cfg.Recording.Zip,
				})
			room.ToggleRecording(rec, recUser)
		}

		room.director.SetViewport(encoderW, encoderH)

		// Spawn video and audio encoding for webRTC
		go room.startVideo(encoderW, encoderH, cfg.Encoder.Video)
		go room.startAudio(gameMeta.AudioSampleRate, cfg.Encoder.Audio)
		//go room.startVoice()
		room.director.Start()
	}(game, roomID)
	return room
}

func resizeToAspect(ratio float64, sw int, sh int) (dw int, dh int) {
	// ratio is always > 0
	dw = int(math.Round(float64(sh)*ratio/2) * 2)
	dh = sh
	if dw > sw {
		dw = sw
		dh = int(math.Round(float64(sw)/ratio/2) * 2)
	}
	return
}

func isGameOnLocal(path string) bool {
	file, err := os.Open(path)
	if err == nil {
		defer func() {
			_ = file.Close()
		}()
	}
	return !errors.Is(err, os.ErrNotExist)
}

func (r *Room) AddConnectionToRoom(peerconnection *webrtc.WebRTC) {
	peerconnection.AttachRoomID(r.ID)
	r.rtcSessions = append(r.rtcSessions, peerconnection)

	go r.startWebRTCSession(peerconnection)
}

func (r *Room) UpdatePlayerIndex(peerconnection *webrtc.WebRTC, playerIndex int) {
	log.Println("Updated player Index to: ", playerIndex)
	peerconnection.PlayerIndex = playerIndex
}

func (r *Room) startWebRTCSession(peerconnection *webrtc.WebRTC) {
	defer func() {
		if r := recover(); r != nil {
			log.Println("Warn: Recovered when sent to close inputChannel")
		}
	}()

	log.Println("Start WebRTC session")
	//go func() {
	//
	//	// set up voice input and output. A room has multiple voice input and only one combined voice output.
	//	for voiceInput := range peerconnection.VoiceInChannel {
	//		// NOTE: when room is no longer running. InputChannel needs to have extra event to go inside the loop
	//		if peerconnection.Done || !peerconnection.IsConnected() || !r.IsRunning {
	//			break
	//		}
	//
	//		if peerconnection.IsConnected() {
	//			r.voiceInChannel <- voiceInput
	//		}
	//
	//	}
	//}()

	// bug: when input channel here = nil, skip and finish
	for input := range peerconnection.InputChannel {
		// NOTE: when room is no longer running. InputChannel needs to have extra event to go inside the loop
		if peerconnection.Done || !peerconnection.IsConnected() || !r.IsRunning {
			break
		}

		if peerconnection.IsConnected() {
			select {
			case r.inputChannel <- nanoarch.InputEvent{RawState: input, PlayerIdx: peerconnection.PlayerIndex, ConnID: peerconnection.ID}:
			default:
			}
		}
	}
	log.Printf("[worker] peer connection is done")
}

// RemoveSession removes a peerconnection from room and return true if there is no more room
func (r *Room) RemoveSession(w *webrtc.WebRTC) {
	log.Println("Cleaning session: ", w.ID)
	// TODO: get list of r.rtcSessions in lock
	for i, s := range r.rtcSessions {
		log.Println("found session: ", w.ID)
		if s.ID == w.ID {
			r.rtcSessions = append(r.rtcSessions[:i], r.rtcSessions[i+1:]...)
			s.RoomID = ""
			log.Println("Removed session ", s.ID, " from room: ", r.ID)
			break
		}
	}
	// Detach input. Send end signal
	select {
	case r.inputChannel <- nanoarch.InputEvent{RawState: []byte{0xFF, 0xFF}, ConnID: w.ID}:
	default:
	}
}

// TODO: Reuse for remove Session
func (r *Room) IsPCInRoom(w *webrtc.WebRTC) bool {
	if r == nil {
		return false
	}
	for _, s := range r.rtcSessions {
		if s.ID == w.ID {
			return true
		}
	}
	return false
}

func (r *Room) Close() {
	if !r.IsRunning {
		return
	}

	r.IsRunning = false
	log.Println("Closing room and director of room ", r.ID)

	// Save game before quit. Only save for game which was previous saved to avoid flooding database
	if r.isRoomExisted() {
		log.Println("Saved Game before closing room")
		// use goroutine here because SaveGame attempt to acquire a emulator lock.
		// the lock is holding before coming to close, so it will cause deadlock if SaveGame is synchronous
		go func() {
			// Save before close, so save can have correct state (Not sure) may again cause deadlock
			if err := r.SaveGame(); err != nil {
				log.Println("[error] couldn't save the game during closing")
			}
			r.director.Close()
		}()
	} else {
		r.director.Close()
	}
	log.Println("Closing input of room ", r.ID)
	close(r.inputChannel)
	//close(r.voiceOutChannel)
	//close(r.voiceInChannel)
	close(r.Done)
	// Close here is a bit wrong because this read channel
	// Just dont close it, let it be gc
	//close(r.imageChannel)
	//close(r.audioChannel)
	if r.rec != nil {
		if err := r.rec.Stop(); err != nil {
			log.Printf("record close err, %v", err)
		}
	}
}

func (r *Room) isRoomExisted() bool {
	// Check if room is in online storage
	_, err := r.onlineStorage.Load(r.ID)
	if err == nil {
		return true
	}
	return isGameOnLocal(r.director.GetHashPath())
}

// SaveGame writes save state on the disk as well as
// uploads it to a cloud storage.
func (r *Room) SaveGame() error {
	// TODO: Move to game view
	if err := r.director.SaveGame(); err != nil {
		return err
	}
	if err := r.onlineStorage.Save(r.ID, r.director.GetHashPath()); err != nil {
		return err
	}
	log.Printf("success, cloud save")
	return nil
}

// saveOnlineRoomToLocal save online room to local.
// !Supports only one file of main save state.
func (r *Room) saveOnlineRoomToLocal(roomID string, savePath string) error {
	data, err := r.onlineStorage.Load(roomID)
	if err != nil {
		return err
	}
	// Save the data fetched from a cloud provider to the local server
	if data != nil {
		if err := ioutil.WriteFile(savePath, data, 0644); err != nil {
			return err
		}
		log.Printf("successfully downloaded cloud save")
	}
	return nil
}

func (r *Room) LoadGame() error { return r.director.LoadGame() }

func (r *Room) ToggleMultitap() error { return r.director.ToggleMultitap() }

func (r *Room) IsEmpty() bool { return len(r.rtcSessions) == 0 }

func (r *Room) IsRunningSessions() bool {
	// If there is running session
	for _, s := range r.rtcSessions {
		if s.IsConnected() {
			return true
		}
	}

	return false
}

func (r *Room) ToggleRecording(active bool, user string) {
	if r.rec == nil {
		return
	}
	r.rec.Set(active, user)
}
