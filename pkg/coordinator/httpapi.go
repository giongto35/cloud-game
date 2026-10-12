package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/auth"
	"github.com/giongto35/cloud-game/v3/pkg/covers"
	"github.com/giongto35/cloud-game/v3/pkg/games"
	"github.com/giongto35/cloud-game/v3/pkg/logger"
	"github.com/giongto35/cloud-game/v3/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v3/pkg/store"
	"github.com/giongto35/cloud-game/v3/pkg/tournament"
)

const sessionCookie = "belko_session"

// API serves the JSON HTTP endpoints for accounts and administration.
type API struct {
	auth   *auth.Service
	store  *store.Store
	tm     *tournament.Manager
	covers *covers.Service
	hub    *Hub
	secure bool
	log    *logger.Logger
}

func sessionToken(r *http.Request) string {
	if c, err := r.Cookie(sessionCookie); err == nil {
		return c.Value
	}
	return ""
}

// accountFromRequest resolves the account of the request session cookie.
func accountFromRequest(r *http.Request, svc *auth.Service) (*auth.Account, string) {
	token := sessionToken(r)
	if token == "" {
		return nil, ""
	}
	acc, err := svc.Authenticate(r.Context(), token)
	if err != nil {
		return nil, ""
	}
	return acc, token
}

func clientIp(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Routes registers all API routes on the mux.
func (a *API) Routes(mux *httpx.Mux) {
	mux.HandleFunc("POST /api/auth/register", a.mutating(a.register))
	mux.HandleFunc("POST /api/auth/login", a.mutating(a.login))
	mux.HandleFunc("POST /api/auth/logout", a.mutating(a.logout))
	mux.HandleFunc("GET /api/auth/me", a.me)

	mux.HandleFunc("PUT /api/account/profile", a.mutating(a.authed(a.updateProfile)))
	mux.HandleFunc("PUT /api/account/password", a.mutating(a.authed(a.changePassword)))
	mux.HandleFunc("GET /api/account/preferences", a.authed(a.getPreferences))
	mux.HandleFunc("PUT /api/account/preferences", a.mutating(a.authed(a.setPreferences)))

	mux.HandleFunc("GET /api/admin/users", a.admin(a.listUsers))
	mux.HandleFunc("POST /api/admin/users/{id}/{action}", a.mutating(a.admin(a.userAction)))
	mux.HandleFunc("PUT /api/admin/users/{id}/role", a.mutating(a.admin(a.userRole)))
	mux.HandleFunc("GET /api/admin/overview", a.admin(a.overview))
	mux.HandleFunc("POST /api/admin/rooms/close", a.mutating(a.admin(a.closeRoom)))
	mux.HandleFunc("GET /api/admin/audit", a.admin(a.audit))
	mux.HandleFunc("POST /api/admin/covers/retry", a.mutating(a.admin(a.coversRetry)))

	mux.HandleFunc("GET /api/covers", a.authed(a.cover))
	a.tournamentRoutes(mux)

	mux.HandleFunc("/api/", func(w http.ResponseWriter, r *http.Request) {
		writeErr(w, &auth.Error{Status: 404, Code: "not_found", Message: "Recurso não encontrado."})
	})
}

// helpers

func writeJson(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	var ae *auth.Error
	if !errors.As(err, &ae) {
		ae = &auth.Error{Status: 500, Code: "internal", Message: "Algo deu errado. Tente novamente em instantes."}
	}
	writeJson(w, ae.Status, map[string]any{"error": map[string]string{"code": ae.Code, "message": ae.Message}})
}

func readJson(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 128*1024))
	if err := dec.Decode(v); err != nil {
		return &auth.Error{Status: 400, Code: "bad_request", Message: "Requisição inválida."}
	}
	return nil
}

// mutating protects state-changing endpoints against CSRF:
// JSON body only and a same-origin Origin header (when present).
func (a *API) mutating(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
			writeErr(w, &auth.Error{Status: 415, Code: "unsupported_media", Message: "Requisição inválida."})
			return
		}
		if origin := r.Header.Get("Origin"); origin != "" {
			u, err := url.Parse(origin)
			if err != nil || !strings.EqualFold(u.Host, r.Host) {
				writeErr(w, &auth.Error{Status: 403, Code: "bad_origin", Message: "Origem não permitida."})
				return
			}
		}
		next(w, r)
	}
}

type authedHandler func(w http.ResponseWriter, r *http.Request, acc *auth.Account, token string)

func (a *API) authed(next authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		acc, token := accountFromRequest(r, a.auth)
		if acc == nil {
			writeErr(w, auth.ErrUnauthenticated)
			return
		}
		if acc.Status != store.StatusApproved {
			writeErr(w, auth.ErrForbidden)
			return
		}
		next(w, r, acc, token)
	}
}

