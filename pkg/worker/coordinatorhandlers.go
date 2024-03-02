package worker

import (
	"encoding/base64"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/com"
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/games"
	"github.com/giongto35/cloud-game/v3/pkg/network/webrtc"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged"
	"github.com/giongto35/cloud-game/v3/pkg/worker/media"
	"github.com/giongto35/cloud-game/v3/pkg/worker/room"
	"github.com/goccy/go-json"
)

// buildConnQuery builds initial connection data query to a coordinator.
func buildConnQuery(id com.Uid, conf config.Worker, address string) (string, error) {
	addr := conf.GetPingAddr(address)
	return toBase64Json(api.ConnectionRequest[com.Uid]{
		Addr:    addr.Hostname(),
		Id:      id,
		IsHTTPS: conf.Server.Https,
		PingURL: addr.String(),
		Port:    conf.GetPort(address),
		Tag:     conf.Tag,
		Zone:    conf.Network.Zone,
	})
}

func (c *coordinator) HandleWebrtcInit(rq api.WebrtcInitRequest[com.Uid], w *Worker, factory *webrtc.ApiFactory) api.Out {
	peer := webrtc.New(c.log, factory)
	localSDP, err := peer.NewCall(w.conf.Encoder.Video.Codec, "opus", func(data any) {
		candidate, err := toBase64Json(data)
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
	sdp, err := toBase64Json(localSDP)
	if err != nil {
		c.log.Error().Err(err).Msgf("SDP encode fail fro [%v]", localSDP)
		return api.EmptyPacket
	}

	user := room.NewGameSession(rq.Id, peer) // use user uid from the coordinator
	c.log.Info().Msgf("Peer connection: %s", user.Id())
	w.router.AddUser(user)

	return api.Out{Payload: sdp}
}

func (c *coordinator) HandleWebrtcAnswer(rq api.WebrtcAnswerRequest[com.Uid], w *Worker) {
	if user := w.router.FindUser(rq.Id); user != nil {
		if err := room.WithWebRTC(user.Session).SetRemoteSDP(rq.Sdp, fromBase64Json); err != nil {
			c.log.Error().Err(err).Msgf("cannot set remote SDP of client [%v]", rq.Id)
		}
	}
}

func (c *coordinator) HandleWebrtcIceCandidate(rs api.WebrtcIceCandidateRequest[com.Uid], w *Worker) {
	if user := w.router.FindUser(rs.Id); user != nil {
		if err := room.WithWebRTC(user.Session).AddCandidate(rs.Candidate, fromBase64Json); err != nil {
			c.log.Error().Err(err).Msgf("cannot add ICE candidate of the client [%v]", rs.Id)
		}
	}
}

func (c *coordinator) HandleGameStart(rq api.StartGameRequest[com.Uid], w *Worker) api.Out {
	user := w.router.FindUser(rq.Id)
	if user == nil {
		c.log.Error().Msgf("no user [%v]", rq.Id)
		return api.EmptyPacket
	}
	user.Index = rq.PlayerIndex

	r := w.router.FindRoom(rq.Room.Rid)

	if r == nil { // new room
		uid := rq.Room.Rid
		if uid == "" {
			uid = games.GenerateRoomID(rq.Game.Name)
		}
		game := games.GameMetadata(rq.Game)

		r = room.NewRoom[*room.GameSession](uid, nil, w.router.Users(), nil)
		r.HandleClose = func() {
			c.CloseRoom(uid)
			c.log.Debug().Msgf("room close request %v sent", uid)
		}

		if other := w.router.Room(); other != nil {
			c.log.Error().Msgf("concurrent room creation: %v / %v", uid, w.router.Room().Id())
			return api.EmptyPacket
		}

		w.router.SetRoom(r)
		c.log.Info().Str("room", r.Id()).Str("game", game.Name).Msg("New room")

		// start the emulator
		app := room.WithEmulator(w.mana.Get(caged.Libretro))
		app.ReloadFrontend()
		app.SetSessionId(uid)
		app.SetSaveOnClose(true)
		app.EnableCloudStorage(uid, w.storage)
		app.EnableRecording(rq.Record, rq.RecordUser, rq.Game.Name)

		r.SetApp(app)

		m := media.NewWebRtcMediaPipe(w.conf.Encoder.Audio, w.conf.Encoder.Video, w.log)

		// recreate the video encoder
		app.VideoChangeCb(func() {
			app.ViewportRecalculate()
			m.VideoW, m.VideoH = app.ViewportSize()
			m.VideoScale = app.Scale()

			if m.IsInitialized() {
				if err := m.Reinit(); err != nil {
					c.log.Error().Err(err).Msgf("reinit fail")
				}
			}

			data, err := api.Wrap(api.Out{T: uint8(api.AppVideoChange), Payload: api.AppVideoInfo{
				W: m.VideoW,
				H: m.VideoH,
				A: app.AspectRatio(),
				S: int(app.Scale()),
			}})
			if err != nil {
				c.log.Error().Err(err).Msgf("wrap")
			}
			r.Send(data)
		})

		w.log.Info().Msgf("Starting the game: %v", rq.Game.Name)
		if err := app.Load(game, w.conf.Worker.Library.BasePath); err != nil {
			c.log.Error().Err(err).Msgf("couldn't load the game %v", game)
			r.Close()
			w.router.SetRoom(nil)
			return api.EmptyPacket
		}

		m.AudioSrcHz = app.AudioSampleRate()
		m.AudioFrame = w.conf.Encoder.Audio.Frame
		m.VideoW, m.VideoH = app.ViewportSize()
		m.VideoScale = app.Scale()

		r.SetMedia(m)

		if err := m.Init(); err != nil {
			c.log.Error().Err(err).Msgf("couldn't init the media")
			r.Close()
			w.router.SetRoom(nil)
			return api.EmptyPacket
		}
		if app.Flipped() {
			m.SetVideoFlip(true)
		}
		m.SetPixFmt(app.PixFormat())
		m.SetRot(app.Rotation())

		r.BindAppMedia()
		r.StartApp()
	}

	c.log.Debug().Msg("Start session input poll")
	room.WithWebRTC(user.Session).OnMessage = func(data []byte) { r.App().SendControl(user.Index, data) }

	c.RegisterRoom(r.Id())

	response := api.StartGameResponse{Room: api.Room{Rid: r.Id()}, Record: w.conf.Recording.Enabled}
	if r.App().AspectEnabled() {
		ww, hh := r.App().ViewportSize()
		response.AV = &api.AppVideoInfo{W: ww, H: hh, A: r.App().AspectRatio(), S: int(r.App().Scale())}
	}

	return api.Out{Payload: response}
}

// HandleTerminateSession handles cases when a user has been disconnected from the websocket of coordinator.
func (c *coordinator) HandleTerminateSession(rq api.TerminateSessionRequest[com.Uid], w *Worker) {
	if user := w.router.FindUser(rq.Id); user != nil {
		w.router.Remove(user)
		c.log.Debug().Msgf(">>> users: %v", w.router.Users())
		user.Disconnect()
	}
}

// HandleQuitGame handles cases when a user manually exits the game.
func (c *coordinator) HandleQuitGame(rq api.GameQuitRequest[com.Uid], w *Worker) {
	if user := w.router.FindUser(rq.Id); user != nil {
		w.router.Remove(user)
		c.log.Debug().Msgf(">>> users: %v", w.router.Users())
	}
}

func (c *coordinator) HandleSaveGame(rq api.SaveGameRequest[com.Uid], w *Worker) api.Out {
	r := w.router.FindRoom(rq.Rid)
	if r == nil {
		return api.ErrPacket
	}
	if err := room.WithEmulator(r.App()).SaveGameState(); err != nil {
		c.log.Error().Err(err).Msg("cannot save game state")
		return api.ErrPacket
	}
	return api.OkPacket
}

func (c *coordinator) HandleLoadGame(rq api.LoadGameRequest[com.Uid], w *Worker) api.Out {
	r := w.router.FindRoom(rq.Rid)
	if r == nil {
		return api.ErrPacket
	}
	if err := room.WithEmulator(r.App()).RestoreGameState(); err != nil {
		c.log.Error().Err(err).Msg("cannot load game state")
		return api.ErrPacket
	}
	return api.OkPacket
}

func (c *coordinator) HandleChangePlayer(rq api.ChangePlayerRequest[com.Uid], w *Worker) api.Out {
	user := w.router.FindUser(rq.Id)
	if user == nil || w.router.FindRoom(rq.Rid) == nil {
		return api.Out{Payload: -1} // semi-predicates
	}
	user.Index = rq.Index
	w.log.Info().Msgf("Updated player index to: %d", rq.Index)
	return api.Out{Payload: rq.Index}
}

func (c *coordinator) HandleToggleMultitap(rq api.ToggleMultitapRequest[com.Uid], w *Worker) api.Out {
	r := w.router.FindRoom(rq.Rid)
	if r == nil {
		return api.ErrPacket
	}
	room.WithEmulator(r.App()).ToggleMultitap()
	return api.OkPacket
}

func (c *coordinator) HandleRecordGame(rq api.RecordGameRequest[com.Uid], w *Worker) api.Out {
	if !w.conf.Recording.Enabled {
		return api.ErrPacket
	}
	r := w.router.FindRoom(rq.Rid)
	if r == nil {
		return api.ErrPacket
	}
	room.WithRecorder(r.App()).ToggleRecording(rq.Active, rq.User)
	return api.OkPacket
}

// fromBase64Json decodes data from a URL-encoded Base64+JSON string.
func fromBase64Json(data string, obj any) error {
	b, err := base64.URLEncoding.DecodeString(data)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, obj)
	if err != nil {
		return err
	}
	return nil
}

// toBase64Json encodes data to a URL-encoded Base64+JSON string.
func toBase64Json(data any) (string, error) {
	if data == nil {
		return "", nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
