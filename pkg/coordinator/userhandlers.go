package coordinator

import (
	"sort"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/games"
)

func (u *User) HandleWebrtcInit() {
	resp, err := u.Worker.WebrtcInit(u.Id())
	if err != nil || resp == nil || *resp == api.EMPTY {
		u.Log.Error().Err(err).Msg("malformed WebRTC init response")
		return
	}
	u.SendWebrtcOffer(*resp)
}

func (u *User) HandleWebrtcAnswer(rq api.WebrtcAnswerUserRequest) {
	u.Worker.WebrtcAnswer(u.Id(), string(rq))
}

func (u *User) HandleWebrtcIceCandidate(rq api.WebrtcUserIceCandidate) {
	u.Worker.WebrtcIceCandidate(u.Id(), string(rq))
}

func (u *User) HandleStartGame(rq api.GameStartUserRequest, launcher games.Launcher, conf coordinator.Config) {
	// +injects game data into the original game request
	// the name of the game either in the `room id` field or
	// it's in the initial request
	game := rq.GameName
	if rq.RoomId != "" {
		name := launcher.ExtractAppNameFromUrl(rq.RoomId)
		if name == "" {
			u.Log.Warn().Msg("couldn't decode game name from the room id")
			return
		}
		game = name
	}

	gameInfo, err := launcher.FindAppByName(game)
	if err != nil {
		u.Log.Error().Err(err).Str("game", game).Msg("couldn't find game info")
		return
	}

	startGameResp, err := u.Worker.StartGame(u.Id(), gameInfo, rq)
	if err != nil || startGameResp == nil {
		u.Log.Error().Err(err).Msg("malformed game start response")
		return
	}
	// Response from worker contains initialized roomID. Set roomID to the session
	u.SetRoom(startGameResp.Rid)
	u.Log.Info().Str("id", startGameResp.Rid).Msg("Received room response from worker")
	u.StartGame()

	// send back recording status
	if conf.Recording.Enabled && rq.Record {
		u.Notify(api.RecordGame, api.OK)
	}
}

func (u *User) HandleQuitGame(rq api.GameQuitRequest) { u.Worker.QuitGame(u.Id(), rq.Room.Rid) }

func (u *User) HandleSaveGame() error {
	resp, err := u.Worker.SaveGame(u.Id(), u.RoomID)
	if err != nil {
		return err
	}
	u.Notify(api.SaveGame, resp)
	return nil
}

func (u *User) HandleLoadGame() error {
	resp, err := u.Worker.LoadGame(u.Id(), u.RoomID)
	if err != nil {
		return err
	}
	u.Notify(api.LoadGame, resp)
	return nil
}

func (u *User) HandleChangePlayer(rq api.ChangePlayerUserRequest) {
	resp, err := u.Worker.ChangePlayer(u.Id(), u.RoomID, int(rq))
	// !to make it a little less convoluted
	if err != nil || resp == nil || *resp == -1 {
		u.Log.Error().Err(err).Msg("player switch failed for some reason")
		return
	}
	u.Notify(api.ChangePlayer, rq)
}

func (u *User) HandleToggleMultitap() { u.Worker.ToggleMultitap(u.Id(), u.RoomID) }

func (u *User) HandleRecordGame(rq api.RecordGameRequest) {
	if u.Worker == nil {
		return
	}

	u.Log.Debug().Msgf("??? room: %v, rec: %v user: %v", u.RoomID, rq.Active, rq.User)

	if u.RoomID == "" {
		u.Log.Error().Msg("Recording in the empty room is not allowed!")
		return
	}

	resp, err := u.Worker.RecordGame(u.Id(), u.RoomID, rq.Active, rq.User)
	if err != nil {
		u.Log.Error().Err(err).Msg("malformed game record request")
		return
	}
	u.Notify(api.RecordGame, resp)
}

func (u *User) handleGetWorkerList(debug bool, info ServerInfo) {
	response := api.GetWorkerListResponse{}
	servers := info.getServerList()

	if debug {
		response.Servers = servers
	} else {
		// not sure if []byte to string always reversible :/
		unique := map[string]*api.Server{}
		for _, s := range servers {
			mid := s.Id.Machine()
			if _, ok := unique[mid]; !ok {
				unique[mid] = &api.Server{Addr: s.Addr, PingURL: s.PingURL, Id: s.Id, InGroup: true}
			}
			unique[mid].Replicas++
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
