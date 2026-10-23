package httpapp

import (
	"fmt"
	"github.com/Wolfvus/padelo/internal/store"
	"net/http"
	"strings"
	"time"
)

func (a *App) createPlayer(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	var body struct {
		Name string   `json:"name"`
		Elo  *float64 `json:"elo"`
	}
	if err := decodeJSON(w, r, &body, 8<<10); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	rating := 1200.0
	if body.Elo != nil {
		rating = *body.Elo
	}
	err := a.store.CreatePlayer(r.Context(), body.Name, rating)
	if err != nil {
		a.storeError(w, err)
		return
	}
	rs, _ := a.store.Rankings(r.Context())
	name := store.NormalizeName(body.Name)
	for _, p := range rs {
		if strings.EqualFold(p.Name, name) {
			writeJSON(w, 201, p)
			return
		}
	}
	writeJSON(w, 201, map[string]any{"name": name, "elo": rating})
}

func (a *App) adjustPlayer(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	var body struct {
		Delta float64 `json:"delta"`
	}
	if err := decodeJSON(w, r, &body, 4<<10); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	name, err := pathParam(r, "player")
	if err != nil {
		writeError(w, 400, "Invalid player")
		return
	}
	rating, err := a.store.AdjustPlayerElo(r.Context(), name, body.Delta)
	if err != nil {
		a.storeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"name": store.NormalizeName(name), "elo": rating, "delta": body.Delta})
}

func (a *App) renamePlayer(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	var body struct {
		Name string `json:"name"`
	}
	if err := decodeJSON(w, r, &body, 8<<10); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	name, _ := pathParam(r, "player")
	renamed, err := a.store.RenamePlayer(r.Context(), name, body.Name)
	if err != nil {
		a.storeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"name": renamed})
}

func (a *App) deletePlayer(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	name, _ := pathParam(r, "player")
	if err := a.store.DeletePlayer(r.Context(), name); err != nil {
		a.storeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]string{"message": fmt.Sprintf("Player %s removed", store.NormalizeName(name))})
}

func (a *App) playerProfile(w http.ResponseWriter, r *http.Request) {
	name, _ := pathParam(r, "player")
	rs, err := a.store.Rankings(r.Context())
	if err != nil {
		a.dbError(w, err)
		return
	}
	var player *store.Ranking
	for i := range rs {
		if strings.EqualFold(rs[i].Name, name) {
			player = &rs[i]
			break
		}
	}
	if player == nil {
		writeError(w, 404, "Unknown player")
		return
	}
	ms, err := a.store.RecentMatches(r.Context(), 10000)
	if err != nil {
		a.dbError(w, err)
		return
	}
	key := strings.ToLower(player.Name)
	filtered := make([]store.Match, 0)
	wins := 0
	teammates := map[string]struct{ Played, Wins int }{}
	results := []bool{}
	var totalDelta float64
	var changeCount int
	bestElo := player.Elo
	for _, m := range ms {
		inW := containsFold(m.Winners, key)
		inL := containsFold(m.Losers, key)
		if !inW && !inL {
			continue
		}
		filtered = append(filtered, m)
		won := inW
		if won {
			wins++
		}
		results = append(results, won)
		team := m.Losers
		if won {
			team = m.Winners
		}
		for _, n := range team {
			if strings.ToLower(n) != key {
				s := teammates[n]
				s.Played++
				if won {
					s.Wins++
				}
				teammates[n] = s
				break
			}
		}
		for _, c := range m.Changes {
			if strings.EqualFold(c.Name, player.Name) {
				totalDelta += c.Delta
				changeCount++
				if c.Before > bestElo {
					bestElo = c.Before
				}
				if c.After > bestElo {
					bestElo = c.After
				}
			}
		}
	}
	streak := 0
	var streakObj any = nil
	if len(results) > 0 {
		kind := results[0]
		for _, v := range results {
			if v != kind {
				break
			}
			streak++
		}
		k := "loss"
		if kind {
			k = "win"
		}
		streakObj = map[string]any{"kind": k, "count": streak}
	}
	bestName := ""
	bp, bw := -1, -1
	for n, s := range teammates {
		if s.Wins > bw || (s.Wins == bw && s.Played > bp) {
			bestName = n
			bp = s.Played
			bw = s.Wins
		}
	}
	var bestMate any = nil
	if bestName != "" {
		s := teammates[bestName]
		bestMate = map[string]any{"name": bestName, "played": s.Played, "wins": s.Wins}
	}
	now := time.Now()
	thisMonth := 0
	for _, m := range filtered {
		t := m.CreatedAt.In(now.Location())
		if t.Year() == now.Year() && t.Month() == now.Month() {
			thisMonth++
		}
	}
	form := []string{}
	for i, v := range results {
		if i >= 5 {
			break
		}
		if v {
			form = append(form, "W")
		} else {
			form = append(form, "L")
		}
	}
	avg := 0.0
	if changeCount > 0 {
		avg = round2(totalDelta / float64(changeCount))
	}
	winRate := 0
	if len(filtered) > 0 {
		winRate = int(float64(wins)/float64(len(filtered))*100 + 0.5)
	}
	recent := filtered
	if len(recent) > 10 {
		recent = recent[:10]
	}
	var last any = nil
	if len(filtered) > 0 {
		last = filtered[0].CreatedAt
	}
	writeJSON(w, 200, map[string]any{"name": player.Name, "elo": player.Elo, "position": player.Position, "wins": wins, "losses": len(filtered) - wins, "matches": recent, "elo_change": round2(totalDelta), "last_played": last, "analytics": map[string]any{"win_rate": winRate, "average_delta": avg, "streak": streakObj, "best_teammate": bestMate, "recent_form": form, "matches_this_month": thisMonth, "best_elo": round2(bestElo)}})
}
