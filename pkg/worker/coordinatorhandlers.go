package worker

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"strconv"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	webrtcConf "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/worker/room"
)

func MakeConnectionRequest(conf worker.Worker, address string) (string, error) {
	req := api.ConnectionRequest{
		Zone:     conf.Network.Zone,
		PingAddr: conf.GetPingAddr(address),
		IsHTTPS:  conf.Server.Https,
	}
	rez, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(rez), nil
}

func (c *Coordinator) HandleTerminateSession(data json.RawMessage, h *Handler) {
	resp, err := c.terminateSession(data)
	if err != nil {
		c.Logf("error: broken terminate session request %v", err)
		return
	}
	c.Logf("Received a terminate session -> %v", resp.Id)
	session := h.sessions.Get(resp.Id)
	if session != nil {
		session.Close()
		h.sessions.Remove(resp.Id)
		h.detachPeerConn(session.peerconnection)
	} else {
		log.Printf("Error: No session for ID: %s\n", resp.Id)
	}
}

func (c *Coordinator) HandleWebrtcInit(packet ipc.InPacket, h *Handler) {
	resp, err := c.webrtcInit(packet.Payload)
	if err != nil {
		c.Logf("error: broken webrtc init request %v", err)
		return
	}

	peerconnection := webrtc.NewWebRTC().WithConfig(webrtcConf.Config{Encoder: h.cfg.Encoder, Webrtc: h.cfg.Webrtc})

	// send back candidate string to browser
	localSDP, err := peerconnection.StartClient(func(cd string) { h.cord.IceCandidate(cd, string(resp.Id)) })

	if err != nil {
		log.Println("error: cannot create new webrtc session", err)
		_ = h.cord.SendPacket(packet.Proxy(ipc.EmptyPacket))
	}

	// Create new sessions when we have new peerconnection initialized
	h.sessions.Add(resp.Id, &Session{peerconnection: peerconnection})
	log.Println("Start peerconnection", resp.Id)

	_ = h.cord.SendPacket(packet.Proxy(localSDP))
}

func (c *Coordinator) HandleWebrtcAnswer(packet ipc.InPacket, h *Handler) {
	var resp api.WebrtcAnswerRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		log.Printf("error: broken webrtc init request %v", err)
		return
	}
	if session := h.sessions.Get(resp.Id); session != nil {
		if err := session.peerconnection.SetRemoteSDP(resp.Sdp); err != nil {
			log.Printf("Error: cannot set RemoteSDP of client: %v beacuse %v", resp.Id, err)
		}
	} else {
		log.Printf("Error: No session for ID: %s", resp.Id)
	}
}

func (c *Coordinator) HandleWebrtcIceCandidate(packet ipc.InPacket, h *Handler) {
	var resp api.WebrtcIceCandidateRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		log.Printf("error: broken webrtc candidate request %v", err)
		return
	}
	if session := h.sessions.Get(resp.Id); session != nil {
		if err := session.peerconnection.AddCandidate(resp.Candidate); err != nil {
			log.Printf("error: cannot add IceCandidate of client: %s", resp.Id)
		}
	} else {
		log.Printf("error: no session for ID: %s", resp.Id)
	}
}

func (c *Coordinator) HandleGameStart(packet ipc.InPacket, h *Handler) {
	var resp api.StartGameRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		log.Printf("error: broken game start request %v", err)
		_ = h.cord.SendPacket(packet.Proxy(ipc.EmptyPacket))
		return
	}
	session := h.sessions.Get(resp.Stateful.Id)
	if session == nil {
		log.Printf("error: no session for ID: %s", resp.Stateful.Id)
		_ = h.cord.SendPacket(packet.Proxy(ipc.EmptyPacket))
		return
	}

	gameMeta := games.GameMetadata{Name: resp.Game.Name, Base: resp.Game.Base, Type: resp.Game.Type, Path: resp.Game.Path}
	gameRoom := h.startGameHandler(gameMeta, resp.Room.Id, resp.PlayerIndex, session.peerconnection)
	session.RoomID = gameRoom.ID
	h.rooms.Add(gameRoom)
	_ = h.cord.SendPacket(packet.Proxy(api.StartGameResponse{Room: api.Room{Id: gameRoom.ID}}))
}

