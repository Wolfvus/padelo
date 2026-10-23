# HTTP API

All browser application calls use `/api/v1/*`. JSON responses use `Content-Type: application/json`; errors have the shape:

```json
{"detail":"Human-readable message"}
```

Public routes need no session. Admin routes require the session cookie issued by `POST /api/v1/session`. Mutating browser requests must also include an `Origin` exactly matching `PUBLIC_ORIGIN`; browsers do this automatically for the same-origin application.

## Health and sessions

| Method | Route | Access | Purpose |
| --- | --- | --- | --- |
| `GET` | `/health` | Public | Ping PostgreSQL and return `{"status":"ok"}` or 503 |
| `POST` | `/api/v1/session` | Public | Authenticate with `{"password":"..."}` and set the admin cookie |
| `GET` | `/api/v1/session` | Public | Return `{"authenticated":true|false}` |
| `GET` | `/api/v1/session/role` | Admin | Return `{"role":"admin","can_write":true}` |
| `DELETE` | `/api/v1/session` | Public | Revoke the current server-side session and clear its cookie |

Login failures may return `429` with a `Retry-After` header. Session responses are marked `no-store`.

## Rankings and players

| Method | Route | Access | Body or query |
| --- | --- | --- | --- |
| `GET` | `/api/v1/rankings` | Public | None |
| `GET` | `/api/v1/players` | Public | None |
| `GET` | `/api/v1/players/{player}/profile` | Public | None |
| `POST` | `/api/v1/players` | Admin | `{"name":"Alex","elo":1200}`; `elo` is optional |
| `PATCH` | `/api/v1/players/{player}` | Admin | `{"name":"New name"}` |
| `PATCH` | `/api/v1/players/{player}/elo` | Admin | `{"delta":25}` |
| `DELETE` | `/api/v1/players/{player}` | Admin | None |

Rankings are ordered by descending Elo and include position and name. For twelve hours after the latest match, `/rankings` also reports movement and each involved player's latest Elo delta. Profiles contain win/loss totals, recent matches, and derived form, streak, teammate, and rating analytics.

Names are trimmed and internal whitespace is collapsed. They are unique case-insensitively and limited to 80 Unicode characters. A player with match history cannot be deleted; rename the player instead if history must be retained.

## Matches

`GET /api/v1/matches?limit=25` is public. `limit` defaults to 25 and must be between 1 and 100.

`POST /api/v1/matches` is admin-only. A single result uses:

```json
{
  "team1": ["Alex", "Blair"],
  "team2": ["Casey", "Drew"],
  "score1": 6,
  "score2": 3
}
```

The four players must exist and be distinct. Scores are integers from 0 through 99 and ties are rejected. The response includes the created match and current rankings.

The UI can submit up to five score entries with `sets`:

```json
{
  "team1": ["Alex", "Blair"],
  "team2": ["Casey", "Drew"],
  "score1": 6,
  "score2": 3,
  "sets": [
    {"team1": 6, "team2": 3},
    {"team1": 4, "team2": 6},
    {"team1": 10, "team2": 8}
  ]
}
```

Despite the field name, every `sets` item is recorded as an independent Elo-rated match. Items commit sequentially, so an error later in the list does not roll back earlier items.

`POST /api/v1/matches/latest/rollback` is admin-only. It deletes the newest match and restores the exact player ratings captured before that match.

## Next match

| Method | Route | Access | Purpose |
| --- | --- | --- | --- |
| `GET` | `/api/v1/next-match` | Public | Return the scheduled match or `{"next_match":null}` |
| `PUT` | `/api/v1/next-match` | Admin | Create or replace the singleton schedule |
| `DELETE` | `/api/v1/next-match` | Admin | Clear the schedule |

The PUT body is:

```json
{"date":"2026-08-25","time":"19:30","location":"Central Padel Club"}
```

Dates use `YYYY-MM-DD`, times use 24-hour `HH:MM`, and the normalized location is 1–160 characters. The API currently accepts past dates; the admin is responsible for clearing or replacing stale schedules.

## Compatibility and integration routes

- `GET /rankings` returns rankings without recent movement metadata.
- `POST /matches` accepts the former winners/losers payload but still requires an admin browser session and valid origin.
- `POST /telegram/webhook` is reserved for Telegram and uses webhook-secret and chat-allowlist authentication rather than a browser session.

Compatibility routes should not be used by new frontend code.
