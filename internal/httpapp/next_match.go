package httpapp

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Wolfvus/padelo/internal/store"
)

type nextMatchBody struct {
	Date     string `json:"date"`
	Time     string `json:"time"`
	Location string `json:"location"`
}

func normalizeNextMatchInput(body nextMatchBody) (nextMatchBody, error) {
	body.Date = strings.TrimSpace(body.Date)
	body.Time = strings.TrimSpace(body.Time)
	body.Location = strings.Join(strings.Fields(body.Location), " ")
	if _, err := time.Parse("2006-01-02", body.Date); err != nil {
		return nextMatchBody{}, fmt.Errorf("date must use YYYY-MM-DD")
	}
	if _, err := time.Parse("15:04", body.Time); err != nil {
		return nextMatchBody{}, fmt.Errorf("time must use HH:MM")
	}
	if body.Location == "" {
		return nextMatchBody{}, fmt.Errorf("location is required")
	}
	if utf8.RuneCountInString(body.Location) > 160 {
		return nextMatchBody{}, fmt.Errorf("location must be 160 characters or fewer")
	}
	return body, nil
}

func (a *App) getNextMatch(w http.ResponseWriter, r *http.Request) {
	next, err := a.store.NextMatch(r.Context())
	if errors.Is(err, store.ErrNotFound) {
		writeJSON(w, http.StatusOK, map[string]any{"next_match": nil})
		return
	}
	if err != nil {
		a.dbError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"next_match": next})
}

func (a *App) saveNextMatch(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	var body nextMatchBody
	if err := decodeJSON(w, r, &body, 8<<10); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	normalized, err := normalizeNextMatchInput(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	next, err := a.store.SaveNextMatch(r.Context(), normalized.Date, normalized.Time, normalized.Location)
	if err != nil {
		a.dbError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"next_match": next})
}

func (a *App) clearNextMatch(w http.ResponseWriter, r *http.Request) {
	if !a.requireAdmin(w, r) {
		return
	}
	if err := a.store.ClearNextMatch(r.Context()); err != nil {
		a.dbError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"next_match": nil})
}
