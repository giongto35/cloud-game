package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/network/webrtc"
)

// buildConnQuery builds initial connection data query to a coordinator.
func buildConnQuery(id api.Uid, conf worker.Worker, address string) (string, error) {
	addr := conf.GetPingAddr(address)
	return com.ToBase64Json(api.ConnectionRequest{
		Addr:    addr.Hostname(),
		Id:      id,
		IsHTTPS: conf.Server.Https,
		PingURL: addr.String(),
		Port:    conf.GetPort(address),
		Tag:     conf.Tag,
		Zone:    conf.Network.Zone,
	})
}

func (c *coordinator) HandleWebrtcInit(rq api.WebrtcInitRequest, w *Worker, connApi *webrtc.ApiFactory) com.Out {
	peer := webrtc.New(c.Log, connApi)
	localSDP, err := peer.NewCall(w.conf.Encoder.Video.Codec, audioCodec, func(data any) {
		candidate, err := com.ToBase64Json(data)
		if err != nil {
			c.Log.Error().Err(err).Msgf("ICE candidate encode fail for [%v]", data)
			return
		}
		c.IceCandidate(candidate, rq.Id)
	})
	if err != nil {
		c.Log.Error().Err(err).Msg("cannot create new webrtc session")
		return com.EmptyPacket
	}
	sdp, err := com.ToBase64Json(localSDP)
	if err != nil {
		c.Log.Error().Err(err).Msgf("SDP encode fail fro [%v]", localSDP)
		return com.EmptyPacket
	}

	// use user uid from the coordinator
	user := NewSession(peer, rq.Id)
	w.router.AddUser(user)
	c.Log.Info().Str("id", rq.Id.String()).Msgf("Peer connection (uid:%s)", user.Id())

	return com.Out{Payload: sdp}
}

func (c *coordinator) HandleWebrtcAnswer(rq api.WebrtcAnswerRequest, w *Worker) {
	if user := w.router.GetUser(rq.Id); user != nil {
		if err := user.GetPeerConn().SetRemoteSDP(rq.Sdp, com.FromBase64Json); err != nil {
			c.Log.Error().Err(err).Msgf("cannot set remote SDP of client [%v]", rq.Id)
		}
	}
}

func (c *coordinator) HandleWebrtcIceCandidate(rs api.WebrtcIceCandidateRequest, w *Worker) {
	if user := w.router.GetUser(rs.Id); user != nil {
		if err := user.GetPeerConn().AddCandidate(rs.Candidate, com.FromBase64Json); err != nil {
			c.Log.Error().Err(err).Msgf("cannot add ICE candidate of the client [%v]", rs.Id)
		}
	}
}

func (c *coordinator) HandleGameStart(rq api.StartGameRequest, w *Worker) com.Out {
	user := w.router.GetUser(rq.Id)
	if user == nil {
		c.Log.Error().Msgf("no user [%v]", rq.Id)
		return com.EmptyPacket
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
		c.Log.Error().Msgf("couldn't create a room [%v]", rq.Id)
		return com.EmptyPacket
	}

	if !room.HasUser(user) {
		room.AddUser(user)
		room.PollUserInput(user)
	}
	user.SetRoom(room)

	c.RegisterRoom(room.GetId())

	return com.Out{Payload: api.StartGameResponse{Room: api.Room{Rid: room.GetId()}, Record: w.conf.Recording.Enabled}}
}

// HandleTerminateSession handles cases when a user has been disconnected from the websocket of coordinator.
func (c *coordinator) HandleTerminateSession(rq api.TerminateSessionRequest, w *Worker) {
	if session := w.router.GetUser(rq.Id); session != nil {
		w.router.RemoveDisconnect(session)
		if room := session.GetSetRoom(nil); room != nil {
			room.CleanupUser(session)
		}
	}
}

// HandleQuitGame handles cases when a user manually exits the game.
func (c *coordinator) HandleQuitGame(rq api.GameQuitRequest, w *Worker) {
	if user := w.router.GetUser(rq.Id); user != nil {
		// we don't strictly need a room id form the request,
		// since users hold their room reference
		// !to remove rid, maybe
		if room := w.router.GetRoom(rq.Rid); room != nil {
			room.CleanupUser(user)
		}
	}
}

func (c *coordinator) HandleSaveGame(rq api.SaveGameRequest, w *Worker) com.Out {
	if room := roomy(rq, w); room != nil {
		if err := room.SaveGame(); err != nil {
			c.Log.Error().Err(err).Msg("cannot save game state")
			return com.ErrPacket
		}
		return com.OkPacket
	}
	return com.ErrPacket
}

func (c *coordinator) HandleLoadGame(rq api.LoadGameRequest, w *Worker) com.Out {
	if room := roomy(rq, w); room != nil {
		if err := room.LoadGame(); err != nil {
			c.Log.Error().Err(err).Msg("cannot load game state")
			return com.ErrPacket
		}
		return com.OkPacket
	}
	return com.ErrPacket
}

func (c *coordinator) HandleChangePlayer(rq api.ChangePlayerRequest, w *Worker) com.Out {
	user := w.router.GetUser(rq.Id)
	if user == nil || w.router.GetRoom(rq.Rid) == nil {
		return com.Out{Payload: -1} // semi-predicates
	}
	user.SetPlayerIndex(rq.Index)
	w.log.Info().Msgf("Updated player index to: %d", rq.Index)
	return com.Out{Payload: rq.Index}
}

func (c *coordinator) HandleToggleMultitap(rq api.ToggleMultitapRequest, w *Worker) com.Out {
	if room := roomy(rq, w); room != nil {
		room.ToggleMultitap()
		return com.OkPacket
	}
	return com.ErrPacket
}

func (c *coordinator) HandleRecordGame(rq api.RecordGameRequest, w *Worker) com.Out {
	if !w.conf.Recording.Enabled {
		return com.ErrPacket
	}
	if room := roomy(rq, w); room != nil {
		room.(*RecordingRoom).ToggleRecording(rq.Active, rq.User)
		return com.OkPacket
	}
	return com.ErrPacket
}

func roomy(rq api.RoomInterface, w *Worker) GamingRoom {
	rid := rq.GetRoom()
	if rid == "" {
		return nil
	}
	room := w.router.GetRoom(rid)
	if room == nil {
		return nil
	}
	return room
}
