package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env                    string
	Addr                   string
	DatabaseURL            string
	PublicOrigin           string
	WebDir                 string
	AdminPassword          string
	SessionTTL             time.Duration
	TelegramWebhookSecret  string
	TelegramAllowedChatIDs map[int64]struct{}
	TelegramEnabled        bool
	TrustCloudflareIP      bool
}

func Load() (Config, error) {
	cfg := Config{
		Env:                   getenv("APP_ENV", "development"),
		Addr:                  ":" + getenv("PORT", "8080"),
		DatabaseURL:           strings.TrimSpace(os.Getenv("DATABASE_URL")),
		PublicOrigin:          strings.TrimRight(strings.TrimSpace(os.Getenv("PUBLIC_ORIGIN")), "/"),
		WebDir:                getenv("WEB_DIR", "web"),
		AdminPassword:         os.Getenv("WEB_ADMIN_PASSWORD"),
		SessionTTL:            12 * time.Hour,
		TelegramWebhookSecret: strings.TrimSpace(os.Getenv("TELEGRAM_WEBHOOK_SECRET")),
		TrustCloudflareIP:     parseBool(os.Getenv("TRUST_CLOUDFLARE_IP")),
	}
	cfg.TelegramEnabled = cfg.TelegramWebhookSecret != ""

	if raw := strings.TrimSpace(os.Getenv("SESSION_TTL_HOURS")); raw != "" {
		h, err := strconv.Atoi(raw)
		if err != nil || h < 1 || h > 168 {
			return Config{}, errors.New("SESSION_TTL_HOURS must be an integer between 1 and 168")
		}
		cfg.SessionTTL = time.Duration(h) * time.Hour
	}

	ids, err := parseChatIDs(os.Getenv("TELEGRAM_ALLOWED_CHAT_IDS"))
	if err != nil {
		return Config{}, err
	}
	cfg.TelegramAllowedChatIDs = ids

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Production() bool { return strings.EqualFold(c.Env, "production") }

func (c Config) CookieName() string {
	if c.Production() {
		return "__Host-padelo_session"
	}
	return "padelo_session"
}

func (c Config) Validate() error {
	if c.DatabaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	dbURL, err := url.Parse(c.DatabaseURL)
	if err != nil || (dbURL.Scheme != "postgres" && dbURL.Scheme != "postgresql") || dbURL.Host == "" {
		return errors.New("DATABASE_URL must be a PostgreSQL connection URL")
	}
	if c.Production() {
		sslmode := strings.ToLower(dbURL.Query().Get("sslmode"))
		// "require" encrypts but skips certificate verification under pgx.
		if sslmode != "verify-ca" && sslmode != "verify-full" {
			return errors.New("DATABASE_URL must use sslmode=verify-ca or verify-full in production")
		}
	}
	if len(c.AdminPassword) < 16 {
		return errors.New("WEB_ADMIN_PASSWORD must be at least 16 characters")
	}
	if c.PublicOrigin == "" {
		return errors.New("PUBLIC_ORIGIN is required")
	}
	u, err := url.Parse(c.PublicOrigin)
	if err != nil || u.Scheme == "" || u.Host == "" || u.Path != "" {
		return errors.New("PUBLIC_ORIGIN must be an origin such as https://padelo.example.com")
	}
	if c.Production() && u.Scheme != "https" {
		return errors.New("PUBLIC_ORIGIN must use https in production")
	}
	if c.TelegramEnabled {
		if len(c.TelegramWebhookSecret) < 32 || len(c.TelegramWebhookSecret) > 256 {
			return errors.New("TELEGRAM_WEBHOOK_SECRET must be between 32 and 256 characters")
		}
		if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(c.TelegramWebhookSecret) {
			return errors.New("TELEGRAM_WEBHOOK_SECRET contains unsupported characters")
		}
		if len(c.TelegramAllowedChatIDs) == 0 {
			return errors.New("TELEGRAM_ALLOWED_CHAT_IDS is required when Telegram is enabled")
		}
	}
	return nil
}

func (c Config) IsAllowedTelegramChat(id int64) bool {
	_, ok := c.TelegramAllowedChatIDs[id]
	return ok
}

func parseChatIDs(raw string) (map[int64]struct{}, error) {
	out := map[int64]struct{}{}
	for _, item := range strings.Split(raw, ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		id, err := strconv.ParseInt(item, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid TELEGRAM_ALLOWED_CHAT_IDS value %q", item)
		}
		out[id] = struct{}{}
	}
	return out, nil
}

func getenv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func parseBool(v string) bool {
	v = strings.TrimSpace(strings.ToLower(v))
	return v == "1" || v == "true" || v == "yes"
}