// startGameHandler starts a game if roomID is given, if not create new room
func (h *Handler) startGameHandler(game games.GameMetadata, existedRoomID string, playerIndex int, peerconnection *webrtc.WebRTC) *room.Room {
	log.Printf("Loading game: %v", game.Name)
	// If we are connecting to coordinator, request corresponding serverID based on roomID
	// TODO: check if existedRoomID is in the current server
	gameRoom := h.rooms.Get(existedRoomID)
	// If room is not running
	if gameRoom == nil {
		log.Println("Got Room from local ", gameRoom, " ID: ", existedRoomID)
		// Create new room and update player index
		gameRoom = h.createRoom(existedRoomID, game)
		gameRoom.UpdatePlayerIndex(peerconnection, playerIndex)

		// Wait for done signal from room
		go func() {
			<-gameRoom.Done
			h.rooms.Remove(gameRoom.ID)
			// TODO add proper non-crash close logic
			// send signal to coordinator that the room is closed, coordinator will remove that room
			h.cord.CloseRoom(gameRoom.ID)
		}()
	}

	// Attach peerconnection to room. If PC is already in room, don't detach
	log.Printf("The peer is in the room: %v", gameRoom.IsPCInRoom(peerconnection))
	if !gameRoom.IsPCInRoom(peerconnection) {
		h.detachPeerConn(peerconnection)
		gameRoom.AddConnectionToRoom(peerconnection)
	}

	// Register room to coordinator if we are connecting to coordinator
	if gameRoom != nil {
		h.cord.RegisterRoom(gameRoom.ID)
	}
	return gameRoom
}

func (c *Coordinator) HandleQuitGame(packet ipc.InPacket, h *Handler) {
	var resp api.GameQuitRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		log.Printf("error: broken game quit request %v", err)
		return
	}
	session := h.sessions.Get(resp.Stateful.Id)
	if session != nil {
		rm := h.rooms.Get(session.RoomID)
		// Defensive coding, check if the peerconnection is in room
		if rm != nil && rm.IsPCInRoom(session.peerconnection) {
			h.detachPeerConn(session.peerconnection)
		}
	} else {
		log.Printf("error: no session for ID: %s", resp.Stateful.Id)
	}
}

func (c *Coordinator) HandleSaveGame(packet ipc.InPacket, h *Handler) {
	var resp api.SaveGameRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		log.Printf("error: broken game save request %v", err)
		return
	}
	log.Printf("RoomID: %v", resp.Room.Id)
	rez := ipc.OkPacket
	if resp.Room.Id != "" {
		rm := h.rooms.Get(resp.Room.Id)
		if rm == nil {
			return
		}
		err := rm.SaveGame()
		if err != nil {
			log.Println("[!] Cannot save game state: ", err)
			rez = ipc.ErrPacket
		}
	} else {
		rez = ipc.ErrPacket
	}
	_ = h.cord.SendPacket(packet.Proxy(rez))
}

func (c *Coordinator) HandleLoadGame(packet ipc.InPacket, h *Handler) {
	log.Println("Loading game state")
	var resp api.LoadGameRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		log.Printf("error: broken game load request %v", err)
		return
	}
	rez := ipc.OkPacket
	if resp.Room.Id != "" {
		rm := h.rooms.Get(resp.Room.Id)
		if rm == nil {
			return
		}
		if err := rm.LoadGame(); err != nil {
			log.Println("[!] Cannot load game state: ", err)
			rez = ipc.ErrPacket
		}
	} else {
		rez = ipc.ErrPacket
	}
	_ = h.cord.SendPacket(packet.Proxy(rez))
}

func (c *Coordinator) HandleChangePlayer(packet ipc.InPacket, h *Handler) {
	var resp api.ChangePlayerRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		log.Printf("error: broken change player request %v", err)
		return
	}

	session := h.sessions.Get(resp.Stateful.Id)
	idx, err := strconv.Atoi(resp.Index)
	rm := h.rooms.Get(resp.Room.Id)
	if rm == nil {
		return
	}
	var rez api.ChangePlayerResponse
	if session != nil && err == nil {
		rm.UpdatePlayerIndex(session.peerconnection, idx)
		rez = strconv.Itoa(idx)
	} else {
		rez = ipc.ErrPacket
	}
	_ = h.cord.SendPacket(packet.Proxy(rez))
}

func (c *Coordinator) HandleToggleMultitap(packet ipc.InPacket, h *Handler) {
	var resp api.ToggleMultitapRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		log.Printf("error: broken toggle multitap request %v", err)
		return
	}

	rez := ipc.OkPacket
	if resp.Room.Id != "" {
		rm := h.rooms.Get(resp.Room.Id)
		if rm == nil {
			return
		}
		if err := rm.ToggleMultitap(); err != nil {
			log.Println("[!] Could not toggle multitap state: ", err)
			rez = ipc.ErrPacket
		}
	} else {
		rez = ipc.ErrPacket
	}
	_ = h.cord.SendPacket(packet.Proxy(rez))
}