func (a *API) admin(next authedHandler) http.HandlerFunc {
	return a.authed(func(w http.ResponseWriter, r *http.Request, acc *auth.Account, token string) {
		if !acc.Can(auth.PermAdminView) {
			writeErr(w, auth.ErrForbidden)
			return
		}
		next(w, r, acc, token)
	})
}

func (a *API) setSessionCookie(w http.ResponseWriter, r *http.Request, token string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		MaxAge:   int(maxAge.Seconds()),
		HttpOnly: true,
		Secure:   a.secure || r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})
}

type accountView struct {
	Id      int64    `json:"id"`
	Name    string   `json:"name"`
	Email   string   `json:"email"`
	Role    string   `json:"role"`
	Status  string   `json:"status"`
	Perms   []string `json:"perms"`
	Primary bool     `json:"primary,omitempty"`
}

func viewAccount(acc *auth.Account) accountView {
	perms := []string{}
	if acc.Status == store.StatusApproved {
		for _, p := range acc.Perms.List() {
			perms = append(perms, string(p))
		}
	}
	return accountView{Id: acc.Id, Name: acc.Name, Email: acc.Email, Role: string(acc.Role),
		Status: string(acc.Status), Perms: perms, Primary: acc.Primary}
}

// auth endpoints

func (a *API) register(w http.ResponseWriter, r *http.Request) {
	if !a.auth.RegisterLimiter.Allow(clientIp(r)) {
		writeErr(w, auth.ErrTooManyAttempts)
		return
	}
	var rq struct{ Name, Email, Password string }
	if err := readJson(r, &rq); err != nil {
		writeErr(w, err)
		return
	}
	if _, err := a.auth.Register(r.Context(), rq.Name, rq.Email, rq.Password); err != nil {
		writeErr(w, err)
		return
	}
	// sign in right away so the person sees the "waiting for approval" screen
	acc, token, err := a.auth.Login(r.Context(), rq.Email, rq.Password, clientIp(r), r.UserAgent())
	if err != nil {
		writeErr(w, err)
		return
	}
	a.setSessionCookie(w, r, token, a.auth.SessionTTL())
	writeJson(w, http.StatusCreated, map[string]any{"user": viewAccount(acc)})
}

func (a *API) login(w http.ResponseWriter, r *http.Request) {
	if !a.auth.LoginLimiter.Allow(clientIp(r)) {
		writeErr(w, auth.ErrTooManyAttempts)
		return
	}
	var rq struct{ Email, Password string }
	if err := readJson(r, &rq); err != nil {
		writeErr(w, err)
		return
	}
	acc, token, err := a.auth.Login(r.Context(), rq.Email, rq.Password, clientIp(r), r.UserAgent())
	if err != nil {
		writeErr(w, err)
		return
	}
	a.setSessionCookie(w, r, token, a.auth.SessionTTL())
	writeJson(w, http.StatusOK, map[string]any{"user": viewAccount(acc)})
}

func (a *API) logout(w http.ResponseWriter, r *http.Request) {
	_ = a.auth.Logout(r.Context(), sessionToken(r))
	a.setSessionCookie(w, r, "", -time.Second)
	writeJson(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *API) me(w http.ResponseWriter, r *http.Request) {
	acc, _ := accountFromRequest(r, a.auth)
	if acc == nil {
		writeErr(w, auth.ErrUnauthenticated)
		return
	}
	out := map[string]any{"user": viewAccount(acc), "moderators": a.auth.ModeratorsEnabled()}
	if acc.Can(auth.PermUsersApprove) {
		out["pending"] = a.auth.PendingCount(r.Context())
	}
	writeJson(w, http.StatusOK, out)
}

// account endpoints

func (a *API) updateProfile(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	var rq struct{ Name string }
	if err := readJson(r, &rq); err != nil {
		writeErr(w, err)
		return
	}
	name, err := a.auth.UpdateName(r.Context(), acc, rq.Name)
	if err != nil {
		writeErr(w, err)
		return
	}
	a.hub.onAccountChanged(acc.Id)
	writeJson(w, http.StatusOK, map[string]any{"name": name})
}

func (a *API) changePassword(w http.ResponseWriter, r *http.Request, acc *auth.Account, token string) {
	var rq struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := readJson(r, &rq); err != nil {
		writeErr(w, err)
		return
	}
	if err := a.auth.ChangePassword(r.Context(), acc, token, rq.Current, rq.New); err != nil {
		writeErr(w, err)
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *API) getPreferences(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	data, err := a.auth.Preferences(r.Context(), acc)
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(data))
}

func (a *API) setPreferences(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 64*1024+1))
	if err != nil || !json.Valid(body) {
		writeErr(w, &auth.Error{Status: 400, Code: "bad_request", Message: "Preferências inválidas."})
		return
	}
	if err := a.auth.SetPreferences(r.Context(), acc, string(body)); err != nil {
		writeErr(w, err)
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"ok": true})
}

