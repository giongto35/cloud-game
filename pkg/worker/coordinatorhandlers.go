package worker

import (
	"encoding/json"
	"log"
	"strconv"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	webrtcConfig "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/worker/room"
)

func (c *Coordinator) HandleIdentifyWorker(data json.RawMessage, h *Handler) {
	resp, err := c.identifyWorkerInRequest(data)
	if err != nil {
		c.Printf("error: broken identify request %v", err)
		return
	}
	c.Printf("[worker] new id: %s", resp)
	h.serverID = resp
}

func (c *Coordinator) HandleTerminateSession(data json.RawMessage, h *Handler) {
	resp, err := c.terminateSession(data)
	if err != nil {
		c.Printf("error: broken terminate session request %v", err)
		return
	}
	c.Println("Received a terminate session ", resp.Id)
	session := h.getSession(resp.Id)
	if session != nil {
		session.Close()
		h.removeSession(resp.Id)
		h.detachPeerConn(session.peerconnection)
	} else {
		log.Printf("Error: No session for ID: %s\n", resp.Id)
	}
	c.Printf("SESSIONS AFTER DELETE: %v", h.sessions)
}

func (c *Coordinator) HandleWebrtcInit(packet ipc.InPacket, h *Handler) {
	log.Println("Received a request to createOffer from browser via coordinator")
	resp, err := c.webrtcInit(packet.Payload)
	if err != nil {
		c.Printf("error: broken webrtc init request %v", err)
		return
	}

	peerconnection := webrtc.NewWebRTC().WithConfig(
		webrtcConfig.Config{Encoder: h.cfg.Encoder, Webrtc: h.cfg.Webrtc},
	)

	localSDP, err := peerconnection.StartClient(
		// send back candidate string to browser
		func(cd string) { h.cord.IceCandidate(cd, string(resp.Id)) },
	)

	if err != nil {
		log.Println("Error: Cannot create new webrtc session", err)
		_ = h.cord.wire.SendPacket(ipc.OutPacket{
			Id:      packet.Id,
			T:       packet.T,
			Payload: "",
		})
	}

	// Create new sessions when we have new peerconnection initialized
	h.addSession(resp.Id, &Session{peerconnection: peerconnection})
	log.Println("Start peerconnection", resp.Id)

	_ = h.cord.wire.SendPacket(ipc.OutPacket{
		Id:      packet.Id,
		T:       packet.T,
		Payload: localSDP,
	})
}

func (c *Coordinator) HandleWebrtcAnswer(packet ipc.InPacket, h *Handler) {
	c.Println("Received answer SDP from browser")
	var resp api.WebrtcAnswerRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.Printf("error: broken webrtc init request %v", err)
		return
	}
	session := h.getSession(resp.Id)
	if session != nil {
		peerconnection := session.peerconnection
		err := peerconnection.SetRemoteSDP(resp.Sdp)
		if err != nil {
			log.Printf("Error: cannot set RemoteSDP of client: %v beacuse %v", resp.Id, err)
		}
	} else {
		log.Printf("Error: No session for ID: %s\n", resp.Id)
	}
}

func (c *Coordinator) HandleWebrtcIceCandidate(packet ipc.InPacket, h *Handler) {
	c.Println("Received remote Ice Candidate from browser")
	var resp api.WebrtcIceCandidateRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.Printf("error: broken webrtc candidate request %v", err)
		return
	}
	session := h.getSession(resp.Id)

	if session != nil {
		peerconnection := session.peerconnection

		err := peerconnection.AddCandidate(resp.Candidate)
		if err != nil {
			log.Printf("Error: Cannot add IceCandidate of client: %s", resp.Id)
		}
	} else {
		log.Printf("Error: No session for ID: %s\n", resp.Id)
	}
}

func (c *Coordinator) HandleGameStart(packet ipc.InPacket, h *Handler) {
	log.Println("Received a start request from coordinator")
	var resp api.StartGameRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.Printf("error: broken game start request %v", err)
		_ = h.cord.wire.SendPacket(ipc.OutPacket{Id: packet.Id, T: packet.T, Payload: ""})
		return
	}
	session := h.getSession(resp.Id)
	if session == nil {
		log.Printf("Error: No session for ID: %s\n", resp.Id)
		_ = h.cord.wire.SendPacket(ipc.OutPacket{Id: packet.Id, T: packet.T, Payload: ""})
		return
	}

	peerconnection := session.peerconnection
	gameMeta := games.GameMetadata{Name: resp.Game.Name, Type: resp.Game.Type, Path: resp.Game.Path}
	gameRoom := h.startGameHandler(gameMeta, resp.RoomId, resp.PlayerIndex, peerconnection)
	session.RoomID = gameRoom.ID
	// TODO: can data race
	h.rooms[gameRoom.ID] = gameRoom

	_ = h.cord.wire.SendPacket(ipc.OutPacket{Id: packet.Id, T: packet.T, Payload: api.StartGameResponse{
		RoomId: gameRoom.ID,
	}})
}

