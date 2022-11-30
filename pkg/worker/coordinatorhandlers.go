package worker

import (
	"strconv"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/client"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/webrtc"
)

func MakeConnectionRequest(id string, conf worker.Worker, address string) (string, error) {
	addr := conf.GetPingAddr(address)
	return toBase64Json(api.ConnectionRequest{
		Addr:    addr.Hostname(),
		Id:      id,
		IsHTTPS: conf.Server.Https,
		PingURL: addr.String(),
		Port:    conf.GetPort(address),
		Tag:     conf.Tag,
		Zone:    conf.Network.Zone,
	})
}

func (c *Coordinator) HandleTerminateSession(rq api.TerminateSessionRequest, h *Handler) {
	if session := h.router.GetUser(rq.Id); session != nil {
		h.TerminateSession(session)
	}
}

func (c *Coordinator) HandleWebrtcInit(packet client.In, h *Handler, connApi *webrtc.ApiFactory) {
	resp, err := c.webrtcInit(packet.Payload)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed WebRTC init request")
		return
	}
	enc := h.conf.Encoder
	peer := webrtc.NewWebRTC(h.conf.Webrtc, c.log, connApi)
	localSDP, err := peer.NewCall(enc.Video.Codec, enc.Audio.Codec, func(data any) {
		candidate, err := toBase64Json(data)
		if err != nil {
			c.log.Error().Err(err).Msgf("ICE candidate encode fail for [%v]", data)
			return
		}
		h.cord.IceCandidate(candidate, resp.Id)
	})
	if err != nil {
		c.log.Error().Err(err).Msg("cannot create new webrtc session")
		_ = h.cord.Route(packet, client.EmptyPacket)
		return
	}
	sdp, err := toBase64Json(localSDP)
	if err != nil {
		c.log.Error().Err(err).Msgf("SDP encode fail fro [%v]", localSDP)
		_ = h.cord.Route(packet, client.EmptyPacket)
		return
	}

	// use user uid from the coordinator
	user := NewSession(peer, resp.Id)
	h.router.AddUser(user)
	c.log.Info().Str("id", string(resp.Id)).Msgf("Peer connection (uid:%s)", user.GetId())
	_ = h.cord.Route(packet, sdp)
}

func (c *Coordinator) HandleWebrtcAnswer(rq api.WebrtcAnswerRequest, h *Handler) {
	if user := h.router.GetUser(rq.Id); user != nil {
		if err := user.GetPeerConn().SetRemoteSDP(rq.Sdp, fromBase64Json); err != nil {
			c.log.Error().Err(err).Msgf("cannot set remote SDP of client [%v]", rq.Id)
		}
	}
}

func (c *Coordinator) HandleWebrtcIceCandidate(rs api.WebrtcIceCandidateRequest, h *Handler) {
	if user := h.router.GetUser(rs.Id); user != nil {
		if err := user.GetPeerConn().AddCandidate(rs.Candidate, fromBase64Json); err != nil {
			c.log.Error().Err(err).Msgf("cannot add ICE candidate of the client [%v]", rs.Id)
		}
	}
}

func (c *Coordinator) HandleGameStart(packet client.In, h *Handler) {
	rq, err := api.Unwrap[api.StartGameRequest](packet.Payload)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed game start request")
		_ = h.cord.Route(packet, client.EmptyPacket)
		return
	}
	resp := *rq
	user := h.router.GetUser(resp.Stateful.Id)
	if user == nil {
		c.log.Error().Msgf("no user [%v]", resp.Stateful.Id)
		_ = h.cord.Route(packet, client.EmptyPacket)
		return
	}
	h.log.Info().Str("game", resp.Game.Name).Msg("Starting the game")
	// trying to find existing room with that id
	playRoom := h.router.GetRoom(resp.Room.Id)
	if playRoom == nil {
		h.log.Info().Str("room", resp.Room.Id).Msg("Create room")

		// recording
		if h.conf.Recording.Enabled {
			h.log.Info().Msgf("RECORD: %v %v", resp.Record, resp.RecordUser)
		}

		playRoom = h.CreateRoom(
			resp.Room.Id,
			games.GameMetadata{Name: resp.Game.Name, Base: resp.Game.Base, Type: resp.Game.Type, Path: resp.Game.Path},
			resp.Record, resp.RecordUser,
			func(room *Room) {
				h.router.RemoveRoom(room)
				// send signal to coordinator that the room is closed, coordinator will remove that room
				h.cord.CloseRoom(room.ID)
				h.log.Debug().Msgf("Room close has been called %v", room.ID)
			},
		)
		h.router.AddRoom(playRoom)
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
	if playRoom == nil {
		c.log.Error().Msgf("couldn't create a room [%v]", resp.Stateful.Id)
		_ = h.cord.Route(packet, client.EmptyPacket)
		return
	}
	h.cord.RegisterRoom(playRoom.ID)
	user.SetRoom(playRoom)
	h.router.AddRoom(playRoom)
	_ = h.cord.Route(packet, api.StartGameResponse{Room: api.Room{Id: playRoom.ID}, Record: h.conf.Recording.Enabled})
}

