package worker

import (
	"encoding/base64"
	"encoding/json"
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
		c.log.Error().Err(err).Msg("terminate session error")
		return
	}
	c.log.Info().Msgf("Received a terminate session [%v]", resp.Id)
	session := h.sessions.Get(resp.Id)
	if session != nil {
		session.Close()
		h.sessions.Remove(resp.Id)
		h.detachPeerConn(session.peerconnection)
	} else {
		c.log.Warn().Msgf("No session for id [%v]", resp.Id)
	}
}

func (c *Coordinator) HandleWebrtcInit(packet ipc.InPacket, h *Handler) {
	resp, err := c.webrtcInit(packet.Payload)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed WebRTC init request")
		return
	}

	peerconnection, err := webrtc.NewWebRTC(c.log)
	if err != nil {
		c.log.Error().Err(err).Msg("WebRTC connection init fail")
	}
	peerconnection = peerconnection.WithConfig(webrtcConf.Config{Encoder: h.cfg.Encoder, Webrtc: h.cfg.Webrtc})
	localSDP, err := peerconnection.StartClient(func(cd string) { h.cord.IceCandidate(cd, string(resp.Id)) })
	if err != nil {
		c.log.Error().Err(err).Msg("cannot create new webrtc session")
		_ = h.cord.SendPacket(packet.Proxy(ipc.EmptyPacket))
	}

	// Create new sessions when we have new peerconnection initialized
	h.sessions.Add(resp.Id, &Session{peerconnection: peerconnection})
	c.log.Info().Msgf("Start peerconnection [%v]", resp.Id)

	_ = h.cord.SendPacket(packet.Proxy(localSDP))
}

func (c *Coordinator) HandleWebrtcAnswer(packet ipc.InPacket, h *Handler) {
	var resp api.WebrtcAnswerRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed WebRTC answer")
		return
	}
	if session := h.sessions.Get(resp.Id); session != nil {
		if err := session.peerconnection.SetRemoteSDP(resp.Sdp); err != nil {
			c.log.Error().Err(err).Msgf("cannot set remote SDP of client [%v]", resp.Id)
		}
	} else {
		c.log.Error().Msgf("no session for id [%v]", resp.Id)
	}
}

func (c *Coordinator) HandleWebrtcIceCandidate(packet ipc.InPacket, h *Handler) {
	var resp api.WebrtcIceCandidateRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed WebRTC candidate request")
		return
	}
	if session := h.sessions.Get(resp.Id); session != nil {
		if err := session.peerconnection.AddCandidate(resp.Candidate); err != nil {
			c.log.Error().Err(err).Msgf("cannot add Ice candidate of client [%v]", resp.Id)
		}
	} else {
		c.log.Error().Msgf("no session for id [%v]", resp.Id)
	}
}

func (c *Coordinator) HandleGameStart(packet ipc.InPacket, h *Handler) {
	var resp api.StartGameRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed game start request")
		_ = h.cord.SendPacket(packet.Proxy(ipc.EmptyPacket))
		return
	}
	session := h.sessions.Get(resp.Stateful.Id)
	if session == nil {
		c.log.Error().Msgf("no session for id [%v]", resp.Stateful.Id)
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
	h.log.Info().Str("game", game.Name).Msg("Start load game")
	// If we are connecting to coordinator, request corresponding serverID based on roomID
	// TODO: check if existedRoomID is in the current server
	gameRoom := h.rooms.Get(existedRoomID)
	// If room is not running
	if gameRoom == nil {
		h.log.Info().Str("room", existedRoomID).Msg("Create room")
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
	h.log.Info().Msgf("The peer is in the room: %v", gameRoom.IsPCInRoom(peerconnection))
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
		c.log.Error().Err(err).Msg("malformed game quit request")
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
		c.log.Error().Msgf("no session for id [%v]", resp.Stateful.Id)
	}
}

func (c *Coordinator) HandleSaveGame(packet ipc.InPacket, h *Handler) {
	var resp api.SaveGameRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed game save request")
		return
	}
	c.log.Info().Str("room", resp.Room.Id).Msg("Got room")
	rez := ipc.OkPacket
	if resp.Room.Id != "" {
		rm := h.rooms.Get(resp.Room.Id)
		if rm == nil {
			return
		}
		err := rm.SaveGame()
		if err != nil {
			c.log.Error().Err(err).Msg("cannot save game state")
			rez = ipc.ErrPacket
		}
	} else {
		rez = ipc.ErrPacket
	}
	_ = h.cord.SendPacket(packet.Proxy(rez))
}

func (c *Coordinator) HandleLoadGame(packet ipc.InPacket, h *Handler) {
	c.log.Info().Msg("Loading game state")
	var resp api.LoadGameRequest
	err := fromJson(packet.Payload, &resp)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed game load request")
		return
	}
	rez := ipc.OkPacket
	if resp.Room.Id != "" {
		rm := h.rooms.Get(resp.Room.Id)
		if rm == nil {
			return
		}
		if err := rm.LoadGame(); err != nil {
			c.log.Error().Err(err).Msg("cannot load game state")
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
		c.log.Error().Err(err).Msg("malformed change player request")
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
		c.log.Error().Err(err).Msg("malformed toggle multitap request")
		return
	}

	rez := ipc.OkPacket
	if resp.Room.Id != "" {
		rm := h.rooms.Get(resp.Room.Id)
		if rm == nil {
			return
		}
		if err := rm.ToggleMultitap(); err != nil {
			c.log.Error().Err(err).Msg("could not toggle multitap state")
			rez = ipc.ErrPacket
		}
	} else {
		rez = ipc.ErrPacket
	}
	_ = h.cord.SendPacket(packet.Proxy(rez))
}
