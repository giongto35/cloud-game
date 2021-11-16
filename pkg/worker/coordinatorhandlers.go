package worker

import (
	"encoding/json"
	"strconv"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/ipc"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
)

func MakeConnectionRequest(conf worker.Worker, address string) (string, error) {
	return toBase64Json(api.ConnectionRequest{
		Zone:     conf.Network.Zone,
		PingAddr: conf.GetPingAddr(address),
		IsHTTPS:  conf.Server.Https,
	})
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
		h.removeUser(session)
	} else {
		c.log.Warn().Msgf("No session for id [%v]", resp.Id)
	}
}

func (c *Coordinator) HandleWebrtcInit(packet ipc.InPacket, h *Handler, connApi *webrtc.ApiFactory) {
	resp, err := c.webrtcInit(packet.Payload)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed WebRTC init request")
		return
	}
	enc := h.cfg.Encoder
	peer := webrtc.NewWebRTC(h.cfg.Webrtc, c.log, connApi)
	localSDP, err := peer.NewCall(enc.Video.Codec, enc.Audio.Codec, func(data interface{}) {
		candidate, err := toBase64Json(data)
		if err != nil {
			c.log.Error().Err(err).Msgf("ICE candidate encode fail for [%v]", data)
			return
		}
		h.cord.IceCandidate(candidate, string(resp.Id))
	})
	if err != nil {
		c.log.Error().Err(err).Msg("cannot create new webrtc session")
		_ = h.cord.SendPacket(packet.Proxy(ipc.EmptyPacket))
		return
	}
	sdp, err := toBase64Json(localSDP)
	if err != nil {
		c.log.Error().Err(err).Msgf("SDP encode fail fro [%v]", localSDP)
		_ = h.cord.SendPacket(packet.Proxy(ipc.EmptyPacket))
		return
	}

	session := NewSession(peer)
	h.sessions.Add(resp.Id, session)
	c.log.Info().Str("id", resp.Id.Short()).Msgf("Peer connection (uid:%s)", session.GetId())

	_ = h.cord.SendPacket(packet.Proxy(sdp))
}

func (c *Coordinator) HandleWebrtcAnswer(packet ipc.InPacket, h *Handler) {
	var resp api.WebrtcAnswerRequest
	if err := fromJson(packet.Payload, &resp); err != nil {
		c.log.Error().Err(err).Msg("malformed WebRTC answer")
		return
	}
	if session := h.sessions.Get(resp.Id); session != nil {
		if err := session.GetPeerConn().SetRemoteSDP(resp.Sdp, fromBase64Json); err != nil {
			c.log.Error().Err(err).Msgf("cannot set remote SDP of client [%v]", resp.Id)
		}
	} else {
		c.log.Error().Msgf("no session for id [%v]", resp.Id)
	}
}

func (c *Coordinator) HandleWebrtcIceCandidate(packet ipc.InPacket, h *Handler) {
	var resp api.WebrtcIceCandidateRequest
	if err := fromJson(packet.Payload, &resp); err != nil {
		c.log.Error().Err(err).Msg("malformed WebRTC candidate request")
		return
	}
	if session := h.sessions.Get(resp.Id); session != nil {
		if err := session.GetPeerConn().AddCandidate(resp.Candidate, fromBase64Json); err != nil {
			c.log.Error().Err(err).Msgf("cannot add Ice candidate of client [%v]", resp.Id)
		}
	} else {
		c.log.Error().Msgf("no session for id [%v]", resp.Id)
	}
}

func (c *Coordinator) HandleGameStart(packet ipc.InPacket, h *Handler) {
	var resp api.StartGameRequest
	if err := fromJson(packet.Payload, &resp); err != nil {
		c.log.Error().Err(err).Msg("malformed game start request")
		_ = h.cord.SendPacket(packet.Proxy(ipc.EmptyPacket))
		return
	}
	session := h.sessions.Get(resp.Stateful.Id)
	if session == nil {
		c.log.Error().Msgf("no session [%v]", resp.Stateful.Id)
		_ = h.cord.SendPacket(packet.Proxy(ipc.EmptyPacket))
		return
	}
	h.log.Info().Str("game", resp.Game.Name).Msg("Starting the game")
	// trying to find existing room with that id
	playRoom := h.rooms.Get(resp.Room.Id)
	if playRoom == nil {
		h.log.Info().Str("room", resp.Room.Id).Msg("Create room")
		playRoom = h.createRoom(
			resp.Room.Id,
			games.GameMetadata{Name: resp.Game.Name, Base: resp.Game.Base, Type: resp.Game.Type, Path: resp.Game.Path},
			func(room *Room) {
				h.rooms.Remove(room.ID)
				// send signal to coordinator that the room is closed, coordinator will remove that room
				h.cord.CloseRoom(room.ID)
				h.log.Debug().Msgf("Room close has been called %v", room.ID)
			},
		)
		session.SetPlayerIndex(resp.PlayerIndex)
		h.log.Info().Msgf("Updated player index to: %d", resp.PlayerIndex)
	}
	// Attach peerconnection to room. If PC is already in room, don't detach
	userInRoom := playRoom.HasUser(session)
	if !userInRoom {
		h.log.Info().Msgf("The peer is not in the room: %v", userInRoom)
		h.removeUser(session)
		playRoom.AddUser(session)
		playRoom.PollUserInput(session)
	}
	// Register room to coordinator if we are connecting to coordinator
	if playRoom != nil {
		h.cord.RegisterRoom(playRoom.ID)
	}
	session.SetRoom(playRoom)
	h.rooms.Add(playRoom)
	_ = h.cord.SendPacket(packet.Proxy(api.StartGameResponse{Room: api.Room{Id: playRoom.ID}}))
}

func (c *Coordinator) HandleQuitGame(packet ipc.InPacket, h *Handler) {
	var resp api.GameQuitRequest
	if err := fromJson(packet.Payload, &resp); err != nil {
		c.log.Error().Err(err).Msg("malformed game quit request")
		return
	}
	if session := h.sessions.Get(resp.Stateful.Id); session != nil {
		rm := h.rooms.Get(session.GetId())
		// Defensive coding, check if the peerconnection is in room
		if rm != nil && rm.HasUser(session) {
			h.removeUser(session)
		}
	} else {
		c.log.Error().Msgf("no session for id [%v]", resp.Stateful.Id)
	}
}

func (c *Coordinator) HandleSaveGame(packet ipc.InPacket, h *Handler) {
	var resp api.SaveGameRequest
	if err := fromJson(packet.Payload, &resp); err != nil {
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
	if err := fromJson(packet.Payload, &resp); err != nil {
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
	if err := fromJson(packet.Payload, &resp); err != nil {
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
		session.SetPlayerIndex(idx)
		h.log.Info().Msgf("Updated player index to: %d", idx)
		rez = strconv.Itoa(idx)
	} else {
		rez = ipc.ErrPacket
	}
	_ = h.cord.SendPacket(packet.Proxy(rez))
}

func (c *Coordinator) HandleToggleMultitap(packet ipc.InPacket, h *Handler) {
	var resp api.ToggleMultitapRequest
	if err := fromJson(packet.Payload, &resp); err != nil {
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
