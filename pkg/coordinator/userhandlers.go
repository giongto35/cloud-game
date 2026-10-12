package coordinator

import (
	"sort"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/api"
	"github.com/giongto35/cloud-game/v3/pkg/auth"
	"github.com/giongto35/cloud-game/v3/pkg/config"
)

func (u *User) HandleInitWebrtcStream(rq api.InitUserWebrtcStreamRequest) {
	if u.w == nil {
		u.log.Warn().Msg("no worker assigned")
		return
	}
	resp, err := u.w.InitWebrtcStream(u.Id().String(), rq.Initiator, rq.Sdp)
	if err != nil || resp == nil || *resp == api.EMPTY {
		u.log.Error().Err(err).Msg("malformed WebRTC init response")
		return
	}
	u.SendWebrtcOffer(string(*resp))
}

func (u *User) HandleWebrtcSignal(rq api.WebrtcSignalUser) {
	if u.w == nil {
		return
	}
	u.w.WebrtcSignal(u.Id().String(), rq.Sdp, rq.Ice)
}

func (u *User) HandleStartGame(rq api.GameStartUserRequest, conf config.CoordinatorConfig, h *Hub) {
	if u.w == nil {
		u.Notify(api.ErrNoFreeSlots, "")
		return
	}
	if rq.MatchId != 0 || rq.ReplayId != 0 {
		u.startSpecial(rq, h)
		return
	}
	// Worker slot / room gating:
	// - A BUSY worker (no free slot) only accepts joins into its current room.
	// - A FREE worker reserves its single slot for the new room, respecting
	//   queue reservations made for other accounts.
	// - When starting a NEW game on a worker that still looks busy, give the
	//   previous room a short grace period to close.
	if rq.RoomId == "" && !u.w.HasSlot() {
		const waitTotal = 3 * time.Second
		const step = 100 * time.Millisecond
		for waited := time.Duration(0); waited < waitTotal && !u.w.HasSlot(); waited += step {
			time.Sleep(step)
		}
	}

	acc := u.Account()
	joining := !u.w.HasSlot()
	if joining {
		if u.w.RoomId == "" || rq.RoomId != u.w.RoomId {
			u.Notify(api.ErrNoFreeSlots, "")
			return
		}
		if h.isBanned(rq.RoomId, acc.Id) {
			u.Notify(api.Kicked, api.KickedResponse{Reason: "Você foi removido desta sala pelo anfitrião."})
			return
		}
		rq.PlayerIndex = h.pickSlot(rq.RoomId, rq.PlayerIndex, rq.Spectator)
	} else {
		if rq.Spectator {
			// there is nothing to watch on a free worker: the room is over
			u.notice("room_gone", "Essa partida já terminou.")
			return
		}
		if !h.claimWorker(u) {
			u.Notify(api.ErrNoFreeSlots, "")
			return
		}
		rq.PlayerIndex = 0
	}

	startGameResp, err := u.w.StartGame(u.Id().String(), rq)
	if err != nil || startGameResp == nil || startGameResp.Rid == "" {
		u.log.Error().Err(err).Msg("malformed game start response")
		if !joining {
			u.w.FreeSlots()
			h.dispatch()
		}
		u.notice("error", "Não foi possível iniciar o jogo. Tente novamente.")
		return
	}
	u.log.Info().Str("id", startGameResp.Rid).Int("slot", rq.PlayerIndex).Msg("Received room response from worker")
	h.joinRoom(u, startGameResp.Rid, rq.PlayerIndex, !joining)
	u.StartGame(startGameResp.Rid, startGameResp.AV, startGameResp.KbMouse, rq.PlayerIndex, h.isHost(u))

	// send back recording status
	if conf.Recording.Enabled && rq.Record {
		u.Notify(api.RecordGame, api.OK)
	}
}

func (u *User) HandleQuitGame(rq api.GameQuitRequest, h *Hub) {
	if u.w == nil {
		return
	}
	if rq.Rid == u.w.RoomId {
		u.w.QuitGame(u.Id().String())
	}
	h.leaveRoom(u, "saiu da sala")
}

