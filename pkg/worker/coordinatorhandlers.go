package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/network/webrtc"
)

// buildConnQuery builds initial connection data query to a coordinator.
func buildConnQuery(id com.Uid, conf worker.Worker, address string) (string, error) {
	addr := conf.GetPingAddr(address)
	return com.ToBase64Json(api.ConnectionRequest[com.Uid]{
		Addr:    addr.Hostname(),
		Id:      id,
		IsHTTPS: conf.Server.Https,
		PingURL: addr.String(),
		Port:    conf.GetPort(address),
		Tag:     conf.Tag,
		Zone:    conf.Network.Zone,
	})
}

func (c *coordinator) HandleWebrtcInit(rq api.WebrtcInitRequest[com.Uid], w *Worker, connApi *webrtc.ApiFactory) api.Out {
	peer := webrtc.New(c.log, connApi)
	localSDP, err := peer.NewCall(w.conf.Encoder.Video.Codec, audioCodec, func(data any) {
		candidate, err := com.ToBase64Json(data)
		if err != nil {
			c.log.Error().Err(err).Msgf("ICE candidate encode fail for [%v]", data)
			return
		}
		c.IceCandidate(candidate, rq.Id)
	})
	if err != nil {
		c.log.Error().Err(err).Msg("cannot create new webrtc session")
		return api.EmptyPacket
	}
	sdp, err := com.ToBase64Json(localSDP)
	if err != nil {
		c.log.Error().Err(err).Msgf("SDP encode fail fro [%v]", localSDP)
		return api.EmptyPacket
	}

	// use user uid from the coordinator
	user := NewSession(peer, rq.Id)
	w.router.AddUser(user)
	c.log.Info().Str("id", rq.Id.String()).Msgf("Peer connection (uid:%s)", user.Id())

	return api.Out{Payload: sdp}
}

func (c *coordinator) HandleWebrtcAnswer(rq api.WebrtcAnswerRequest[com.Uid], w *Worker) {
	if user := w.router.GetUser(rq.Id); user != nil {
		if err := user.GetPeerConn().SetRemoteSDP(rq.Sdp, com.FromBase64Json); err != nil {
			c.log.Error().Err(err).Msgf("cannot set remote SDP of client [%v]", rq.Id)
		}
	}
}

func (c *coordinator) HandleWebrtcIceCandidate(rs api.WebrtcIceCandidateRequest[com.Uid], w *Worker) {
	if user := w.router.GetUser(rs.Id); user != nil {
		if err := user.GetPeerConn().AddCandidate(rs.Candidate, com.FromBase64Json); err != nil {
			c.log.Error().Err(err).Msgf("cannot add ICE candidate of the client [%v]", rs.Id)
		}
	}
}

func (c *coordinator) HandleGameStart(rq api.StartGameRequest[com.Uid], w *Worker) api.Out {
	user := w.router.GetUser(rq.Id)
	if user == nil {
		c.log.Error().Msgf("no user [%v]", rq.Id)
		return api.EmptyPacket
	}
	w.log.Info().Msgf("Starting game: %v", rq.Game.Name)

	room := w.router.GetRoom(rq.Rid)
	if room == nil {
		room = NewRoom(
			rq.Room.Rid,
			games.GameMetadata{Name: rq.Game.Name, Base: rq.Game.Base, Type: rq.Game.Type, Path: rq.Game.Path},
			func(room *Room) {
				w.router.RemoveRoom()
				c.CloseRoom(room.id)
				w.log.Debug().Msgf("Room close has been called %v", room.id)
			},
			w.conf,
			w.log,
		)
		user.SetPlayerIndex(rq.PlayerIndex)

		if w.storage != nil {
			room = WithCloudStorage(room, w.storage)
		}
		if w.conf.Recording.Enabled {
			room = WithRecording(room.(*Room), rq.Record, rq.RecordUser, rq.Game.Name, w.conf)
		}
		w.router.SetRoom(room)

		room.StartEmulator()

		if w.conf.Emulator.AutosaveSec > 0 {
			// !to can crash if emulator starts earlier
			go room.EnableAutosave(w.conf.Emulator.AutosaveSec)
		}
	}

	if room == nil {
		c.log.Error().Msgf("couldn't create a room [%v]", rq.Id)
		return api.EmptyPacket
	}

	if !room.HasUser(user) {
		room.AddUser(user)
		room.PollUserInput(user)
	}
	user.SetRoom(room)

	c.RegisterRoom(room.GetId())

	return api.Out{Payload: api.StartGameResponse{Room: api.Room{Rid: room.GetId()}, Record: w.conf.Recording.Enabled}}
}

// HandleTerminateSession handles cases when a user has been disconnected from the websocket of coordinator.
func (c *coordinator) HandleTerminateSession(rq api.TerminateSessionRequest[com.Uid], w *Worker) {
	session := w.router.GetUser(rq.Id)
	if session != nil {
		w.router.RemoveDisconnect(session)
		if room := session.GetSetRoom(nil); room != nil {
			room.CleanupUser(session)
		}
	}
}

// HandleQuitGame handles cases when a user manually exits the game.
func (c *coordinator) HandleQuitGame(rq api.GameQuitRequest[com.Uid], w *Worker) {
	if user := w.router.GetUser(rq.Id); user != nil {
		// we don't strictly need a room id form the request,
		// since users hold their room reference
		// !to remove rid, maybe
		if room := w.router.GetRoom(rq.Rid); room != nil {
			room.CleanupUser(user)
		}
	}
}

func (c *coordinator) HandleSaveGame(rq api.SaveGameRequest[com.Uid], w *Worker) api.Out {
	room := w.router.GetRoom(rq.Rid)
	if room == nil {
		return api.ErrPacket
	}
	if err := room.SaveGame(); err != nil {
		c.log.Error().Err(err).Msg("cannot save game state")
		return api.ErrPacket
	}
	return api.OkPacket
}

func (c *coordinator) HandleLoadGame(rq api.LoadGameRequest[com.Uid], w *Worker) api.Out {
	room := w.router.GetRoom(rq.Rid)
	if room == nil {
		return api.ErrPacket
	}
	if err := room.LoadGame(); err != nil {
		c.log.Error().Err(err).Msg("cannot load game state")
		return api.ErrPacket
	}
	return api.OkPacket
}

func (c *coordinator) HandleChangePlayer(rq api.ChangePlayerRequest[com.Uid], w *Worker) api.Out {
	user := w.router.GetUser(rq.Id)
	if user == nil || w.router.GetRoom(rq.Rid) == nil {
		return api.Out{Payload: -1} // semi-predicates
	}
	user.SetPlayerIndex(rq.Index)
	w.log.Info().Msgf("Updated player index to: %d", rq.Index)
	return api.Out{Payload: rq.Index}
}

func (c *coordinator) HandleToggleMultitap(rq api.ToggleMultitapRequest[com.Uid], w *Worker) api.Out {
	room := w.router.GetRoom(rq.Rid)
	if room == nil {
		return api.ErrPacket
	}
	room.ToggleMultitap()
	return api.OkPacket
}

func (c *coordinator) HandleRecordGame(rq api.RecordGameRequest[com.Uid], w *Worker) api.Out {
	if !w.conf.Recording.Enabled {
		return api.ErrPacket
	}
	room := w.router.GetRoom(rq.Rid)
	if room == nil {
		return api.ErrPacket
	}
	room.(*RecordingRoom).ToggleRecording(rq.Active, rq.User)
	return api.OkPacket
}
