# Deployment

`render.yaml` describes a native Go Render web service. The same principles apply on another platform: build one binary, serve the `web/` directory from the working directory, and provide PostgreSQL plus the required secrets.

## VPS with Docker and Caddy

The repository includes `Dockerfile` and `deploy/vps/docker-compose.yml` for a VPS that already runs Caddy in Docker. The compose project attaches Padelo to the existing Caddy network only; it does not publish a host port or modify other applications.

1. Clone or update this repository at `/opt/padelo` on the VPS.
2. Copy `deploy/vps/.env.example` to `deploy/vps/.env`, set every required value, and keep it readable only by root.
3. Point the public hostname's DNS A record to the VPS before enabling its Caddy route. If the DNS record is Cloudflare-proxied, set `TRUST_CLOUDFLARE_IP=true`; otherwise set it to `false`.
4. Add a Caddy site block that reverse-proxies the hostname to `padelo:8080`, then validate and reload Caddy.
5. Run `docker compose -f deploy/vps/docker-compose.yml up -d --build` from the repository root.

Set `CADDY_NETWORK` in `deploy/vps/.env` to your reverse proxy’s existing Docker network (defaults to `caddy`). Configure your own hostname and infrastructure.

## Environment variables

| Variable | Required | Description |
| --- | --- | --- |
| `APP_ENV` | Yes | Use `production` in production; defaults to `development` |
| `PORT` | Platform | Listen port; defaults to `8080` |
| `DATABASE_URL` | Yes | PostgreSQL URL; production requires an accepted TLS `sslmode` |
| `PUBLIC_ORIGIN` | Yes | Exact origin with scheme and host, no path or trailing slash |
| `WEB_ADMIN_PASSWORD` | Yes | Unique password of at least 16 characters |
| `WEB_DIR` | No | Static directory; defaults to `web` |
| `SESSION_TTL_HOURS` | No | Admin session lifetime, 1–168; defaults to 12 |
| `TELEGRAM_WEBHOOK_SECRET` | No | Enables Telegram; 32–256 URL-safe characters |
| `TELEGRAM_ALLOWED_CHAT_IDS` | With Telegram | Comma-separated signed integer chat IDs |
| `TRUST_CLOUDFLARE_IP` | No | Trust `CF-Connecting-IP` for login limits; defaults to false |

Obsolete Python and token variables such as `WEB_APP_PASSWORD`, `WEB_SESSION_SECRET`, and `WEB_COOKIE_SECURE` are unused.

## Render commands

```text
Build:  go build -tags netgo -ldflags '-s -w' -o app ./cmd/padelo
Start:  ./app
Health: /health
```

Set every `sync: false` value in the Render dashboard. Keep the repository root as the working directory so `WEB_DIR=web` resolves correctly.

## Release sequence

1. Run the full checks from [development.md](development.md).
2. Back up PostgreSQL before schema-affecting releases.
3. Review environment values, especially production TLS, public origin, and Telegram pairing.
4. Deploy the binary and allow startup schema checks to complete.
5. Confirm `/health`, the public ranking, an admin login/logout, and the intended write flow.
6. If Telegram is enabled, send a read-only `leaderboard` command from an allowed chat.

Production uses a `__Host-` secure session cookie. TLS termination must make the public site HTTPS, and `PUBLIC_ORIGIN` must be the exact browser-visible origin or writes will fail origin validation.

## Rollback

Application rollback means deploying the prior known-good binary. Do not automatically reverse database changes. Because startup schema changes are additive, the earlier binary should generally ignore new objects, but confirm compatibility before release. Restore a database backup only for confirmed data corruption and only after preserving the failed state for investigation.
