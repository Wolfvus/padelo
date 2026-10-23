package httpapp

import (
	"context"
	"github.com/Wolfvus/padelo/internal/security"
	"net/http"
	"strconv"
	"time"
)

func (a *App) index(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	http.ServeFile(w, r, a.cfg.WebDir+"/index.html")
}

func (a *App) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := a.store.Health(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "unhealthy")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	var body struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &body, 4<<10); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if len(body.Password) < 1 || len(body.Password) > 512 {
		writeError(w, http.StatusBadRequest, "Invalid password")
		return
	}
	key := security.ClientKey(r, a.cfg.TrustCloudflareIP)
	now := time.Now()
	if wait := a.limiter.RetryAfter(key, now); wait > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(max(1, int(wait.Seconds()))))
		writeError(w, http.StatusTooManyRequests, "Too many login attempts. Try again later.")
		return
	}
	if !security.ConstantTimeEqual(body.Password, a.cfg.AdminPassword) {
		wait := a.limiter.Failure(key, now)
		if wait > 0 {
			w.Header().Set("Retry-After", strconv.Itoa(int(wait.Seconds())))
		}
		writeError(w, http.StatusUnauthorized, "Invalid password")
		return
	}
	a.limiter.Success(key)
	raw, hash, err := security.NewOpaqueToken()
	if err != nil {
		writeError(w, 500, "Internal server error")
		return
	}
	expires := now.Add(a.cfg.SessionTTL)
	a.store.PruneSessions(r.Context())
	if err := a.store.CreateSession(r.Context(), hash, "admin", expires); err != nil {
		a.log.Error("create session", "error", err)
		writeError(w, 500, "Internal server error")
		return
	}
	http.SetCookie(w, &http.Cookie{Name: a.cfg.CookieName(), Value: raw, Path: "/", HttpOnly: true, Secure: a.cfg.Production(), SameSite: http.SameSiteStrictMode, MaxAge: int(a.cfg.SessionTTL.Seconds()), Expires: expires})
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, map[string]any{"message": "Authenticated", "role": "admin"})
}

func (a *App) sessionStatus(w http.ResponseWriter, r *http.Request) {
	_, ok := a.currentRole(r)
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]bool{"authenticated": ok})
}

func (a *App) sessionRole(w http.ResponseWriter, r *http.Request) {
	role, ok := a.currentRole(r)
	if !ok {
		writeError(w, 401, "Not authenticated")
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]any{"role": role, "can_write": role == "admin"})
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(a.cfg.CookieName()); err == nil {
		_ = a.store.DeleteSession(r.Context(), security.HashToken(c.Value))
	}
	http.SetCookie(w, &http.Cookie{Name: a.cfg.CookieName(), Value: "", Path: "/", HttpOnly: true, Secure: a.cfg.Production(), SameSite: http.SameSiteStrictMode, MaxAge: -1, Expires: time.Unix(1, 0)})
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, map[string]string{"message": "Logged out"})
}