// admin endpoints

type userView struct {
	Id          int64      `json:"id"`
	Name        string     `json:"name"`
	Email       string     `json:"email"`
	Role        string     `json:"role"`
	Status      string     `json:"status"`
	Primary     bool       `json:"primary,omitempty"`
	Online      bool       `json:"online,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ApprovedAt  *time.Time `json:"approved_at,omitempty"`
	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
}

func (a *API) viewUser(u store.User, online map[int64]bool) userView {
	return userView{Id: u.Id, Name: u.Name, Email: u.Email, Role: string(u.Role), Status: string(u.Status),
		Primary: a.auth.IsPrimary(&u), Online: online[u.Id], CreatedAt: u.CreatedAt, ApprovedAt: u.ApprovedAt,
		LastLoginAt: u.LastLoginAt}
}

func (a *API) listUsers(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	status := store.Status(r.URL.Query().Get("status"))
	users, err := a.auth.ListUsers(r.Context(), acc, status)
	if err != nil {
		writeErr(w, err)
		return
	}
	online := a.hub.onlineAccounts()
	out := make([]userView, 0, len(users))
	for _, u := range users {
		out = append(out, a.viewUser(u, online))
	}
	writeJson(w, http.StatusOK, map[string]any{"users": out})
}

func pathId(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, auth.ErrNotFound
	}
	return id, nil
}

func (a *API) userAction(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	id, err := pathId(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	var u *store.User
	switch r.PathValue("action") {
	case "approve", "unblock":
		u, err = a.auth.SetStatus(r.Context(), acc, id, store.StatusApproved)
	case "reject":
		u, err = a.auth.SetStatus(r.Context(), acc, id, store.StatusRejected)
	case "block":
		u, err = a.auth.SetStatus(r.Context(), acc, id, store.StatusBlocked)
	case "reset-password":
		password, err := a.auth.ResetPassword(r.Context(), acc, id)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJson(w, http.StatusOK, map[string]any{"password": password})
		return
	default:
		writeErr(w, &auth.Error{Status: 404, Code: "not_found", Message: "Ação desconhecida."})
		return
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"user": a.viewUser(*u, a.hub.onlineAccounts())})
}

func (a *API) userRole(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	id, err := pathId(r)
	if err != nil {
		writeErr(w, err)
		return
	}
	var rq struct{ Role string }
	if err := readJson(r, &rq); err != nil {
		writeErr(w, err)
		return
	}
	u, err := a.auth.SetRole(r.Context(), acc, id, store.Role(rq.Role))
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"user": a.viewUser(*u, a.hub.onlineAccounts())})
}

func (a *API) overview(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	if !acc.Can(auth.PermRoomsModerate) {
		writeErr(w, auth.ErrForbidden)
		return
	}
	out := a.hub.Overview(r.Context())
	if a.covers != nil {
		found, missing, _ := a.covers.Stats(r.Context())
		out["covers"] = map[string]int{"found": found, "missing": missing}
	}
	writeJson(w, http.StatusOK, out)
}

func (a *API) closeRoom(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	if !acc.Can(auth.PermRoomsModerate) {
		writeErr(w, auth.ErrForbidden)
		return
	}
	var rq struct {
		Room string `json:"room"`
	}
	if err := readJson(r, &rq); err != nil {
		writeErr(w, err)
		return
	}
	if !a.hub.CloseRoom(rq.Room) {
		writeErr(w, &auth.Error{Status: 404, Code: "not_found", Message: "Sala não encontrada."})
		return
	}
	a.auth.Audit(r.Context(), acc, "room.closed", games.ExtractGame(rq.Room))
	writeJson(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *API) audit(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	entries, err := a.auth.AuditLog(r.Context(), acc, 200)
	if err != nil {
		writeErr(w, err)
		return
	}
	type entry struct {
		Id        int64     `json:"id"`
		Actor     string    `json:"actor"`
		Action    string    `json:"action"`
		Target    string    `json:"target"`
		Details   string    `json:"details"`
		CreatedAt time.Time `json:"created_at"`
	}
	out := make([]entry, 0, len(entries))
	for _, e := range entries {
		out = append(out, entry{e.Id, e.ActorName, e.Action, e.Target, e.Details, e.CreatedAt})
	}
	writeJson(w, http.StatusOK, map[string]any{"entries": out})
}

// cover serves the stored cover image of a game, fetching it the first time.
func (a *API) cover(w http.ResponseWriter, r *http.Request, _ *auth.Account, _ string) {
	q := r.URL.Query()
	system, game := q.Get("system"), q.Get("game")
	if a.covers == nil || system == "" || game == "" || len(game) > 256 || len(system) > 32 {
		w.Header().Set("Cache-Control", "private, max-age=3600")
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
	defer cancel()
	path, err := a.covers.Path(ctx, system, game)
	if errors.Is(err, covers.ErrNotFound) {
		w.Header().Set("Cache-Control", "private, max-age=3600")
		http.NotFound(w, r)
		return
	}
	if err != nil {
		a.log.Warn().Err(err).Str("game", game).Msg("cover fetch failed")
		w.Header().Set("Cache-Control", "no-store")
		http.Error(w, "cover unavailable", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=2592000")
	http.ServeFile(w, r, path)
}

func (a *API) coversRetry(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	if !acc.Can(auth.PermUsersManage) {
		writeErr(w, auth.ErrForbidden)
		return
	}
	if a.covers == nil {
		writeErr(w, &auth.Error{Status: 400, Code: "covers_disabled", Message: "As capas estão desativadas na configuração."})
		return
	}
	if err := a.covers.Retry(r.Context()); err != nil {
		writeErr(w, err)
		return
	}
	a.auth.Audit(r.Context(), acc, "covers.retry", "")
	writeJson(w, http.StatusOK, map[string]any{"ok": true})
}

// Hub admin helpers

func (h *Hub) onlineAccounts() map[int64]bool {
	out := map[int64]bool{}
	for _, u := range h.userList() {
		out[u.Account().Id] = true
	}
	return out
}

type overviewWorker struct {
	Id       string `json:"id"`
	Addr     string `json:"addr"`
	Zone     string `json:"zone,omitempty"`
	Tag      string `json:"tag,omitempty"`
	Busy     bool   `json:"busy"`
	Room     string `json:"room,omitempty"`
	Reserved string `json:"reserved,omitempty"`
	Games    int    `json:"games"`
}

type overviewQueue struct {
	Name      string    `json:"name"`
	Game      string    `json:"game"`
	Status    string    `json:"status"`
	Since     time.Time `json:"since"`
	Connected bool      `json:"connected"`
}

type overviewOnline struct {
	Id    int64  `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Room  string `json:"room,omitempty"`
	Slot  int    `json:"slot"`
}

