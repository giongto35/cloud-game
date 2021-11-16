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
	if session := h.router.GetUser(resp.Id); session != nil {
		h.TerminateSession(session)
	}
}

func (c *Coordinator) HandleWebrtcInit(packet ipc.InPacket, h *Handler, connApi *webrtc.ApiFactory) {
	resp, err := c.webrtcInit(packet.Payload)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed WebRTC init request")
		return
	}
	enc := h.conf.Encoder
	peer := webrtc.NewWebRTC(h.conf.Webrtc, c.log, connApi)
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

	// use user uid from the coordinator
	user := NewSession(peer, resp.Id)
	h.router.AddUser(user)
	c.log.Info().Str("id", resp.Id.Short()).Msgf("Peer connection (uid:%s)", user.GetId())

	_ = h.cord.SendPacket(packet.Proxy(sdp))
}

func (c *Coordinator) HandleWebrtcAnswer(packet ipc.InPacket, h *Handler) {
	var resp api.WebrtcAnswerRequest
	if err := fromJson(packet.Payload, &resp); err != nil {
		c.log.Error().Err(err).Msg("malformed WebRTC answer")
		return
	}
	if user := h.router.GetUser(resp.Id); user != nil {
		if err := user.GetPeerConn().SetRemoteSDP(resp.Sdp, fromBase64Json); err != nil {
			c.log.Error().Err(err).Msgf("cannot set remote SDP of client [%v]", resp.Id)
		}
	}
}

func (c *Coordinator) HandleWebrtcIceCandidate(packet ipc.InPacket, h *Handler) {
	var resp api.WebrtcIceCandidateRequest
	if err := fromJson(packet.Payload, &resp); err != nil {
		c.log.Error().Err(err).Msg("malformed WebRTC candidate request")
		return
	}
	if user := h.router.GetUser(resp.Id); user != nil {
		if err := user.GetPeerConn().AddCandidate(resp.Candidate, fromBase64Json); err != nil {
			c.log.Error().Err(err).Msgf("cannot add Ice candidate of client [%v]", resp.Id)
		}
	}
}

func (c *Coordinator) HandleGameStart(packet ipc.InPacket, h *Handler) {
	var resp api.StartGameRequest
	if err := fromJson(packet.Payload, &resp); err != nil {
		c.log.Error().Err(err).Msg("malformed game start request")
		_ = h.cord.SendPacket(packet.Proxy(ipc.EmptyPacket))
		return
	}
	user := h.router.GetUser(resp.Stateful.Id)
	if user == nil {
		c.log.Error().Msgf("no user [%v]", resp.Stateful.Id)
		_ = h.cord.SendPacket(packet.Proxy(ipc.EmptyPacket))
		return
	}
	h.log.Info().Str("game", resp.Game.Name).Msg("Starting the game")
	// trying to find existing room with that id
	playRoom := h.router.GetRoom(resp.Room.Id)
	if playRoom == nil {
		h.log.Info().Str("room", resp.Room.Id).Msg("Create room")
		playRoom = h.createRoom(
			resp.Room.Id,
			games.GameMetadata{Name: resp.Game.Name, Base: resp.Game.Base, Type: resp.Game.Type, Path: resp.Game.Path},
			func(room *Room) {
				h.router.RemoveRoom(room)
				// send signal to coordinator that the room is closed, coordinator will remove that room
				h.cord.CloseRoom(room.ID)
				h.log.Debug().Msgf("Room close has been called %v", room.ID)
			},
		)
		user.SetPlayerIndex(resp.PlayerIndex)
		h.log.Info().Msgf("Updated player index to: %d", resp.PlayerIndex)
	}
	// Attach peerconnection to room. If PC is already in room, don't detach
	userInRoom := playRoom.HasUser(user)
	if !userInRoom {
		h.log.Info().Msgf("The peer is not in the room: %v", userInRoom)
		h.removeUser(user)
		playRoom.AddUser(user)
		playRoom.PollUserInput(user)
	}
	// Register room to coordinator if we are connecting to coordinator
	if playRoom != nil {
		h.cord.RegisterRoom(playRoom.ID)
	}
	user.SetRoom(playRoom)
	h.router.AddRoom(playRoom)
	_ = h.cord.SendPacket(packet.Proxy(api.StartGameResponse{Room: api.Room{Id: playRoom.ID}}))
}

func (c *Coordinator) HandleQuitGame(packet ipc.InPacket, h *Handler) {
	var resp api.GameQuitRequest
	if err := fromJson(packet.Payload, &resp); err != nil {
		c.log.Error().Err(err).Msg("malformed game quit request")
		return
	}
	if user := h.router.GetUser(resp.Stateful.Id); user != nil {
		if room := h.router.GetRoom(resp.Room.Id); room != nil {
			if room.HasUser(user) {
				h.removeUser(user)
			}
		}
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
		room := h.router.GetRoom(resp.Room.Id)
		if room == nil {
			return
		}
		err := room.SaveGame()
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
		rm := h.router.GetRoom(resp.Room.Id)
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
	user := h.router.GetUser(resp.Stateful.Id)
	idx, err := strconv.Atoi(resp.Index)
	if h.router.GetRoom(resp.Room.Id) == nil {
		return
	}
	var rez api.ChangePlayerResponse
	if user != nil && err == nil {
		user.SetPlayerIndex(idx)
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
		room := h.router.GetRoom(resp.Room.Id)
		if room == nil {
			return
		}
		if err := room.ToggleMultitap(); err != nil {
			c.log.Error().Err(err).Msg("could not toggle multitap state")
			rez = ipc.ErrPacket
		}
	} else {
		rez = ipc.ErrPacket
	}
	_ = h.cord.SendPacket(packet.Proxy(rez))
}
