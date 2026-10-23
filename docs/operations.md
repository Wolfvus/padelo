# Operations runbook

## Health and startup

`GET /health` performs a PostgreSQL ping with a two-second timeout. A 200 response with `{"status":"ok"}` confirms the process can reach the database; 503 means the process is running but PostgreSQL is unavailable.

Startup logs are JSON. Common fatal messages are:

- `invalid configuration`: correct the named environment value.
- `database initialization failed`: check URL syntax, TLS mode, networking, and credentials.
- `database migration failed`: inspect PostgreSQL permissions and the failing schema statement.

The pool permits up to six connections, idles unused connections after five minutes, and checks health every 30 seconds. Include this capacity when sizing PostgreSQL connection limits across multiple instances.

## Backups and recovery

Use provider-native backups or PostgreSQL dumps before deployments that affect schema or critical write behavior. Regularly test restoring into a separate database. Never test recovery against production.

For data corruption, first stop administrative writes, preserve logs and a database snapshot, determine the affected rows and time range, and then choose a targeted repair or restore. Avoid broad manual rating edits without retaining an audit record.

## Session incidents

Logout revokes one session. After password disclosure or suspected session theft:

1. Rotate `WEB_ADMIN_PASSWORD`.
2. Revoke every current browser session:

   ```sql
   DELETE FROM web_sessions;
   ```

3. Redeploy or restart if required by the platform's secret handling.
4. Review recent match and player changes for unauthorized activity.

## Match mistakes

The admin rollback operation reverses only the newest match and restores exact previous Elo values. Repeat it only when each newest match is known to be wrong. For older mistakes, take a backup and plan a deliberate data repair; removing a historical row without replaying subsequent ratings makes history inconsistent.

## Next-match operations

Admins schedule or clear the next match from the settings menu. With no stored schedule, the public card is hidden and the API returns `null`. There is no automatic expiry, timezone conversion, or notification; clear or replace stale entries manually.

## Telegram incidents

- Forged-request concern: rotate the webhook secret in both the service and Telegram.
- Wrong chat receiving access: remove its ID from `TELEGRAM_ALLOWED_CHAT_IDS` and redeploy.
- Duplicate delivery: check `telegram_updates`; known update IDs are intentionally ignored.
- Command failed after acknowledgement: inspect logs and database state, then apply the intended action manually. Claimed update IDs are not automatically retried.
- Full disable: unset `TELEGRAM_WEBHOOK_SECRET`, redeploy, and optionally delete the Telegram webhook.

## Production smoke test

After every deployment, verify health, public rankings, match history, player profiles, next-match presence/absence, admin authentication, and logout. Exercise writes only with a planned reversible test or during an actual authorized operation.
