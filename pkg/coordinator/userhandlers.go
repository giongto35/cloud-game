package coordinator

import (
	"sort"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/com"
	"github.com/giongto35/cloud-game/v3/pkg/config"
	"github.com/giongto35/cloud-game/v3/pkg/games"
)

func (u *User) HandleWebrtcInit() {
	resp, err := u.w.WebrtcInit(u.Id())
	if err != nil || resp == nil || *resp == api.EMPTY {
		u.log.Error().Err(err).Msg("malformed WebRTC init response")
		return
	}
	u.SendWebrtcOffer(string(*resp))
}

func (u *User) HandleWebrtcAnswer(rq api.WebrtcAnswerUserRequest) {
	u.w.WebrtcAnswer(u.Id(), string(rq))
}

func (u *User) HandleWebrtcIceCandidate(rq api.WebrtcUserIceCandidate) {
	u.w.WebrtcIceCandidate(u.Id(), string(rq))
}

func (u *User) HandleStartGame(rq api.GameStartUserRequest, launcher games.Launcher, conf config.CoordinatorConfig) {
	// +injects game data into the original game request
	// the name of the game either in the `room id` field or
	// it's in the initial request
	game := rq.GameName
	if rq.RoomId != "" {
		name := launcher.ExtractAppNameFromUrl(rq.RoomId)
		if name == "" {
			u.log.Warn().Msg("couldn't decode game name from the room id")
			return
		}
		game = name
	}

	gameInfo, err := launcher.FindAppByName(game)
	if err != nil {
		u.log.Error().Err(err).Send()
		return
	}

	startGameResp, err := u.w.StartGame(u.Id(), gameInfo, rq)
	if err != nil || startGameResp == nil {
		u.log.Error().Err(err).Msg("malformed game start response")
		return
	}
	if startGameResp.Rid == "" {
		u.log.Error().Msg("there is no room")
		return
	}
	u.log.Info().Str("id", startGameResp.Rid).Msg("Received room response from worker")
	u.StartGame(startGameResp.AV, startGameResp.KbMouse)

	// send back recording status
	if conf.Recording.Enabled && rq.Record {
		u.Notify(api.RecordGame, api.OK)
	}
}

func (u *User) HandleQuitGame(rq api.GameQuitRequest[com.Uid]) {
	if rq.Room.Rid == u.w.RoomId {
		u.w.QuitGame(u.Id())
	}
}

func (u *User) HandleSaveGame() error {
	resp, err := u.w.SaveGame(u.Id())
	if err != nil {
		return err
	}
	u.Notify(api.SaveGame, resp)
	return nil
}

func (u *User) HandleLoadGame() error {
	resp, err := u.w.LoadGame(u.Id())
	if err != nil {
		return err
	}
	u.Notify(api.LoadGame, resp)
	return nil
}

func (u *User) HandleChangePlayer(rq api.ChangePlayerUserRequest) {
	resp, err := u.w.ChangePlayer(u.Id(), int(rq))
	// !to make it a little less convoluted
	if err != nil || resp == nil || *resp == -1 {
		u.log.Error().Err(err).Msgf("player select fail, req: %v", rq)
		return
	}
	u.Notify(api.ChangePlayer, rq)
}

func (u *User) HandleRecordGame(rq api.RecordGameRequest[com.Uid]) {
	if u.w == nil {
		return
	}

	u.log.Debug().Msgf("??? room: %v, rec: %v user: %v", u.w.RoomId, rq.Active, rq.User)

	if u.w.RoomId == "" {
		u.log.Error().Msg("Recording in the empty room is not allowed!")
		return
	}

	resp, err := u.w.RecordGame(u.Id(), rq.Active, rq.User)
	if err != nil {
		u.log.Error().Err(err).Msg("malformed game record request")
		return
	}
	u.Notify(api.RecordGame, resp)
}

func (u *User) handleGetWorkerList(debug bool, info HasServerInfo) {
	response := api.GetWorkerListResponse{}
	servers := info.GetServerList()

	if debug {
		response.Servers = servers
	} else {
		unique := map[string]*api.Server{}
		for _, s := range servers {
			mid := s.Machine
			if _, ok := unique[mid]; !ok {
				unique[mid] = &api.Server{Addr: s.Addr, PingURL: s.PingURL, Id: s.Id, InGroup: true}
			}
			v := unique[mid]
			if v != nil {
				v.Replicas++
			}
		}
		for _, v := range unique {
			response.Servers = append(response.Servers, *v)
		}
	}
	if len(response.Servers) > 0 {
		sort.SliceStable(response.Servers, func(i, j int) bool {
			if response.Servers[i].Addr != response.Servers[j].Addr {
				return response.Servers[i].Addr < response.Servers[j].Addr
			}
			return response.Servers[i].Port < response.Servers[j].Port
		})
	}
	u.Notify(api.GetWorkerList, response)
}
