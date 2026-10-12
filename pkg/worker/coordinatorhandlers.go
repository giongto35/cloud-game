package worker

import (
	"encoding/json"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/com"
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/games"
	"github.com/giongto35/cloud-game/v3/pkg/network/webrtc"
	"github.com/giongto35/cloud-game/v3/pkg/worker/caged"
	"github.com/giongto35/cloud-game/v3/pkg/worker/media"
	"github.com/giongto35/cloud-game/v3/pkg/worker/room"
)

// maxPlayers is the number of emulator input ports.
const maxPlayers = 4

// clampPlayer maps invalid player indexes to -1 (spectator, no input).
func clampPlayer(i int) int {
	if i < 0 || i >= maxPlayers {
		return -1
	}
	return i
}

// buildConnQuery builds initial connection data query to a coordinator.
func buildConnQuery(id com.Uid, conf config.Worker, address string) (string, error) {
	addr := conf.GetPingAddr(address)
	return toJson(api.ConnectionRequest[com.Uid]{
		Addr:    addr.Hostname(),
		Id:      id,
		IsHTTPS: conf.Server.Https,
		PingURL: addr.String(),
		Port:    conf.GetPort(address),
		Tag:     conf.Tag,
		Zone:    conf.Network.Zone,
	})
}

func (c *coordinator) HandleInitWebrtcStream(rq api.InitWebrtcStreamRequest, w *Worker, factory *webrtc.ApiFactory) api.Out {
	var err error
	defer func() {
		if err != nil {
			c.log.Error().Err(err).Str("peer", rq.Id).Msg("")
		}
	}()

	peer := webrtc.New(c.log, factory)
	codec := w.conf.Encoder.Video.Codec

	if err = peer.NewConnection(codec, func(ice *string) { c.IceCandidate(*ice, rq.Id) }); err != nil {
		return api.EmptyPacket
	}

	if rq.Initiator {
		if err = peer.HandleSignal(nil, &rq.Sdp); err != nil {
			return api.EmptyPacket
		}
	}

	var sdp string
	sdp, err = peer.OfferAnswer(!rq.Initiator)
	if err != nil {
		return api.EmptyPacket
	}

	user := room.NewGameSession(rq.Id, peer) // use user uid from the coordinator
	c.log.Info().Msgf("Peer connection: %s", user.Id())
	w.router.AddUser(user)

	return api.Out{Payload: sdp}
}

func (c *coordinator) HandleWebrtcSignal(rq api.WebrtcSignalRequest, w *Worker) {
	user := w.router.FindUser(rq.Id)
	if user == nil {
		return
	}

	if webrtc := room.WithWebRTC(user.Session); webrtc != nil {
		if err := webrtc.HandleSignal(rq.Ice, rq.Sdp); err != nil {
			c.log.Error().Err(err).Msgf("cannot handle the signal from [%v]", rq.Id)
		}
	}
}

