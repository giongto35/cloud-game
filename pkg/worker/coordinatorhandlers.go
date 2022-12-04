package worker

import (
	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/com"
	"github.com/giongto35/cloud-game/v2/pkg/config/worker"
	"github.com/giongto35/cloud-game/v2/pkg/games"
	"github.com/giongto35/cloud-game/v2/pkg/network/webrtc"
)

func MakeConnectionRequest(id string, conf worker.Worker, address string) (string, error) {
	addr := conf.GetPingAddr(address)
	return api.ToBase64Json(api.ConnectionRequest{
		Addr:    addr.Hostname(),
		Id:      id,
		IsHTTPS: conf.Server.Https,
		PingURL: addr.String(),
		Port:    conf.GetPort(address),
		Tag:     conf.Tag,
		Zone:    conf.Network.Zone,
	})
}

func (c *coordinator) HandleWebrtcInit(rq api.WebrtcInitRequest, s *Service, connApi *webrtc.ApiFactory) com.Out {
	enc := s.conf.Encoder
	peer := webrtc.NewWebRTC(s.conf.Webrtc, c.Log, connApi)
	localSDP, err := peer.NewCall(enc.Video.Codec, enc.Audio.Codec, func(data any) {
		candidate, err := api.ToBase64Json(data)
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
	sdp, err := api.ToBase64Json(localSDP)
	if err != nil {
		c.Log.Error().Err(err).Msgf("SDP encode fail fro [%v]", localSDP)
		return com.EmptyPacket
	}

	// use user uid from the coordinator
	user := NewSession(peer, rq.Id)
	s.router.AddUser(user)
	c.Log.Info().Str("id", string(rq.Id)).Msgf("Peer connection (uid:%s)", user.Id())

	return com.Out{Payload: sdp}
}

func (c *coordinator) HandleWebrtcAnswer(rq api.WebrtcAnswerRequest, s *Service) {
	if user := s.router.GetUser(rq.Id); user != nil {
		if err := user.GetPeerConn().SetRemoteSDP(rq.Sdp, api.FromBase64Json); err != nil {
			c.Log.Error().Err(err).Msgf("cannot set remote SDP of client [%v]", rq.Id)
		}
	}
}

func (c *coordinator) HandleWebrtcIceCandidate(rs api.WebrtcIceCandidateRequest, s *Service) {
	if user := s.router.GetUser(rs.Id); user != nil {
		if err := user.GetPeerConn().AddCandidate(rs.Candidate, api.FromBase64Json); err != nil {
			c.Log.Error().Err(err).Msgf("cannot add ICE candidate of the client [%v]", rs.Id)
		}
	}
}

func (c *coordinator) HandleGameStart(rq api.StartGameRequest, s *Service) com.Out {
	user := s.router.GetUser(rq.Id)
	if user == nil {
		c.Log.Error().Msgf("no user [%v]", rq.Id)
		return com.EmptyPacket
	}
	s.log.Info().Str("game", rq.Game.Name).Msg("Starting the game")

	room := s.router.GetRoom(rq.Rid)
	if room == nil {
		s.log.Info().Str("room", rq.Rid).Msg("Create room")
		room = NewRoom(
			rq.Room.Rid,
			games.GameMetadata{Name: rq.Game.Name, Base: rq.Game.Base, Type: rq.Game.Type, Path: rq.Game.Path},
			s.storage,
			func(room *Room) {
				s.router.RemoveRoom()
				c.CloseRoom(room.id)
				s.log.Debug().Msgf("Room close has been called %v", room.id)
			},
			rq.Record, rq.RecordUser,
			s.conf,
			s.log,
		)
		s.router.SetRoom(room)
		user.SetPlayerIndex(rq.PlayerIndex)
		s.log.Info().Msgf("Updated player index to: %d", rq.PlayerIndex)
		if s.conf.Recording.Enabled {
			s.log.Info().Msgf("RECORD: %v %v", rq.Record, rq.RecordUser)
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

	c.RegisterRoom(room.id)
	user.SetRoom(room)

	return com.Out{Payload: api.StartGameResponse{Room: api.Room{Rid: room.id}, Record: s.conf.Recording.Enabled}}
}

// HandleTerminateSession handles cases when a user has been disconnected from the websocket of coordinator.
func (c *coordinator) HandleTerminateSession(rq api.TerminateSessionRequest, s *Service) {
	if session := s.router.GetUser(rq.Id); session != nil {
		session.Close()
		s.router.RemoveUser(session)
		room := session.GetRoom()
		if room == nil || room.IsEmpty() {
			return
		}
		room.RemoveUser(session)
		s.log.Info().Msg("Closing peer connection")
		if room.IsEmpty() {
			s.log.Info().Msg("Closing an empty room")
			room.Close()
		}
	}
}

// HandleQuitGame handles cases when a user manually exits the game.
func (c *coordinator) HandleQuitGame(rq api.GameQuitRequest, s *Service) {
	if user := s.router.GetUser(rq.Id); user != nil {
		if room := s.router.GetRoom(rq.Rid); room != nil {
			if room.HasUser(user) && !room.IsEmpty() {
				room.RemoveUser(user)
				s.log.Info().Msg("Closing peer connection")
				if room.IsEmpty() {
					s.log.Info().Msg("Closing an empty room")
					room.Close()
				}
			}
		}
	}
}

func (c *coordinator) HandleSaveGame(rq api.SaveGameRequest, s *Service) com.Out {
	if room := roomy(rq, s); room != nil {
		if err := room.SaveGame(); err != nil {
			c.Log.Error().Err(err).Msg("cannot save game state")
			return com.ErrPacket
		}
		return com.OkPacket
	}
	return com.ErrPacket
}

func (c *coordinator) HandleLoadGame(rq api.LoadGameRequest, s *Service) com.Out {
	if room := roomy(rq, s); room != nil {
		if err := room.LoadGame(); err != nil {
			c.Log.Error().Err(err).Msg("cannot load game state")
			return com.ErrPacket
		}
		return com.OkPacket
	}
	return com.ErrPacket
}

func (c *coordinator) HandleChangePlayer(rq api.ChangePlayerRequest, s *Service) com.Out {
	user := s.router.GetUser(rq.Id)
	if user == nil || s.router.GetRoom(rq.Rid) == nil {
		return com.Out{Payload: -1} // semi-predicates
	}
	user.SetPlayerIndex(rq.Index)
	s.log.Info().Msgf("Updated player index to: %d", rq.Index)
	return com.Out{Payload: rq.Index}
}

func (c *coordinator) HandleToggleMultitap(rq api.ToggleMultitapRequest, s *Service) com.Out {
	if room := roomy(rq, s); room != nil {
		room.ToggleMultitap()
		return com.OkPacket
	}
	return com.ErrPacket
}

func (c *coordinator) HandleRecordGame(rq api.RecordGameRequest, s *Service) com.Out {
	if !s.conf.Recording.Enabled {
		return com.ErrPacket
	}
	if room := roomy(rq, s); room != nil {
		room.ToggleRecording(rq.Active, rq.User)
		return com.OkPacket
	}
	return com.ErrPacket
}

func roomy(rq api.RoomInterface, s *Service) *Room {
	rid := rq.GetRoom()
	if rid == "" {
		return nil
	}
	room := s.router.GetRoom(rid)
	if room == nil {
		return nil
	}
	return room
}
