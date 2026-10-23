package httpapp

import (
	"context"
	"github.com/Wolfvus/padelo/internal/security"
	"github.com/Wolfvus/padelo/internal/store"
	"net/http"
	"sort"
	"strings"
	"time"
)

func (a *App) currentRole(r *http.Request) (string, bool) {
	c, err := r.Cookie(a.cfg.CookieName())
	if err != nil || len(c.Value) > 128 {
		return "", false
	}
	role, err := a.store.SessionRole(r.Context(), security.HashToken(c.Value))
	return role, err == nil
}

func (a *App) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	role, ok := a.currentRole(r)
	if !ok {
		writeError(w, 401, "Not authenticated")
		return false
	}
	if role != "admin" {
		writeError(w, 403, "Admin access required")
		return false
	}
	return true
}

func (a *App) rankingsLegacy(w http.ResponseWriter, r *http.Request) {
	rs, err := a.store.Rankings(r.Context())
	if err != nil {
		a.dbError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"rankings": rs})
}

func (a *App) rankings(w http.ResponseWriter, r *http.Request) {
	rs, err := a.rankingsWithMovement(r.Context())
	if err != nil {
		a.dbError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"rankings": rs})
}

func (a *App) players(w http.ResponseWriter, r *http.Request) {
	rs, err := a.store.Rankings(r.Context())
	if err != nil {
		a.dbError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"players": rs})
}

func (a *App) rankingsWithMovement(ctx context.Context) ([]store.Ranking, error) {
	rs, err := a.store.Rankings(ctx)
	if err != nil {
		return nil, err
	}
	ms, err := a.store.RecentMatches(ctx, 1)
	if err != nil {
		return nil, err
	}
	if len(ms) == 0 || time.Since(ms[0].CreatedAt) > 12*time.Hour {
		return rs, nil
	}
	prev := map[string]float64{}
	for _, r := range rs {
		prev[strings.ToLower(r.Name)] = r.Elo
	}
	delta := map[string]float64{}
	for _, c := range ms[0].Changes {
		prev[strings.ToLower(c.Name)] = c.Before
		delta[strings.ToLower(c.Name)] = c.Delta
	}
	copyR := append([]store.Ranking(nil), rs...)
	sort.Slice(copyR, func(i, j int) bool {
		pi, pj := prev[strings.ToLower(copyR[i].Name)], prev[strings.ToLower(copyR[j].Name)]
		if pi == pj {
			return strings.ToLower(copyR[i].Name) < strings.ToLower(copyR[j].Name)
		}
		return pi > pj
	})
	prevPos := map[string]int{}
	for i, r := range copyR {
		prevPos[strings.ToLower(r.Name)] = i + 1
	}
	for i := range rs {
		key := strings.ToLower(rs[i].Name)
		rs[i].RankChange = prevPos[key] - rs[i].Position
		if d, ok := delta[key]; ok {
			v := d
			rs[i].LastDelta = &v
		}
	}
	return rs, nil
}