func (c *coordinator) HandleGameStart(rq api.StartGameRequest, w *Worker) api.Out {
	user := w.router.FindUser(rq.Id)
	if user == nil {
		c.log.Error().Msgf("no user [%v]", rq.Id)
		return api.EmptyPacket
	}
	user.Index = clampPlayer(rq.PlayerIndex)

	r := w.router.FindRoom(rq.Rid)

	// +injects game data into the original game request
	// the name of the game either in the `room id` field or
	// it's in the initial request
	gameName := rq.Game
	if rq.Rid != "" {
		name := w.launcher.ExtractAppNameFromUrl(rq.Rid)
		if name == "" {
			c.log.Warn().Msg("couldn't decode game name from the room id")
			return api.EmptyPacket
		}
		gameName = name
	}

	gameInfo, err := w.launcher.FindAppByName(gameName)
	if err != nil {
		c.log.Error().Err(err).Send()
		return api.EmptyPacket
	}

	if r == nil { // new room
		uid := rq.Rid
		if uid == "" {
			uid = games.GenerateRoomID(gameName)
		}
		game := games.GameMetadata(gameInfo)

		r = room.NewRoom(uid, nil, w.router.Users(), nil)
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
		app.EnableRecording(rq.Record, rq.RecordUser, gameName)

		r.SetApp(app)

		m := media.NewGstreamer(w.conf.Encoder, w.log)

		// recreate the video encoder
		app.VideoChangeCb(func() {
			app.ViewportRecalculate()
			m.VideoW, m.VideoH = app.ViewportSize()
			m.VideoScale, m.ScaleMethod = app.Scale()

			if err := m.Reinit(); err != nil {
				c.log.Error().Err(err).Msgf("reinit fail")
			}

			data, err := api.Wrap(api.Out{
				T: uint8(api.AppVideoChange),
				Payload: api.AppVideoInfo{
					W:    m.VideoW,
					H:    m.VideoH,
					A:    app.AspectRatio(),
					S:    int(m.VideoScale),
					SM:   m.ScaleMethod,
					Flip: app.Flipped(),
					Rot:  app.Rotation(),
				}})
			if err != nil {
				c.log.Error().Err(err).Msgf("wrap")
			}
			r.Send(data)
		})

		w.log.Info().Msgf("Starting the game: %v", gameName)
		if err := app.Load(game, w.conf.Library.BasePath); err != nil {
			c.log.Error().Err(err).Msgf("couldn't load the game %v", game)
			r.Close()
			w.router.SetRoom(nil)
			return api.EmptyPacket
		}

		m.AudioSrcHz = app.AudioSampleRate()
		m.VideoW, m.VideoH = app.ViewportSize()
		m.VideoScale, m.ScaleMethod = app.Scale()
		coreConf := w.conf.Emulator.GetLibretroCoreConfig(game.System)
		m.MaxThreads = coreConf.MaxThreads
		m.VideoFPS = app.FPS()
		m.VideoVFR = coreConf.VFR

		m.SetPixFmt(app.PixFormat())
		m.SetRot(app.Rotation())

		r.SetMedia(m)

		if err := m.Init(); err != nil {
			c.log.Error().Err(err).Msgf("couldn't init the media")
			r.Close()
			w.router.SetRoom(nil)
			return api.EmptyPacket
		}

		r.InitMedia()

		if rq.Tournament != nil || len(rq.Replay) > 0 {
			if err := c.setupMatchRoom(app.Frontend(), rq, uid, gameName, game.System); err != nil {
				c.log.Error().Err(err).Msg("couldn't prepare the tournament room")
				r.Close()
				w.router.SetRoom(nil)
				return api.EmptyPacket
			}
		}

		r.StartApp()
	}

	needsKbMouse := r.App().KbMouseSupport()

	// spectators (index -1) and out-of-range indexes never reach the emulator
	input := func(device byte) func([]byte) {
		return func(data []byte) {
			if idx := user.Index; idx >= 0 && idx < maxPlayers {
				r.App().Input(idx, device, data)
			}
		}
	}
	s := room.WithWebRTC(user.Session)
	s.OnMessage(input(byte(caged.RetroPad)))
	if needsKbMouse {
		_, _ = s.Channel("keyboard", nil, input(byte(caged.Keyboard)))
		_, _ = s.Channel("mouse", nil, input(byte(caged.Mouse)))
	}

	c.RegisterRoom(r.Id())

	response := api.StartGameResponse{
		Room:    api.Room{Rid: r.Id()},
		Record:  w.conf.Recording.Enabled,
		KbMouse: needsKbMouse,
	}

	if r.App().AspectEnabled() {
		ww, hh := r.App().ViewportSize()
		scale, scaleM := r.App().Scale()
		response.AV = &api.AppVideoInfo{
			W:    ww,
			H:    hh,
			A:    r.App().AspectRatio(),
			S:    int(scale),
			SM:   scaleM,
			Flip: r.App().Flipped(),
			Rot:  r.App().Rotation(),
		}
	}

	return api.Out{Payload: response}
}

// HandleTerminateSession handles cases when a user has been disconnected from the websocket of coordinator.
func (c *coordinator) HandleTerminateSession(rq api.TerminateSessionRequest, w *Worker) {
	if user := w.router.FindUser(rq.Id); user != nil {
		w.router.Remove(user)
		user.Disconnect()
	}
}

// HandleQuitGame handles cases when a user manually exits the game.
func (c *coordinator) HandleQuitGame(rq api.GameQuitRequest, w *Worker) {
	if user := w.router.FindUser(rq.Id); user != nil {
		w.router.Remove(user)
	}
}

func (c *coordinator) HandleResetGame(rq api.ResetGameRequest, w *Worker) api.Out {
	if r := w.router.FindRoom(rq.Rid); r != nil {
		room.WithEmulator(r.App()).Reset()
		return api.OkPacket
	}
	return api.ErrPacket
}

func (c *coordinator) HandleSaveGame(rq api.SaveGameRequest, w *Worker) api.Out {
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

func (c *coordinator) HandleLoadGame(rq api.LoadGameRequest, w *Worker) api.Out {
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

func (c *coordinator) HandleChangePlayer(rq api.ChangePlayerRequest, w *Worker) api.Out {
	user := w.router.FindUser(rq.Id)
	if user == nil || w.router.FindRoom(rq.Rid) == nil {
		return api.Out{Payload: -1} // semi-predicates
	}
	user.Index = clampPlayer(rq.Index)
	w.log.Info().Msgf("Updated player index to: %d", user.Index)
	return api.Out{Payload: user.Index}
}

func (c *coordinator) HandleRecordGame(rq api.RecordGameRequest, w *Worker) api.Out {
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

func toJson(data any) (string, error) {
	if data == nil {
		return "", nil
	}
	b, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
