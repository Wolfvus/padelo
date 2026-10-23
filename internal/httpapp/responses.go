package httpapp

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/Wolfvus/padelo/internal/store"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"detail": msg})
}

func pathParam(r *http.Request, key string) (string, error) {
	v := r.PathValue(key)
	if v == "" {
		return "", errors.New("invalid path parameter")
	}
	return v, nil
}

func urlParseOrigin(origin string) (string, error) {
	i := strings.Index(origin, "://")
	if i < 0 {
		return "", errors.New("invalid origin")
	}
	host := origin[i+3:]
	if host == "" || strings.Contains(host, "/") {
		return "", errors.New("invalid origin")
	}
	return host, nil
}

func (a *App) telegramErrorMessage(err error) string {
	switch {
	case errors.Is(err, store.ErrUnknownPlayer):
		return "Unknown player."
	case errors.Is(err, store.ErrDuplicatePlayer):
		return "Player already exists."
	case errors.Is(err, store.ErrPlayerHasHistory):
		return "Player has match history and cannot be removed."
	case errors.Is(err, store.ErrDuplicateMatch):
		return "This match has already been processed."
	default:
		a.log.Error("telegram command failed", "error", err)
		return "Internal error."
	}
}

func randomID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return prefix + hex.EncodeToString(b[:])
}
