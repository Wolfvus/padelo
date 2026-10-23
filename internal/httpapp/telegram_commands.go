package httpapp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Wolfvus/padelo/internal/store"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
)

func (a *App) processTelegram(ctx context.Context, updateID, chatID int64, text string) (string, error) {
	lower := strings.ToLower(text)
	switch lower {
	case "/start", "/help":
		return "Padel Elo bot\n\nRecord a match:\n1. Send teams: Player A, Player B - Player C, Player D\n2. Send a score: 6-4\n\nCheck rankings with: leaderboard", nil
	case "leaderboard":
		rs, err := a.store.Rankings(ctx)
		if err != nil {
			return "", err
		}
		parts := make([]string, len(rs))
		for i, p := range rs {
			parts[i] = fmt.Sprintf("%d. %s: %.2f", p.Position, p.Name, p.Elo)
		}
		if len(parts) == 0 {
			return "No players found.", nil
		}
		return strings.Join(parts, "\n"), nil
	case "history":
		ms, err := a.store.RecentMatches(ctx, 10)
		if err != nil {
			return "", err
		}
		if len(ms) == 0 {
			return "No match history.", nil
		}
		parts := make([]string, len(ms))
		for i, m := range ms {
			parts[i] = fmt.Sprintf("%d. %s def. %s (%d-%d)", i+1, strings.Join(m.Winners, " and "), strings.Join(m.Losers, " and "), m.WinnerScore, m.LoserScore)
		}
		return strings.Join(parts, "\n"), nil
	case "rollback":
		m, _, err := a.store.RollbackLatest(ctx)
		if errors.Is(err, store.ErrNotFound) {
			return "No matches to rollback.", nil
		}
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Rolled back match %s.", m.ID), nil
	}
	if m := addRE.FindStringSubmatch(text); m != nil {
		rating := 1200.0
		if m[2] != "" {
			v, err := strconv.ParseFloat(m[2], 64)
			if err != nil {
				return "invalid", nil
			}
			rating = v
		}
		name := store.NormalizeName(m[1])
		if err := a.store.CreatePlayer(ctx, name, rating); err != nil {
			return "", err
		}
		return fmt.Sprintf("Added %s with Elo %.2f.", name, rating), nil
	}
	if m := removeRE.FindStringSubmatch(text); m != nil {
		name := store.NormalizeName(m[1])
		if err := a.store.DeletePlayer(ctx, name); err != nil {
			return "", err
		}
		return fmt.Sprintf("Removed %s.", name), nil
	}
	if m := adjustRE.FindStringSubmatch(text); m != nil {
		d, _ := strconv.ParseFloat(m[2], 64)
		name := store.NormalizeName(m[1])
		rating, err := a.store.AdjustPlayerElo(ctx, name, d)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Adjusted %s to %.2f.", name, rating), nil
	}
	if m := teamRE.FindStringSubmatch(text); m != nil {
		a1, a2, b1, b2 := store.NormalizeName(m[1]), store.NormalizeName(m[2]), store.NormalizeName(m[3]), store.NormalizeName(m[4])
		seen := map[string]bool{}
		for _, n := range []string{a1, a2, b1, b2} {
			k := strings.ToLower(n)
			if n == "" || seen[k] {
				return "A match must contain four distinct players.", nil
			}
			seen[k] = true
		}
		if err := a.store.SetActiveTeams(ctx, strconv.FormatInt(chatID, 10), [2]string{a1, a2}, [2]string{b1, b2}); err != nil {
			return "", err
		}
		return fmt.Sprintf("Active teams set: %s and %s vs %s and %s. Send the score next, for example 6-4.", a1, a2, b1, b2), nil
	}
	lines := strings.Split(text, "\n")
	scores := [][2]int{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		m := scoreRE.FindStringSubmatch(line)
		if m == nil {
			return "invalid", nil
		}
		x, _ := strconv.Atoi(m[1])
		y, _ := strconv.Atoi(m[2])
		if x == y {
			return "A match cannot end in a tie.", nil
		}
		scores = append(scores, [2]int{x, y})
	}
	if len(scores) > 0 {
		ta, tb, err := a.store.ActiveTeams(ctx, strconv.FormatInt(chatID, 10))
		if errors.Is(err, store.ErrNotFound) {
			return "No active teams found. Send teams first in this format: Player A, Player B - Player C, Player D", nil
		}
		if err != nil {
			return "", err
		}
		parts := []string{}
		for i, sc := range scores {
			m, _, err := a.store.RecordMatch(ctx, store.RecordInput{Team1: ta, Team2: tb, Score1: sc[0], Score2: sc[1], K: 32, ID: fmt.Sprintf("tg-%d-%d", updateID, i)})
			if err != nil {
				return "", err
			}
			parts = append(parts, fmt.Sprintf("Recorded %s and %s vs %s and %s (%d-%d).", ta[0], ta[1], tb[0], tb[1], sc[0], sc[1]))
			for j, c := range m.Changes {
				parts = append(parts, fmt.Sprintf("%d. %s: %.2f -> %.2f", j+1, c.Name, c.Before, c.After))
			}
		}
		return strings.Join(parts, "\n"), nil
	}
	return "invalid", nil
}

func (a *App) storeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrUnknownPlayer):
		writeError(w, 404, err.Error())
	case errors.Is(err, store.ErrDuplicatePlayer), errors.Is(err, store.ErrDuplicateMatch), errors.Is(err, store.ErrPlayerHasHistory):
		writeError(w, 409, err.Error())
	case errors.Is(err, store.ErrNotFound):
		writeError(w, 404, "Not found")
	default:
		if strings.Contains(strings.ToLower(err.Error()), "invalid") || strings.Contains(strings.ToLower(err.Error()), "must") || strings.Contains(strings.ToLower(err.Error()), "cannot") {
			writeError(w, 400, err.Error())
		} else {
			a.dbError(w, err)
		}
	}
}

func (a *App) dbError(w http.ResponseWriter, err error) {
	a.log.Error("database request failed", "error", err)
	writeError(w, 500, "Internal server error")
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any, limit int64) error {
	return decodeJSONWithMode(w, r, dst, limit, true)
}

func decodeJSONLoose(w http.ResponseWriter, r *http.Request, dst any, limit int64) error {
	return decodeJSONWithMode(w, r, dst, limit, false)
}

func decodeJSONWithMode(w http.ResponseWriter, r *http.Request, dst any, limit int64, strict bool) error {
	ct := r.Header.Get("Content-Type")
	mt, _, err := mime.ParseMediaType(ct)
	if err != nil || mt != "application/json" {
		return errors.New("Content-Type must be application/json")
	}
	r.Body = http.MaxBytesReader(w, r.Body, limit)
	dec := json.NewDecoder(r.Body)
	if strict {
		dec.DisallowUnknownFields()
	}
	if err := dec.Decode(dst); err != nil {
		return errors.New("Invalid JSON body")
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return errors.New("JSON body must contain a single object")
	}
	return nil
}
