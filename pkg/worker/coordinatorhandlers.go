package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/comm"
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

func (c *Coordinator) HandleWebrtcInit(rq api.WebrtcInitRequest, h *Handler, connApi *webrtc.ApiFactory) comm.Out {
	enc := h.conf.Encoder
	peer := webrtc.NewWebRTC(h.conf.Webrtc, c.Log, connApi)
	localSDP, err := peer.NewCall(enc.Video.Codec, enc.Audio.Codec, func(data any) {
		candidate, err := toBase64Json(data)
		if err != nil {
			c.Log.Error().Err(err).Msgf("ICE candidate encode fail for [%v]", data)
			return
		}
		h.cord.IceCandidate(candidate, rq.Id)
	})
	if err != nil {
		c.Log.Error().Err(err).Msg("cannot create new webrtc session")
		return comm.EmptyPacket
	}
	sdp, err := toBase64Json(localSDP)
	if err != nil {
		c.Log.Error().Err(err).Msgf("SDP encode fail fro [%v]", localSDP)
		return comm.EmptyPacket
	}

	// use user uid from the coordinator
	user := NewSession(peer, rq.Id)
	h.router.AddUser(user)
	c.Log.Info().Str("id", string(rq.Id)).Msgf("Peer connection (uid:%s)", user.GetId())

	return comm.Out{Payload: sdp}
}

func (c *Coordinator) HandleWebrtcAnswer(rq api.WebrtcAnswerRequest, h *Handler) {
	if user := h.router.GetUser(rq.Id); user != nil {
		if err := user.GetPeerConn().SetRemoteSDP(rq.Sdp, fromBase64Json); err != nil {
			c.Log.Error().Err(err).Msgf("cannot set remote SDP of client [%v]", rq.Id)
		}
	}
}

func (c *Coordinator) HandleWebrtcIceCandidate(rs api.WebrtcIceCandidateRequest, h *Handler) {
	if user := h.router.GetUser(rs.Id); user != nil {
		if err := user.GetPeerConn().AddCandidate(rs.Candidate, fromBase64Json); err != nil {
			c.Log.Error().Err(err).Msgf("cannot add ICE candidate of the client [%v]", rs.Id)
		}
	}
}

func (c *Coordinator) HandleGameStart(rq api.StartGameRequest, h *Handler) comm.Out {
	user := h.router.GetUser(rq.Id)
	if user == nil {
		c.Log.Error().Msgf("no user [%v]", rq.Id)
		return comm.EmptyPacket
	}
	h.log.Info().Str("game", rq.Game.Name).Msg("Starting the game")
	// trying to find existing room with that id
	room := h.router.GetRoom(rq.Rid)
	if room == nil {
		h.log.Info().Str("room", rq.Rid).Msg("Create room")

		// recording
		if h.conf.Recording.Enabled {
			h.log.Info().Msgf("RECORD: %v %v", rq.Record, rq.RecordUser)
		}

		room = h.CreateRoom(
			rq.Room.Rid,
			games.GameMetadata{Name: rq.Game.Name, Base: rq.Game.Base, Type: rq.Game.Type, Path: rq.Game.Path},
			rq.Record, rq.RecordUser,
			func(room *Room) {
				h.router.RemoveRoom(room)
				// send signal to coordinator that the room is closed, coordinator will remove that room
				h.cord.CloseRoom(room.ID)
				h.log.Debug().Msgf("Room close has been called %v", room.ID)
			},
		)
		h.router.AddRoom(room)
		user.SetPlayerIndex(rq.PlayerIndex)
		h.log.Info().Msgf("Updated player index to: %d", rq.PlayerIndex)
	}
	// Attach peerconnection to room. If PC is already in room, don't detach
	if !room.HasUser(user) {
		h.removeUser(user)
		room.AddUser(user)
		room.PollUserInput(user)
	} else {
		h.log.Info().Msg("The peer was not detached")
	}
	// Register room to coordinator if we are connecting to coordinator
	if room == nil {
		c.Log.Error().Msgf("couldn't create a room [%v]", rq.Id)
		return comm.EmptyPacket
	}
	h.cord.RegisterRoom(room.ID)
	user.SetRoom(room)
	h.router.AddRoom(room)
	return comm.Out{Payload: api.StartGameResponse{Room: api.Room{Rid: room.ID}, Record: h.conf.Recording.Enabled}}
}

func (c *Coordinator) HandleQuitGame(rq api.GameQuitRequest, h *Handler) {
	if user := h.router.GetUser(rq.Id); user != nil {
		if room := h.router.GetRoom(rq.Rid); room != nil {
			if room.HasUser(user) {
				h.removeUser(user)
			}
		}
	}
}

func (c *Coordinator) HandleSaveGame(rq api.SaveGameRequest, h *Handler) comm.Out {
	room := h.router.GetRoom(rq.Rid)
	if room == nil {
		return comm.ErrPacket
	}
	if err := room.SaveGame(); err != nil {
		c.Log.Error().Err(err).Msg("cannot save game state")
		return comm.ErrPacket
	}
	return comm.OkPacket
}

func (c *Coordinator) HandleLoadGame(rq api.LoadGameRequest, h *Handler) comm.Out {
	room := h.router.GetRoom(rq.Rid)
	if room == nil {
		return comm.ErrPacket
	}
	if err := room.LoadGame(); err != nil {
		c.Log.Error().Err(err).Msg("cannot load game state")
		return comm.ErrPacket
	}
	return comm.OkPacket
}

func (c *Coordinator) HandleChangePlayer(rq api.ChangePlayerRequest, h *Handler) comm.Out {
	user := h.router.GetUser(rq.Id)
	if user == nil || h.router.GetRoom(rq.Rid) == nil {
		return comm.Out{Payload: -1} // semi-predicates
	}
	user.SetPlayerIndex(rq.Index)
	h.log.Info().Msgf("Updated player index to: %d", rq.Index)
	return comm.Out{Payload: rq.Index}
}

func (c *Coordinator) HandleToggleMultitap(rq api.ToggleMultitapRequest, h *Handler) comm.Out {
	if rq.Rid == "" {
		return comm.ErrPacket
	}
	room := h.router.GetRoom(rq.Rid)
	if room == nil {
		return comm.ErrPacket
	}
	room.ToggleMultitap()
	return comm.OkPacket
}

func (c *Coordinator) HandleRecordGame(rq api.RecordGameRequest, h *Handler) comm.Out {
	if !h.conf.Recording.Enabled || rq.Rid == "" {
		return comm.ErrPacket
	}
	room := h.router.GetRoom(rq.Rid)
	if room == nil {
		return comm.ErrPacket
	}
	room.ToggleRecording(rq.Active, rq.User)
	return comm.OkPacket
}