// Overview returns a snapshot of servers, rooms, queue and online people.
func (h *Hub) Overview(ctx context.Context) map[string]any {
	users := h.userList()
	workers := h.workerList()
	h.mu.Lock()
	lobby := h.lobbyStateLocked(users, workers)
	ws := make([]overviewWorker, 0, len(workers))
	for _, w := range workers {
		ow := overviewWorker{Id: w.Id().String(), Addr: w.Addr, Zone: w.Zone, Tag: w.Tag, Busy: !w.HasSlot(),
			Room: w.RoomId, Games: len(w.AppNames())}
		if w.Port != "" {
			ow.Addr += ":" + w.Port
		}
		if e := h.reservationLocked(w); e != nil {
			ow.Reserved = e.name
		}
		ws = append(ws, ow)
	}
	queue := make([]overviewQueue, 0, len(h.queue))
	for _, e := range h.queue {
		st := "waiting"
		if e.called != nil {
			st = "called"
		}
		queue = append(queue, overviewQueue{Name: e.name, Game: e.game, Status: st, Since: e.joined, Connected: e.goneAt.IsZero()})
	}
	online := make([]overviewOnline, 0, len(users))
	for _, u := range users {
		a := u.Account()
		online = append(online, overviewOnline{Id: a.Id, Name: a.Name, Email: a.Email, Room: games.ExtractGame(u.room), Slot: u.slot})
	}
	h.mu.Unlock()
	return map[string]any{
		"workers": ws, "rooms": lobby.Rooms, "queue": queue, "online": online,
		"pending": h.auth.PendingCount(ctx),
	}
}

// CloseRoom disconnects everyone from the worker that runs the room, which closes it.
func (h *Hub) CloseRoom(rid string) bool {
	if rid == "" {
		return false
	}
	var target *Worker
	for _, w := range h.workerList() {
		if w.RoomId == rid {
			target = w
		}
	}
	if target == nil {
		return false
	}
	for _, u := range h.userList() {
		if u.w == target {
			h.Kick(u, "A sala foi encerrada pela administração.")
		}
	}
	return true
}

// indexHandler gates the app page behind an approved session.
func (a *API) gate(r *http.Request) (redirect string) {
	acc, _ := accountFromRequest(r, a.auth)
	if acc == nil || !acc.Can(auth.PermPlay) {
		target := "/entrar"
		if r.URL.RawQuery != "" && acc == nil {
			// keep invite links working after signing in
			target += "?next=" + url.QueryEscape("/?"+r.URL.RawQuery)
		}
		return target
	}
	return ""
}