// startGameHandler starts a game if roomID is given, if not create new room
func (h *Handler) startGameHandler(game games.GameMetadata, existedRoomID string, playerIndex int, peerconnection *webrtc.WebRTC) *room.Room {
	log.Printf("Loading game: %v\n", game.Name)
	// If we are connecting to coordinator, request corresponding serverID based on roomID
	// TODO: check if existedRoomID is in the current server
	gameRoom := h.getRoom(existedRoomID)
	// If room is not running
	if gameRoom == nil {
		log.Println("Got Room from local ", gameRoom, " ID: ", existedRoomID)
		// Create new room and update player index
		gameRoom = h.createRoom(existedRoomID, game)
		gameRoom.UpdatePlayerIndex(peerconnection, playerIndex)

		// Wait for done signal from room
		go func() {
			<-gameRoom.Done
			h.detachRoom(gameRoom.ID)
			// TODO add proper non-crash close logic
			// send signal to coordinator that the room is closed, coordinator will remove that room
			h.cord.CloseRoom(gameRoom.ID)
		}()
	}

	// Attach peerconnection to room. If PC is already in room, don't detach
	log.Println("Is PC in room", gameRoom.IsPCInRoom(peerconnection))
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
	log.Println("Received a quit request from coordinator")
	var resp api.GameQuitRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.Printf("error: broken game quit request %v", err)
		return
	}
	session := h.getSession(resp.Id)

	if session != nil {
		// Defensive coding, check if the peerconnection is in room
		if h.getRoom(session.RoomID).IsPCInRoom(session.peerconnection) {
			h.detachPeerConn(session.peerconnection)
		}
	} else {
		log.Printf("Error: No session for ID: %s\n", resp.Id)
	}
}

func (c *Coordinator) HandleSaveGame(packet ipc.InPacket, h *Handler) {
	c.Println("Received a save game from coordinator")
	var resp api.SaveGameRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.Printf("error: broken game save request %v", err)
		return
	}
	c.Println("RoomID:", resp.RoomId)
	rez := "ok"
	if resp.RoomId != "" {
		gameRoom := h.getRoom(resp.RoomId)
		if gameRoom == nil {
			return
		}
		err := gameRoom.SaveGame()
		if err != nil {
			log.Println("[!] Cannot save game state: ", err)
			rez = "error"
		}
	} else {
		rez = "error"
	}

	_ = h.cord.wire.SendPacket(ipc.OutPacket{Id: packet.Id, T: packet.T, Payload: api.SaveGameResponse(rez)})
}

func (c *Coordinator) HandleLoadGame(packet ipc.InPacket, h *Handler) {
	log.Println("Received a load game from coordinator")
	log.Println("Loading game state")
	var resp api.LoadGameRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.Printf("error: broken game load request %v", err)
		return
	}
	rez := "ok"
	if resp.RoomId != "" {
		err := h.getRoom(resp.RoomId).LoadGame()
		if err != nil {
			log.Println("[!] Cannot load game state: ", err)
			rez = "error"
		}
	} else {
		rez = "error"
	}
	_ = h.cord.wire.SendPacket(ipc.OutPacket{Id: packet.Id, T: packet.T, Payload: api.SaveGameResponse(rez)})
}

func (c *Coordinator) HandleChangePlayer(packet ipc.InPacket, h *Handler) {
	log.Println("Received an update player index event from coordinator")
	var resp api.ChangePlayerRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.Printf("error: broken change player request %v", err)
		return
	}

	session := h.getSession(resp.Id)
	idx, err := strconv.Atoi(resp.Index)
	gameRoom := h.getRoom(resp.RoomId)

	var rez api.ChangePlayerResponse
	if gameRoom != nil && session != nil && err == nil {
		gameRoom.UpdatePlayerIndex(session.peerconnection, idx)
		rez = strconv.Itoa(idx)
	} else {
		rez = "error"
	}
	_ = h.cord.wire.SendPacket(ipc.OutPacket{Id: packet.Id, T: packet.T, Payload: rez})
}

func (c *Coordinator) HandleToggleMultitap(packet ipc.InPacket, h *Handler) {
	log.Println("Received a multitap toggle from coordinator")
	var resp api.ToggleMultitapRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.Printf("error: broken toggle multitap request %v", err)
		return
	}

	rez := "ok"
	if resp.RoomId != "" {
		err := h.getRoom(resp.RoomId).ToggleMultitap()
		if err != nil {
			log.Println("[!] Could not toggle multitap state: ", err)
			rez = "error"
		}
	} else {
		rez = "error"
	}
	_ = h.cord.wire.SendPacket(ipc.OutPacket{Id: packet.Id, T: packet.T, Payload: rez})
}
