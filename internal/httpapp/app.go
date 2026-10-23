package httpapp

import (
	"github.com/Wolfvus/padelo/internal/config"
	"github.com/Wolfvus/padelo/internal/security"
	"github.com/Wolfvus/padelo/internal/store"
	"log/slog"
	"net/http"
	"strings"
)

type App struct {
	cfg        config.Config
	store      *store.Store
	limiter    *security.LoginLimiter
	log        *slog.Logger
	publicHost string
}

func New(cfg config.Config, st *store.Store, logger *slog.Logger) http.Handler {
	u, _ := urlParseOrigin(cfg.PublicOrigin)
	a := &App{cfg: cfg, store: st, limiter: security.NewLoginLimiter(), log: logger, publicHost: u}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", a.health)
	mux.HandleFunc("GET /rankings", a.rankingsLegacy)
	mux.HandleFunc("POST /matches", a.legacyCreateMatch)
	mux.HandleFunc("POST /telegram/webhook", a.telegramWebhook)

	mux.HandleFunc("POST /api/v1/session", a.login)
	mux.HandleFunc("GET /api/v1/session", a.sessionStatus)
	mux.HandleFunc("GET /api/v1/session/role", a.sessionRole)
	mux.HandleFunc("DELETE /api/v1/session", a.logout)
	mux.HandleFunc("GET /api/v1/rankings", a.rankings)
	mux.HandleFunc("GET /api/v1/players", a.players)
	mux.HandleFunc("POST /api/v1/players", a.createPlayer)
	mux.HandleFunc("GET /api/v1/players/{player}/profile", a.playerProfile)
	mux.HandleFunc("PATCH /api/v1/players/{player}/elo", a.adjustPlayer)
	mux.HandleFunc("PATCH /api/v1/players/{player}", a.renamePlayer)
	mux.HandleFunc("DELETE /api/v1/players/{player}", a.deletePlayer)
	mux.HandleFunc("POST /api/v1/matches", a.createMatch)
	mux.HandleFunc("GET /api/v1/matches", a.matches)
	mux.HandleFunc("POST /api/v1/matches/latest/rollback", a.rollback)
	mux.HandleFunc("GET /api/v1/next-match", a.getNextMatch)
	mux.HandleFunc("PUT /api/v1/next-match", a.saveNextMatch)
	mux.HandleFunc("DELETE /api/v1/next-match", a.clearNextMatch)

	static := http.StripPrefix("/static/", http.FileServer(http.Dir(cfg.WebDir)))
	mux.Handle("GET /static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		static.ServeHTTP(w, r)
	}))
	mux.HandleFunc("GET /{$}", a.index)

	return a.recoverer(a.securityHeaders(a.hostGuard(a.sameOriginGuard(mux))))
}

func (a *App) hostGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" && r.URL.Path != "/telegram/webhook" && !strings.EqualFold(r.Host, a.publicHost) {
			writeError(w, http.StatusMisdirectedRequest, "Invalid host")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) sameOriginGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mutating := r.Method != http.MethodGet && r.Method != http.MethodHead && r.Method != http.MethodOptions
		browserAPI := strings.HasPrefix(r.URL.Path, "/api/v1/") || r.URL.Path == "/matches"
		if mutating && browserAPI {
			if err := security.RequireOrigin(r, a.cfg.PublicOrigin); err != nil {
				writeError(w, http.StatusForbidden, "Invalid request origin")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		h.Set("Cross-Origin-Opener-Policy", "same-origin")
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self'; font-src 'self'; frame-ancestors 'none'; base-uri 'none'; form-action 'self'; manifest-src 'self'; worker-src 'self'")
		if a.cfg.Production() {
			h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				a.log.Error("panic", "error", v, "path", r.URL.Path)
				writeError(w, http.StatusInternalServerError, "Internal server error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