func (u *User) HandleResetGame(rq api.ResetGameRequest, h *Hub) {
	if u.w == nil || rq.Rid != u.w.RoomId {
		return
	}
	if !h.canControl(u) {
		u.notice("error", "Somente o anfitrião pode reiniciar o jogo.")
		return
	}
	u.w.ResetGame(u.Id().String())
	h.systemChat(h.roomOf(u), u.Account().Name+" reiniciou o jogo")
}

func (u *User) HandleSaveGame(h *Hub) error {
	if u.w == nil {
		return nil
	}
	if !h.canControl(u) {
		u.notice("error", "Somente o anfitrião pode salvar o jogo.")
		return nil
	}
	resp, err := u.w.SaveGame(u.Id().String())
	if err != nil {
		return err
	}

	if *resp == api.OK {
		if id, _ := api.ExplodeDeepLink(u.w.RoomId); id != "" {
			u.w.AddSession(id)
		}
		h.systemChat(h.roomOf(u), u.Account().Name+" salvou o jogo")
	}

	u.Notify(api.SaveGame, resp)
	return nil
}

func (u *User) HandleLoadGame(h *Hub) error {
	if u.w == nil {
		return nil
	}
	if !h.canControl(u) {
		u.notice("error", "Somente o anfitrião pode carregar o jogo salvo.")
		return nil
	}
	resp, err := u.w.LoadGame(u.Id().String())
	if err != nil {
		return err
	}
	if *resp == api.OK {
		h.systemChat(h.roomOf(u), u.Account().Name+" carregou o jogo salvo")
	}
	u.Notify(api.LoadGame, resp)
	return nil
}

// moveToSlot changes the slot of a room member (validated) and informs the worker.
func (h *Hub) moveToSlot(target *User, slot int) bool {
	if target.w == nil {
		return false
	}
	prev, ok := h.setSlot(target, slot)
	if !ok {
		return false
	}
	if prev == slot {
		return true
	}
	resp, err := target.w.ChangePlayer(target.Id().String(), slot)
	if err != nil || resp == nil || int(*resp) != slot {
		target.log.Error().Err(err).Msgf("player select fail, req: %v", slot)
		h.restoreSlot(target, prev)
		return false
	}
	target.Notify(api.ChangePlayer, slot)
	name := target.Account().Name
	if slot == SpectatorSlot {
		h.systemChat(h.roomOf(target), name+" agora está assistindo")
	} else {
		h.systemChat(h.roomOf(target), name+" agora é o Jogador "+itoa(slot+1))
	}
	h.scheduleSync()
	return true
}

func (u *User) HandleChangePlayer(rq api.ChangePlayerUserRequest, h *Hub) {
	if !h.moveToSlot(u, int(rq)) {
		u.notice("error", "Essa vaga não está disponível.")
	}
}

// findMember finds another member of the same room by connection id.
func (h *Hub) findMember(u *User, id string) *User {
	t := h.users.Find(id)
	if t == nil {
		return nil
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if u.room == "" || t.room != u.room {
		return nil
	}
	return t
}

func (u *User) HandleRoomAssign(rq api.RoomAssignRequest, h *Hub) {
	if !h.canControl(u) {
		u.notice("error", "Somente o anfitrião pode mover jogadores.")
		return
	}
	t := h.findMember(u, rq.User)
	if t == nil || !h.moveToSlot(t, rq.Slot) {
		u.notice("error", "Não foi possível mover esse jogador.")
	}
}

func (u *User) HandleRoomKick(rq api.RoomKickRequest, h *Hub) {
	if !h.canControl(u) && !(h.isLocked(u) && u.Account().Can(auth.PermRoomsModerate)) {
		u.notice("error", "Somente o anfitrião pode remover pessoas da sala.")
		return
	}
	t := h.findMember(u, rq.User)
	if t == nil || t == u {
		return
	}
	if t.Account().Can(auth.PermRoomsModerate) && !u.Account().Can(auth.PermRoomsModerate) {
		u.notice("error", "Você não pode remover um moderador.")
		return
	}
	rid := h.roomOf(t)
	h.ban(rid, t.Account().Id)
	h.leaveRoom(t, "foi removido da sala")
	h.Kick(t, "Você foi removido da sala por "+u.Account().Name+".")
}

func (u *User) HandleRecordGame(rq api.RecordGameRequest) {
	if u.w == nil {
		return
	}

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
