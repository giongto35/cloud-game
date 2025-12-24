package coordinator

import (
	"sort"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/config"
)

func (u *User) HandleWebrtcInit() {
	uid := u.Id().String()
	resp, err := u.w.WebrtcInit(uid)
	if err != nil || resp == nil || *resp == api.EMPTY {
		u.log.Error().Err(err).Msg("malformed WebRTC init response")
		return
	}
	u.SendWebrtcOffer(string(*resp))
}

func (u *User) HandleWebrtcAnswer(rq api.WebrtcAnswerUserRequest) {
	u.w.WebrtcAnswer(u.Id().String(), string(rq))
}

func (u *User) HandleWebrtcIceCandidate(rq api.WebrtcUserIceCandidate) {
	u.w.WebrtcIceCandidate(u.Id().String(), string(rq))
}

func (u *User) HandleStartGame(rq api.GameStartUserRequest, conf config.CoordinatorConfig) {
	// Worker slot / room gating:
	// - If the worker is BUSY (no free slot), we must not create another room.
	//   * If the worker has already reported a room id, only allow requests
	//     for that same room (deep-link joins / reloads).
	//   * If the worker hasn't reported a room yet, deny any new StartGame to
	//     avoid racing concurrent room creation on the worker.
	//   * When the user is starting a NEW game (empty room id), we give the
	//     worker a short grace period to close the previous room and free the
	//     slot before rejecting with "no slots".
	// - If the worker is FREE, reserve the slot lazily before starting the
	//   game; the room id (if any) comes from the request / worker.

	// Grace period: when there's no room id in the request (new game) but the
	// worker still appears busy, wait a bit for the previous room to close.
	if rq.RoomId == "" && !u.w.HasSlot() {
		const waitTotal = 3 * time.Second
		const step = 100 * time.Millisecond
		waited := time.Duration(0)
		for waited < waitTotal {
			if u.w.HasSlot() {
				break
			}
			time.Sleep(step)
			waited += step
		}
	}

	busy := !u.w.HasSlot()
	if busy {
		if u.w.RoomId == "" {
			u.Notify(api.ErrNoFreeSlots, "")
			return
		}
		if rq.RoomId == "" {
			// No room id but worker is busy -> assume user wants to continue
			// the existing room instead of starting a parallel game.
			rq.RoomId = u.w.RoomId
		} else if rq.RoomId != u.w.RoomId {
			u.Notify(api.ErrNoFreeSlots, "")
			return
		}
	} else {
		// Worker is free: try to reserve the single slot for this new room.
		if !u.w.TryReserve() {
			u.Notify(api.ErrNoFreeSlots, "")
			return
		}
	}

	startGameResp, err := u.w.StartGame(u.Id().String(), rq)
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

func (u *User) HandleQuitGame(rq api.GameQuitRequest) {
	if rq.Rid == u.w.RoomId {
		u.w.QuitGame(u.Id().String())
	}
}

func (u *User) HandleResetGame(rq api.ResetGameRequest) {
	if rq.Rid != u.w.RoomId {
		return
	}
	u.w.ResetGame(u.Id().String())
}

func (u *User) HandleSaveGame() error {
	resp, err := u.w.SaveGame(u.Id().String())
	if err != nil {
		return err
	}

	if *resp == api.OK {
		if id, _ := api.ExplodeDeepLink(u.w.RoomId); id != "" {
			u.w.AddSession(id)
		}
	}

	u.Notify(api.SaveGame, resp)
	return nil
}

func (u *User) HandleLoadGame() error {
	resp, err := u.w.LoadGame(u.Id().String())
	if err != nil {
		return err
	}
	u.Notify(api.LoadGame, resp)
	return nil
}

func (u *User) HandleChangePlayer(rq api.ChangePlayerUserRequest) {
	resp, err := u.w.ChangePlayer(u.Id().String(), int(rq))
	// !to make it a little less convoluted
	if err != nil || resp == nil || *resp == -1 {
		u.log.Error().Err(err).Msgf("player select fail, req: %v", rq)
		return
	}
	u.Notify(api.ChangePlayer, rq)
}

func (u *User) HandleRecordGame(rq api.RecordGameRequest) {
	if u.w == nil {
		return
	}

	u.log.Debug().Msgf("??? room: %v, rec: %v user: %v", u.w.RoomId, rq.Active, rq.User)

	if u.w.RoomId == "" {
		u.log.Error().Msg("Recording in the empty room is not allowed!")
		return
	}

	resp, err := u.w.RecordGame(u.Id().String(), rq.Active, rq.User)
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
