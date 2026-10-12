package coordinator

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/giongto35/cloud-game/v3/pkg/auth"
	"github.com/giongto35/cloud-game/v3/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v3/pkg/store"
	"github.com/giongto35/cloud-game/v3/pkg/tournament"
	"github.com/giongto35/cloud-game/v3/pkg/tourney"
)

func (a *API) tournamentRoutes(mux *httpx.Mux) {
	mux.HandleFunc("GET /api/tournaments", a.authed(a.listTournaments))
	mux.HandleFunc("GET /api/tournaments/games", a.authed(a.tournamentGames))
	mux.HandleFunc("POST /api/tournaments", a.mutating(a.authed(a.createTournament)))
	mux.HandleFunc("GET /api/tournaments/{id}", a.authed(a.getTournament))
	mux.HandleFunc("POST /api/tournaments/{id}/{action}", a.mutating(a.authed(a.tournamentAction)))
	mux.HandleFunc("POST /api/matches/{id}/winner", a.mutating(a.authed(a.matchWinner)))
	mux.HandleFunc("POST /api/replays/{id}/watch", a.mutating(a.authed(a.watchReplay)))
	mux.HandleFunc("GET /api/replays/{id}/file", a.authed(a.replayFile))
}

func tErr(err error) error {
	switch {
	case errors.Is(err, tournament.ErrNotFound):
		return &auth.Error{Status: 404, Code: "not_found", Message: "Torneio ou partida não encontrado."}
	case errors.Is(err, tournament.ErrState), errors.Is(err, tournament.ErrFewPlayers), errors.Is(err, tournament.ErrBadWinner),
		errors.Is(err, tournament.ErrUnknownGame), errors.Is(err, tournament.ErrAlreadyJoind), errors.Is(err, tournament.ErrNotRunning):
		msg := err.Error()
		return &auth.Error{Status: 400, Code: "invalid", Message: strings.ToUpper(msg[:1]) + msg[1:] + "."}
	}
	return err
}

func idParam(r *http.Request, name string) int64 {
	id, _ := strconv.ParseInt(r.PathValue(name), 10, 64)
	return id
}

func (a *API) requireTournaments(w http.ResponseWriter) bool {
	if a.tm == nil {
		writeErr(w, &auth.Error{Status: 404, Code: "disabled", Message: "Os torneios estão desativados."})
		return false
	}
	return true
}

type playerView struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

type matchView struct {
	Id       int64       `json:"id"`
	Round    int         `json:"round"`
	Position int         `json:"position"`
	Status   string      `json:"status"`
	P1       *playerView `json:"p1,omitempty"`
	P2       *playerView `json:"p2,omitempty"`
	Winner   *int64      `json:"winner,omitempty"`
	Source   string      `json:"source,omitempty"`
	Score    string      `json:"score,omitempty"`
	Replay   *int64      `json:"replay,omitempty"`
	Live     bool        `json:"live,omitempty"`
}

type roundView struct {
	Round   int         `json:"round"`
	Name    string      `json:"name"`
	Matches []matchView `json:"matches"`
}

type tournamentView struct {
	Id         int64       `json:"id"`
	Name       string      `json:"name"`
	Game       string      `json:"game"`
	System     string      `json:"system"`
	Rule       string      `json:"rule"`
	RuleName   string      `json:"rule_name"`
	Hint       string      `json:"hint"`
	Status     string      `json:"status"`
	Players    int         `json:"players"`
	Winner     *playerView `json:"winner,omitempty"`
	Joined     bool        `json:"joined"`
	CreatedAt  time.Time   `json:"created_at"`
	StartedAt  *time.Time  `json:"started_at,omitempty"`
	FinishedAt *time.Time  `json:"finished_at,omitempty"`
}