func (c *Coordinator) HandleQuitGame(rq api.GameQuitRequest, h *Handler) {
	if user := h.router.GetUser(rq.Stateful.Id); user != nil {
		if room := h.router.GetRoom(rq.Room.Id); room != nil {
			if room.HasUser(user) {
				h.removeUser(user)
			}
		}
	}
}

func (c *Coordinator) HandleSaveGame(packet client.In, h *Handler) {
	resp, err := api.Unwrap[api.SaveGameRequest](packet.Payload)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed game save request")
		return
	}
	c.log.Info().Str("room", resp.Room.Id).Msg("Got room")
	rez := client.OkPacket
	if resp.Room.Id != "" {
		room := h.router.GetRoom(resp.Room.Id)
		if room == nil {
			return
		}
		err := room.SaveGame()
		if err != nil {
			c.log.Error().Err(err).Msg("cannot save game state")
			rez = client.ErrPacket
		}
	} else {
		rez = client.ErrPacket
	}
	_ = h.cord.Route(packet, rez)
}

func (c *Coordinator) HandleLoadGame(packet client.In, h *Handler) {
	c.log.Info().Msg("Loading game state")
	resp, err := api.Unwrap[api.LoadGameRequest](packet.Payload)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed game load request")
		return
	}
	rez := client.OkPacket
	if resp.Room.Id != "" {
		rm := h.router.GetRoom(resp.Room.Id)
		if rm == nil {
			return
		}
		if err := rm.LoadGame(); err != nil {
			c.log.Error().Err(err).Msg("cannot load game state")
			rez = client.ErrPacket
		}
	} else {
		rez = client.ErrPacket
	}
	_ = h.cord.Route(packet, rez)
}

func (c *Coordinator) HandleChangePlayer(packet client.In, h *Handler) {
	resp, err := api.Unwrap[api.ChangePlayerRequest](packet.Payload)
	if err != nil {
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
		rez = client.ErrPacket
	}
	_ = h.cord.Route(packet, rez)
}

func (c *Coordinator) HandleToggleMultitap(packet client.In, h *Handler) {
	resp, err := api.Unwrap[api.ToggleMultitapRequest](packet.Payload)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed toggle multitap request")
		return
	}
	rez := client.OkPacket
	if resp.Room.Id != "" {
		room := h.router.GetRoom(resp.Room.Id)
		if room == nil {
			return
		}
		room.ToggleMultitap()
	} else {
		rez = client.ErrPacket
	}
	_ = h.cord.Route(packet, rez)
}

func (c *Coordinator) HandleRecordGame(packet client.In, h *Handler) {
	var rez = client.OkPacket
	defer func() {
		_ = h.cord.Route(packet, rez)
	}()

	resp, err := api.Unwrap[api.RecordGameRequest](packet.Payload)
	if err != nil {
		c.log.Error().Err(err).Msg("malformed record game request")
		rez = client.ErrPacket
		return
	}

	if !h.conf.Recording.Enabled {
		rez = client.ErrPacket
		return
	}

	if resp.Room.Id != "" {
		room := h.router.GetRoom(resp.Room.Id)
		if room == nil {
			return
		}
		room.ToggleRecording(resp.Active, resp.User)
	} else {
		rez = client.ErrPacket
		return
	}
}
