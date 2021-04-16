package worker

import (
	"log"
	"strconv"

	webrtcConfig "github.com/giongto35/cloud-game/v2/pkg/config/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/cws"
	"github.com/giongto35/cloud-game/v2/pkg/cws/api"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
	"github.com/giongto35/cloud-game/v2/pkg/worker/room"
)

func (h *Handler) handleServerId() cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Printf("[worker] new id: %s", resp.Data)
		h.serverID = resp.Data
		// unlock worker if it's locked
		h.w.lock.Unlock()
		return cws.EmptyPacket
	}
}

func (h *Handler) handleTerminateSession() cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Received a terminate session ", resp.SessionID)
		session := h.getSession(resp.SessionID)
		if session != nil {
			session.Close()
			delete(h.sessions, resp.SessionID)
			h.detachPeerConn(session.peerconnection)
		} else {
			log.Printf("Error: No session for ID: %s\n", resp.SessionID)
		}
		return cws.EmptyPacket
	}
}

func (h *Handler) handleInitWebrtc() cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Received a request to createOffer from browser via coordinator")

		peerconnection := webrtc.NewWebRTC().WithConfig(
			webrtcConfig.Config{Encoder: h.cfg.Encoder, Webrtc: h.cfg.Webrtc},
		)

		localSDP, err := peerconnection.StartClient(
			// send back candidate string to browser
			func(cd string) { h.oClient.Send(api.IceCandidatePacket(cd, resp.SessionID), nil) },
		)

		if err != nil {
			log.Println("Error: Cannot create new webrtc session", err)
			return cws.EmptyPacket
		}

		// Create new sessions when we have new peerconnection initialized
		h.sessions[resp.SessionID] = &Session{peerconnection: peerconnection}
		log.Println("Start peerconnection", resp.SessionID)

		return api.OfferPacket(localSDP)
	}
}

func (h *Handler) handleAnswer() cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Received answer SDP from browser")
		session := h.getSession(resp.SessionID)
		if session != nil {
			peerconnection := session.peerconnection
			err := peerconnection.SetRemoteSDP(resp.Data)
			if err != nil {
				log.Printf("Error: cannot set RemoteSDP of client: %v beacuse %v", resp.SessionID, err)
			}
		} else {
			log.Printf("Error: No session for ID: %s\n", resp.SessionID)
		}
		return cws.EmptyPacket
	}
}

func (h *Handler) handleIceCandidate() cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Received remote Ice Candidate from browser")
		session := h.getSession(resp.SessionID)

		if session != nil {
			peerconnection := session.peerconnection

			err := peerconnection.AddCandidate(resp.Data)
			if err != nil {
				log.Printf("Error: Cannot add IceCandidate of client: %s", resp.SessionID)
			}
		} else {
			log.Printf("Error: No session for ID: %s\n", resp.SessionID)
		}
		return cws.EmptyPacket
	}
}

func (h *Handler) handleGameStart() cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Received a start request from coordinator")
		session := h.getSession(resp.SessionID)
		if session == nil {
			log.Printf("Error: No session for ID: %s\n", resp.SessionID)
			return cws.EmptyPacket
		}

		peerconnection := session.peerconnection
		// TODO: Standardize for all types of packet. Make WSPacket generic
		startPacket := api.GameStartCall{}
		if err := startPacket.From(resp.Data); err != nil {
			return cws.EmptyPacket
		}
		gameMeta := games.GameMetadata{
			Name: startPacket.Name,
			Type: startPacket.Type,
			Path: startPacket.Path,
		}

		gameRoom := h.startGameHandler(gameMeta, resp.RoomID, resp.PlayerIndex, peerconnection)
		session.RoomID = gameRoom.ID
		// TODO: can data race
		h.rooms[gameRoom.ID] = gameRoom

		return api.StartPacket(gameRoom.ID)
	}
}

func (h *Handler) handleGameQuit() cws.PacketHandler {
	return func(resp cws.WSPacket) cws.WSPacket {
		log.Println("Received a quit request from coordinator")
		session := h.getSession(resp.SessionID)

		if session != nil {
			// Defensive coding, check if the peerconnection is in room
			if h.getRoom(session.RoomID).IsPCInRoom(session.peerconnection) {
				h.detachPeerConn(session.peerconnection)
			}
		} else {
			log.Printf("Error: No session for ID: %s\n", resp.SessionID)
		}
		return cws.EmptyPacket
	}
}

func (h *Handler) handleGameSave() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received a save game from coordinator")
		log.Println("RoomID:", resp.RoomID)
		req.ID = api.GameSave
		req.Data = "ok"
		if resp.RoomID != "" {
			gameRoom := h.getRoom(resp.RoomID)
			if gameRoom == nil {
				return
			}
			err := gameRoom.SaveGame()
			if err != nil {
				log.Println("[!] Cannot save game state: ", err)
				req.Data = "error"
			}
		} else {
			req.Data = "error"
		}
		return req
	}
}

func (h *Handler) handleGameLoad() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received a load game from coordinator")
		log.Println("Loading game state")
		req.ID = api.GameLoad
		req.Data = "ok"
		if resp.RoomID != "" {
			err := h.getRoom(resp.RoomID).LoadGame()
			if err != nil {
				log.Println("[!] Cannot load game state: ", err)
				req.Data = "error"
			}
		} else {
			req.Data = "error"
		}
		return req
	}
}

func (h *Handler) handleGamePlayerSelect() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received an update player index event from coordinator")
		req.ID = api.GamePlayerSelect

		session := h.getSession(resp.SessionID)
		idx, err := strconv.Atoi(resp.Data)
		gameRoom := h.getRoom(resp.RoomID)

		if gameRoom != nil && session != nil && err == nil {
			gameRoom.UpdatePlayerIndex(session.peerconnection, idx)
			req.Data = strconv.Itoa(idx)
		} else {
			req.Data = "error"
		}
		return req
	}
}

func (h *Handler) handleGameMultitap() cws.PacketHandler {
	return func(resp cws.WSPacket) (req cws.WSPacket) {
		log.Println("Received a multitap toggle from coordinator")
		req.ID = api.GameMultitap
		req.Data = "ok"
		if resp.RoomID != "" {
			err := h.getRoom(resp.RoomID).ToggleMultitap()
			if err != nil {
				log.Println("[!] Could not toggle multitap state: ", err)
				req.Data = "error"
			}
		} else {
			req.Data = "error"
		}
		return req
	}
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
			// send signal to coordinator that the room is closed, coordinator will remove that room
			h.oClient.Send(api.CloseRoomPacket(gameRoom.ID), nil)
		}()
	}

	// Attach peerconnection to room. If PC is already in room, don't detach
	log.Println("Is PC in room", gameRoom.IsPCInRoom(peerconnection))
	if !gameRoom.IsPCInRoom(peerconnection) {
		h.detachPeerConn(peerconnection)
		gameRoom.AddConnectionToRoom(peerconnection)
	}

	// Register room to coordinator if we are connecting to coordinator
	if gameRoom != nil && h.oClient != nil {
		h.oClient.Send(api.RegisterRoomPacket(gameRoom.ID), nil)
	}
	return gameRoom
}