func (a *API) viewTournament(t *store.Tournament, players []store.TournamentPlayer, me int64) tournamentView {
	v := tournamentView{Id: t.Id, Name: t.Name, Game: t.Game, System: t.System, Rule: t.Rule, Status: t.Status,
		Players: len(players), CreatedAt: t.CreatedAt, StartedAt: t.StartedAt, FinishedAt: t.FinishedAt}
	if r := tourney.ByKey(t.Rule); r != nil {
		v.RuleName, v.Hint = r.Name, r.Hint
	}
	for _, p := range players {
		if p.UserId == me {
			v.Joined = true
		}
		if t.WinnerId != nil && p.UserId == *t.WinnerId {
			v.Winner = &playerView{p.UserId, p.Name}
		}
	}
	return v
}

func (a *API) listTournaments(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	if !a.requireTournaments(w) {
		return
	}
	list, err := a.store.Tournaments(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	out := make([]tournamentView, 0, len(list))
	for i := range list {
		players, _ := a.store.TournamentPlayers(r.Context(), list[i].Id)
		out = append(out, a.viewTournament(&list[i], players, acc.Id))
	}
	running, queued := a.tm.Snapshot()
	writeJson(w, http.StatusOK, map[string]any{"tournaments": out, "can_manage": acc.Can(auth.PermRoomsModerate),
		"scheduler": map[string]any{"running": len(running), "queued": len(queued), "max": a.tm.MaxConcurrent()}})
}

// tournamentGames lists the library games with automatic result detection.
func (a *API) tournamentGames(w http.ResponseWriter, _ *http.Request, _ *auth.Account, _ string) {
	type game struct {
		Title  string `json:"title"`
		System string `json:"system"`
		Rule   string `json:"rule"`
		Name   string `json:"name"`
		Hint   string `json:"hint"`
	}
	out := []game{}
	for _, g := range a.hub.allGames() {
		if rule := tourney.Find(g.Title); rule != nil {
			out = append(out, game{g.Title, g.System, rule.Key, rule.Name, rule.Hint})
		}
	}
	slices.SortFunc(out, func(x, y game) int { return strings.Compare(x.Title, y.Title) })
	writeJson(w, http.StatusOK, map[string]any{"games": out})
}

func (a *API) createTournament(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	if !a.requireTournaments(w) {
		return
	}
	if !acc.Can(auth.PermRoomsModerate) {
		writeErr(w, auth.ErrForbidden)
		return
	}
	var rq struct{ Name, Game string }
	if err := readJson(r, &rq); err != nil {
		writeErr(w, err)
		return
	}
	rq.Name = strings.Join(strings.Fields(rq.Name), " ")
	if n := utf8.RuneCountInString(rq.Name); n < 3 || n > 60 {
		writeErr(w, &auth.Error{Status: 400, Code: "invalid_name", Message: "Dê ao torneio um nome entre 3 e 60 caracteres."})
		return
	}
	system := ""
	for _, g := range a.hub.allGames() {
		if g.Title == rq.Game {
			system = g.System
		}
	}
	if system == "" {
		writeErr(w, &auth.Error{Status: 400, Code: "unknown_game", Message: "Este jogo não está na biblioteca dos servidores."})
		return
	}
	t, err := a.tm.Create(r.Context(), rq.Name, rq.Game, system, acc.Id)
	if err != nil {
		writeErr(w, tErr(err))
		return
	}
	a.auth.Audit(r.Context(), acc, "tournament.created", fmt.Sprintf("%s (%s)", t.Name, t.Game))
	a.hub.Notify(nil, "tournaments", "", 0)
	writeJson(w, http.StatusCreated, map[string]any{"id": t.Id})
}

func (a *API) getTournament(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	if !a.requireTournaments(w) {
		return
	}
	ctx := r.Context()
	t, err := a.store.Tournament(ctx, idParam(r, "id"))
	if err != nil {
		writeErr(w, tErr(tournament.ErrNotFound))
		return
	}
	players, _ := a.store.TournamentPlayers(ctx, t.Id)
	names := map[int64]string{}
	pv := make([]playerView, 0, len(players))
	for _, p := range players {
		names[p.UserId] = p.Name
		pv = append(pv, playerView{p.UserId, p.Name})
	}
	pl := func(id *int64) *playerView {
		if id == nil {
			return nil
		}
		return &playerView{*id, names[*id]}
	}
	matches, _ := a.store.Matches(ctx, t.Id)
	rounds := 0
	for _, m := range matches {
		rounds = max(rounds, m.Round)
	}
	rv := make([]roundView, rounds)
	for i := range rv {
		rv[i] = roundView{Round: i + 1, Name: tournament.RoundName(i+1, rounds), Matches: []matchView{}}
	}
	var mine *matchView
	for _, m := range matches {
		live := m.Status == tournament.StatusRunning && a.hub.specialWorker(specialMatch, m.Id) != nil
		mv := matchView{Id: m.Id, Round: m.Round, Position: m.Position, Status: m.Status, P1: pl(m.P1), P2: pl(m.P2),
			Winner: m.WinnerId, Source: m.Source, Score: m.Score, Replay: m.ReplayId, Live: live}
		rv[m.Round-1].Matches = append(rv[m.Round-1].Matches, mv)
		if (m.Status == tournament.StatusStarting || m.Status == tournament.StatusRunning) &&
			((m.P1 != nil && *m.P1 == acc.Id) || (m.P2 != nil && *m.P2 == acc.Id)) {
			mine = &mv
		}
	}
	running, queued := a.tm.Snapshot()
	writeJson(w, http.StatusOK, map[string]any{
		"tournament": a.viewTournament(t, players, acc.Id), "players": pv, "rounds": rv, "my_match": mine,
		"can_manage": acc.Can(auth.PermRoomsModerate),
		"scheduler":  map[string]any{"running": running, "queued": queued, "max": a.tm.MaxConcurrent()},
	})
}

func (a *API) tournamentAction(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	if !a.requireTournaments(w) {
		return
	}
	ctx, id, action := r.Context(), idParam(r, "id"), r.PathValue("action")
	var err error
	switch action {
	case "join":
		err = a.tm.Join(ctx, id, acc.Id)
	case "leave":
		err = a.tm.Leave(ctx, id, acc.Id)
	case "start", "cancel":
		if !acc.Can(auth.PermRoomsModerate) {
			writeErr(w, auth.ErrForbidden)
			return
		}
		if action == "start" {
			err = a.tm.Begin(ctx, id)
		} else {
			err = a.tm.Cancel(ctx, id)
		}
		if err == nil {
			a.auth.Audit(ctx, acc, "tournament."+action, strconv.FormatInt(id, 10))
		}
	default:
		writeErr(w, &auth.Error{Status: 404, Code: "not_found", Message: "Ação desconhecida."})
		return
	}
	if err != nil {
		writeErr(w, tErr(err))
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *API) matchWinner(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	if !a.requireTournaments(w) {
		return
	}
	if !acc.Can(auth.PermRoomsModerate) {
		writeErr(w, auth.ErrForbidden)
		return
	}
	var rq struct {
		Winner int64 `json:"winner"`
	}
	if err := readJson(r, &rq); err != nil {
		writeErr(w, err)
		return
	}
	id := idParam(r, "id")
	if err := a.tm.SetWinner(id, rq.Winner); err != nil {
		writeErr(w, tErr(err))
		return
	}
	a.auth.Audit(r.Context(), acc, "match.winner_set", fmt.Sprintf("partida %d", id))
	writeJson(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *API) watchReplay(w http.ResponseWriter, r *http.Request, _ *auth.Account, _ string) {
	if !a.requireTournaments(w) {
		return
	}
	info, err := a.store.Replay(r.Context(), idParam(r, "id"))
	if err != nil {
		writeErr(w, tErr(tournament.ErrNotFound))
		return
	}
	if !a.hub.ReserveReplay(info.Id, info.Game) {
		writeErr(w, &auth.Error{Status: 409, Code: "busy", Message: "Nenhum servidor livre para o replay agora. Tente em instantes."})
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"id": info.Id, "game": info.Game})
}

func (a *API) replayFile(w http.ResponseWriter, r *http.Request, _ *auth.Account, _ string) {
	if !a.requireTournaments(w) {
		return
	}
	info, data, err := a.tm.ReplayData(r.Context(), idParam(r, "id"))
	if err != nil {
		writeErr(w, tErr(tournament.ErrNotFound))
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filepath.Base(info.File)))
	_, _ = w.Write(data)
}
