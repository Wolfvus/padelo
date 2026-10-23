package httpapp

import (
	"github.com/Wolfvus/padelo/internal/security"
	"net/http"
	"regexp"
	"strings"
)

var addRE = regexp.MustCompile(`(?i)^add\s+(.+?)(?:\s+(-?\d+(?:\.\d+)?))?\s*$`)

var removeRE = regexp.MustCompile(`(?i)^remove\s+(.+?)\s*$`)

var adjustRE = regexp.MustCompile(`(?i)^adjust\s+(.+?)\s+([+-]\d+(?:\.\d+)?)\s*$`)

type tgUpdate struct {
	UpdateID int64 `json:"update_id"`
	Message  *struct {
		Chat struct {
			ID int64 `json:"id"`
		} `json:"chat"`
		Text string `json:"text"`
	} `json:"message"`
}

func (a *App) telegramWebhook(w http.ResponseWriter, r *http.Request) {
	if !a.cfg.TelegramEnabled {
		http.NotFound(w, r)
		return
	}
	if !security.ConstantTimeEqual(r.Header.Get("X-Telegram-Bot-Api-Secret-Token"), a.cfg.TelegramWebhookSecret) {
		writeError(w, 401, "Unauthorized")
		return
	}
	var u tgUpdate
	if err := decodeJSONLoose(w, r, &u, 1<<20); err != nil {
		writeError(w, 400, "Invalid update")
		return
	}
	if u.UpdateID <= 0 || u.Message == nil || !a.cfg.IsAllowedTelegramChat(u.Message.Chat.ID) {
		writeJSON(w, 200, map[string]bool{"ok": true})
		return
	}
	fresh, err := a.store.TryMarkTelegramUpdate(r.Context(), u.UpdateID)
	if err != nil {
		a.dbError(w, err)
		return
	}
	if !fresh {
		writeJSON(w, 200, map[string]bool{"ok": true})
		return
	}
	text := strings.TrimSpace(u.Message.Text)
	if text == "" {
		a.tgReply(w, u.Message.Chat.ID, "invalid")
		return
	}
	msg, err := a.processTelegram(r.Context(), u.UpdateID, u.Message.Chat.ID, text)
	if err != nil {
		msg = a.telegramErrorMessage(err)
	}
	a.tgReply(w, u.Message.Chat.ID, msg)
}

func (a *App) tgReply(w http.ResponseWriter, chat int64, text string) {
	writeJSON(w, 200, map[string]any{"method": "sendMessage", "chat_id": chat, "text": text})
}
