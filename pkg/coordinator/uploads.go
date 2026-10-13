package coordinator

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/giongto35/cloud-game/v3/pkg/auth"
	"github.com/giongto35/cloud-game/v3/pkg/imgutil"
	"github.com/giongto35/cloud-game/v3/pkg/network/httpx"
	"github.com/giongto35/cloud-game/v3/pkg/store"
)

const (
	maxAvatarUpload = 4 << 20
	maxCoverUpload  = 8 << 20
	avatarSide      = 256
	coverSide       = 640
)

func (a *API) uploadRoutes(mux *httpx.Mux) {
	mux.HandleFunc("GET /api/avatars/{id}", a.authed(a.getAvatar))
	mux.HandleFunc("PUT /api/account/avatar", a.upload(a.authed(a.putAvatar)))
	mux.HandleFunc("DELETE /api/account/avatar", a.mutating(a.authed(a.deleteAvatar)))

	mux.HandleFunc("PUT /api/admin/covers", a.upload(a.admin(a.putCover)))
	mux.HandleFunc("DELETE /api/admin/covers", a.mutating(a.admin(a.deleteCover)))
}

// upload protects endpoints that receive a picture as the raw request body:
// image content type and same-origin requests only (the CSRF rules of mutating).
func (a *API) upload(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "image/") {
			writeErr(w, &auth.Error{Status: 415, Code: "unsupported_media", Message: "Envie um arquivo de imagem (PNG, JPEG ou GIF)."})
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

func readImage(w http.ResponseWriter, r *http.Request, limit int64, o imgutil.Options) ([]byte, bool) {
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, limit))
	if err != nil {
		writeErr(w, &auth.Error{Status: 413, Code: "too_large", Message: "A imagem é grande demais (máximo " + strconv.FormatInt(limit>>20, 10) + " MB)."})
		return nil, false
	}
	png, err := imgutil.Process(data, o)
	switch {
	case errors.Is(err, imgutil.ErrTooLarge):
		writeErr(w, &auth.Error{Status: 413, Code: "too_large", Message: "As dimensões da imagem são grandes demais."})
		return nil, false
	case err != nil:
		writeErr(w, &auth.Error{Status: 400, Code: "bad_image", Message: "Não foi possível ler a imagem. Use PNG, JPEG ou GIF."})
		return nil, false
	}
	return png, true
}

func (a *API) getAvatar(w http.ResponseWriter, r *http.Request, _ *auth.Account, _ string) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	data, updated, err := a.store.Avatar(r.Context(), id)
	if errors.Is(err, store.ErrNotFound) {
		w.Header().Set("Cache-Control", "private, max-age=300")
		http.NotFound(w, r)
		return
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	http.ServeContent(w, r, "", updated, bytes.NewReader(data))
}

func (a *API) putAvatar(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	png, ok := readImage(w, r, maxAvatarUpload, imgutil.Options{Square: true, MaxSide: avatarSide})
	if !ok {
		return
	}
	if err := a.store.SetAvatar(r.Context(), acc.Id, png); err != nil {
		writeErr(w, err)
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"ok": true, "version": time.Now().Unix()})
}

func (a *API) deleteAvatar(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	if err := a.store.DeleteAvatar(r.Context(), acc.Id); err != nil {
		writeErr(w, err)
		return
	}
	writeJson(w, http.StatusOK, map[string]any{"ok": true})
}

func (a *API) coverTarget(w http.ResponseWriter, r *http.Request, acc *auth.Account) (system, game string, ok bool) {
	if !acc.Can(auth.PermUsersManage) {
		writeErr(w, auth.ErrForbidden)
		return "", "", false
	}
	if a.covers == nil {
		writeErr(w, &auth.Error{Status: 400, Code: "covers_disabled", Message: "As capas estão desativadas na configuração."})
		return "", "", false
	}
	q := r.URL.Query()
	system, game = q.Get("system"), q.Get("game")
	if system == "" || game == "" || len(game) > 256 || len(system) > 32 {
		writeErr(w, &auth.Error{Status: 400, Code: "bad_request", Message: "Informe o sistema e o jogo."})
		return "", "", false
	}
	return system, game, true
}

// putCover replaces the cover of a game with an uploaded picture, or adds one when it doesn't exist.
func (a *API) putCover(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	system, game, ok := a.coverTarget(w, r, acc)
	if !ok {
		return
	}
	png, ok := readImage(w, r, maxCoverUpload, imgutil.Options{MaxSide: coverSide})
	if !ok {
		return
	}
	if err := a.covers.SetCustom(r.Context(), system, game, png); err != nil {
		writeErr(w, err)
		return
	}
	a.auth.Audit(r.Context(), acc, "cover.set", game)
	writeJson(w, http.StatusOK, map[string]any{"ok": true, "version": time.Now().Unix()})
}

// deleteCover goes back to the automatic cover of a game.
func (a *API) deleteCover(w http.ResponseWriter, r *http.Request, acc *auth.Account, _ string) {
	system, game, ok := a.coverTarget(w, r, acc)
	if !ok {
		return
	}
	if err := a.covers.Reset(r.Context(), system, game); err != nil {
		writeErr(w, err)
		return
	}
	a.auth.Audit(r.Context(), acc, "cover.reset", game)
	writeJson(w, http.StatusOK, map[string]any{"ok": true})
}
