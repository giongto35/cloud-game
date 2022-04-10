package coordinator

import (
	"encoding/json"
	"sort"
	"strconv"

	"github.com/giongto35/cloud-game/v2/pkg/api"
	"github.com/giongto35/cloud-game/v2/pkg/config/coordinator"
	"github.com/giongto35/cloud-game/v2/pkg/config/shared"
	"github.com/giongto35/cloud-game/v2/pkg/launcher"
	"github.com/rs/xid"
)

func (u *User) HandleWebrtcInit() {
	if u.Worker == nil {
		return
	}
	resp, err := u.Worker.WebrtcInit(u.Id())
	if err != nil || resp == "" {
		u.log.Error().Err(err).Msg("malformed WebRTC init response")
		return
	}
	u.SendWebrtcOffer(resp)
}

func (u *User) HandleWebrtcAnswer(data json.RawMessage) {
	var req api.WebrtcAnswerUserRequest
	if err := json.Unmarshal(data, &req); err != nil {
		u.log.Error().Err(err).Msg("malformed WebRTC answer request")
		return
	}
	u.Worker.WebrtcAnswer(u.Id(), req)
}

func (u *User) HandleWebrtcIceCandidate(data json.RawMessage) {
	var req api.WebrtcIceCandidateUserRequest
	if err := json.Unmarshal(data, &req); err != nil {
		u.log.Error().Err(err).Msg("malformed Ice candidate request")
		return
	}
	u.Worker.WebrtcIceCandidate(u.Id(), req)
}

func (u *User) HandleStartGame(data json.RawMessage, launcher launcher.Launcher, conf coordinator.Config) {
	var req api.GameStartUserRequest
	if err := json.Unmarshal(data, &req); err != nil {
		u.log.Error().Err(err).Msg("malformed game start request")
		return
	}
	// +injects game data into the original game request
	// the name of the game either in the `room id` field or
	// it's in the initial request
	game := req.GameName
	if req.RoomId != "" {
		name := launcher.ExtractAppNameFromUrl(req.RoomId)
		if name == "" {
			u.log.Warn().Msg("couldn't decode game name from the room id")
			return
		}
		game = name
	}

	gameInfo, err := launcher.FindAppByName(game)
	if err != nil {
		u.log.Error().Err(err).Str("game", game).Msg("couldn't find game info")
		return
	}

	workerResp, err := u.Worker.StartGame(u.Id(), gameInfo, req, conf.Recording.Enabled)
	if err != nil {
		u.log.Error().Err(err).Msg("malformed game start response")
		return
	}
	// Response from worker contains initialized roomID. Set roomID to the session
	u.SetRoom(workerResp.Id)
	u.log.Info().Str("id", workerResp.Id).Msg("Received room response from worker")

	if err = u.StartGame(); err != nil {
		u.log.Error().Err(err).Msg("couldn't send back start request")
		return
	}

	// send back recording status
	if conf.Recording.Enabled && req.Record {
		_ = u.SendAndForget(api.RecordGame, "ok")
	}
}

func (u *User) HandleQuitGame(data json.RawMessage) {
	var req api.GameQuitRequest
	if err := json.Unmarshal(data, &req); err != nil {
		u.log.Error().Err(err).Msg("malformed game quit request")
		return
	}
	u.Worker.QuitGame(u.Id(), req.Room.Id)
}

func (u *User) HandleSaveGame() {
	resp, err := u.Worker.SaveGame(u.Id(), u.RoomID)
	if err != nil {
		u.log.Error().Err(err).Msg("malformed game save request")
		return
	}
	u.Notify(api.SaveGame, resp)
}

func (u *User) HandleLoadGame() {
	resp, err := u.Worker.LoadGame(u.Id(), u.RoomID)
	if err != nil {
		u.log.Error().Err(err).Msg("malformed game load request")
		return
	}
	u.Notify(api.LoadGame, resp)
}

func (u *User) HandleChangePlayer(data json.RawMessage) {
	var req api.ChangePlayerUserRequest
	if err := json.Unmarshal(data, &req); err != nil {
		u.log.Error().Err(err).Msg("malformed player change request")
		return
	}
	resp, err := u.Worker.ChangePlayer(u.Id(), u.RoomID, req)
	if err != nil || resp == "error" {
		u.log.Error().Err(err).Msg("player switch failed for some reason")
	}
	idx, err := strconv.Atoi(resp)
	if err != nil {
		u.log.Error().Err(err).Msg("malformed player change response")
		return
	}
	u.Notify(api.ChangePlayer, idx)
}

func (u *User) HandleToggleMultitap() { u.Worker.ToggleMultitap(u.Id(), u.RoomID) }

func (u *User) HandleRecordGame(data json.RawMessage, conf shared.Recording) {
	if !conf.Enabled {
		u.log.Warn().Msg("Recording should be disabled!")
		return
	}
	req := api.RecordGameRequest{}
	if err := json.Unmarshal(data, &req); err != nil {
		u.log.Error().Err(err).Msg("malformed record game request")
		return
	}

	u.log.Debug().Msgf("??? room: %v, rec: %v user: %v", u.RoomID, req.Active, req.User)

	if u.RoomID == "" {
		u.log.Error().Msg("Recording in the empty room is not allowed!")
		return
	}
	if u.Worker == nil {
		return
	}

	resp, err := u.Worker.RecordGame(u.Id(), u.RoomID, req.Active, req.User)
	if err != nil {
		u.log.Error().Err(err).Msg("malformed game record request")
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
			if id, err := xid.FromString(s.Id); err == nil {
				mid := string(id.Machine())
				if _, ok := unique[mid]; !ok {
					unique[mid] = &api.Server{Addr: s.Addr, PingURL: s.PingURL, Id: s.Id, InGroup: true}
				}
				unique[mid].Replicas++
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
