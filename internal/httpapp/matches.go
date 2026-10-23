package httpapp

import (
	"fmt"
	"github.com/Wolfvus/padelo/internal/store"
	"net/http"
	"regexp"
	"strconv"
)

type createMatchBody struct {
	Team1  []string         `json:"team1"`
	Team2  []string         `json:"team2"`
	Score1 int              `json:"score1"`
	Score2 int              `json:"score2"`
	Sets   []store.SetScore `json:"sets,omitempty"`
}

func (a *App) matches(w http.ResponseWriter, r *http.Request) {
	limit := 25
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 || n > 100 {
			writeError(w, 400, "Limit must be between 1 and 100")
			return
		}
		limit = n
	}
	ms, err := a.store.RecentMatches(r.Context(), limit)
	if err != nil {
		a.dbError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"matches": ms})
}

func (a *App) createMatch(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	var body createMatchBody
	if err := decodeJSON(w, r, &body, 16<<10); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if len(body.Team1) != 2 || len(body.Team2) != 2 {
		writeError(w, 400, "Each team must contain exactly two players")
		return
	}
	scores := []store.SetScore{{Team1: body.Score1, Team2: body.Score2}}
	if len(body.Sets) > 0 {
		if len(body.Sets) > 5 {
			writeError(w, 400, "At most five matches may be recorded at once")
			return
		}
		scores = body.Sets
	}
	results := []store.Match{}
	var rankings []store.Ranking
	for _, sc := range scores {
		m, rs, err := a.store.RecordMatch(r.Context(), store.RecordInput{Team1: [2]string{body.Team1[0], body.Team1[1]}, Team2: [2]string{body.Team2[0], body.Team2[1]}, Score1: sc.Team1, Score2: sc.Team2, K: 32, ID: randomID("web-")})
		if err != nil {
			a.storeError(w, err)
			return
		}
		results = append(results, m)
		rankings = rs
	}
	latest := results[len(results)-1]
	msg := fmt.Sprintf("Recorded %s and %s over %s and %s (%d-%d).", latest.Winners[0], latest.Winners[1], latest.Losers[0], latest.Losers[1], latest.WinnerScore, latest.LoserScore)
	if len(results) > 1 {
		msg = fmt.Sprintf("%d matches recorded.", len(results))
	}
	writeJSON(w, 201, map[string]any{"message": msg, "match": latest, "matches": results, "rankings": rankings})
}

func (a *App) rollback(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	m, rs, err := a.store.RollbackLatest(r.Context())
	if err != nil {
		a.storeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"message": "Match rolled back.", "match": m, "rankings": rs})
}

func (a *App) legacyCreateMatch(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	var body struct {
		Winners     []string `json:"winners"`
		Losers      []string `json:"losers"`
		WinnerScore int      `json:"winner_score"`
		LoserScore  int      `json:"loser_score"`
		K           float64  `json:"k"`
	}
	if err := decodeJSON(w, r, &body, 16<<10); err != nil {
		writeError(w, 400, err.Error())
		return
	}
	if len(body.Winners) != 2 || len(body.Losers) != 2 {
		writeError(w, 400, "Each team must contain exactly two players")
		return
	}
	k := body.K
	if k == 0 {
		k = 32
	}
	m, rs, err := a.store.RecordMatch(r.Context(), store.RecordInput{Team1: [2]string{body.Winners[0], body.Winners[1]}, Team2: [2]string{body.Losers[0], body.Losers[1]}, Score1: body.WinnerScore, Score2: body.LoserScore, K: k})
	if err != nil {
		a.storeError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"message": fmt.Sprintf("Recorded %s and %s over %s and %s (%d-%d).", m.Winners[0], m.Winners[1], m.Losers[0], m.Losers[1], m.WinnerScore, m.LoserScore), "match": m, "rankings": rs})
}

// Telegram webhook
var teamRE = regexp.MustCompile(`^\s*([^,\n]+)\s*,\s*([^\-\n]+)\s*-\s*([^,\n]+)\s*,\s*([^\n]+)\s*$`)

var scoreRE = regexp.MustCompile(`^\s*(\d{1,2})\s*-\s*(\d{1,2})\s*$`)
