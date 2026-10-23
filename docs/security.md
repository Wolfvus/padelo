# Security model

Padelo is hardened for a small internet-facing group service. The server, not the browser, is the authorization boundary.

## Production invariants

1. Set `APP_ENV=production`.
2. Set `PUBLIC_ORIGIN` to the exact HTTPS browser origin.
3. Use a unique random `WEB_ADMIN_PASSWORD` of at least 16 characters; 24 or more is preferable.
4. Use a PostgreSQL `DATABASE_URL` with `sslmode=verify-full` (or `verify-ca`); `require` does not verify the server certificate.
5. Keep Telegram disabled unless it is needed. When enabled, configure both a strong webhook secret and an explicit chat allowlist.
6. Never commit `.env`, database credentials, admin passwords, Telegram tokens, webhook secrets, or hosting secrets.
7. Back up PostgreSQL before application or schema migrations.

Configuration validation fails closed when these production requirements are not met.

## Access model

Rankings, players, profiles, match history, health, and next-match details are public by design. Player changes, manual Elo changes, match recording, rollback, and next-match scheduling require an admin session. Compatibility write routes have the same admin requirement.

UI visibility does not grant or deny access. Administrative handlers call server-side authorization regardless of which controls the browser displays.

## Sessions and credentials

Successful login creates 32 cryptographically random bytes encoded as an opaque token. The browser receives only an `HttpOnly`, `SameSite=Strict` cookie; PostgreSQL stores only its SHA-256 hash with role and expiry. Logout deletes that row. Expired rows are invalid and are pruned opportunistically on login.

Production cookies use the `__Host-` prefix, `Secure`, `HttpOnly`, path `/`, and no Domain attribute. Password comparison is constant-time. Admin tokens and passwords are never placed in localStorage or returned in JSON.

The in-process login limiter allows four ordinary failures per client key within ten minutes; the fifth creates a 15-minute lockout. A global backstop responds after 60 failures in the rolling window. Limits reset when the process restarts and are not coordinated across instances.

By default, client identity uses the TCP peer. Enable `TRUST_CLOUDFLARE_IP` only when the origin cannot be reached except through trusted Cloudflare infrastructure; otherwise an attacker could spoof `CF-Connecting-IP`.

## Browser request integrity

Mutating `/api/v1/*` and compatibility `/matches` requests require an `Origin` exactly equal to `PUBLIC_ORIGIN`, in addition to the admin cookie. Request bodies have endpoint-specific size limits and strict browser JSON decoding rejects unknown fields, malformed input, multiple values, and incorrect content types.

The host guard rejects unexpected hosts except on `/health` and `/telegram/webhook`. Responses include a restrictive Content Security Policy, frame denial, no-sniff, no-referrer, permissions restrictions, and cross-origin opener isolation. Production also sends one year of HSTS including subdomains.

## Telegram trust boundary

The webhook secret header is verified with constant-time comparison before request decoding. Only configured chat IDs may run commands. Valid update IDs are persisted before command processing so Telegram retries cannot repeat a mutation. Telegram requests do not use browser cookies or browser origin validation.

The Telegram bot is an administrative interface: allowed chat members can add, remove, and adjust players, record and roll back matches. Treat chat membership and the bot account as privileged.

## Database integrity

Match recording, rollback, and rename operations use serializable PostgreSQL transactions. Match recording locks player rows in stable order, updates all four ratings, and inserts history atomically. Each match stores exact before/after changes so rollback does not recompute historical values. Database constraints bound ratings, scores, session roles, names, and the singleton next match.

## Secret rotation

- Admin password: rotate the platform secret and delete all rows from `web_sessions`.
- Telegram webhook secret: update the application and re-register Telegram's webhook with the same new secret.
- Database credentials: rotate with the provider, update `DATABASE_URL`, and redeploy; confirm health before revoking old credentials when overlap is available.

Do not log secret values or paste them into issues, commits, screenshots, or support messages.

## Scope and limitations

The application does not provide named administrator accounts, MFA, per-user audit trails, distributed rate limiting, a WAF, multi-region session replication, automatic next-match expiry, or automated database backups. A single shared admin password means actions cannot be attributed to an individual.

Report suspected vulnerabilities privately to the repository owner. Include affected routes, reproduction steps, impact, and any suggested mitigation without including live credentials or production personal data.
