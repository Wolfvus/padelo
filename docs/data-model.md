# Data model

PostgreSQL is Padelo's only persistence layer. `internal/store/postgres.go` owns the additive startup schema. There is currently no external migration framework.

## Tables

### `players`

| Column | Meaning |
| --- | --- |
| `name TEXT PRIMARY KEY` | Canonical display name |
| `elo DOUBLE PRECISION` | Current rating, constrained to a broad safe range |

A unique index on `lower(name)` makes names case-insensitively unique. New players default to 1200 through the API.

### `matches`

| Column | Meaning |
| --- | --- |
| `id TEXT PRIMARY KEY` | Idempotency and ordering tie-break identifier |
| `winners TEXT[]`, `losers TEXT[]` | Two canonical player names per team |
| `winner_score`, `loser_score` | Normalized winner-first score |
| `changes_json TEXT` | Exact before, after, and delta values for each player |
| `sets_json TEXT` | Compatibility metadata; current browser batches create separate rows |
| `created_at TIMESTAMPTZ` | UTC creation time |

The stored change snapshot makes exact latest-match rollback possible even if the Elo formula changes later.

### `contexts`

Stores the last two Telegram teams for a chat in `context_key`, `team1`, and `team2`. It lets subsequent score-only messages reuse a lineup.

### `web_sessions`

Stores a SHA-256 token hash, the `admin` role, creation time, and expiry. Raw session tokens exist only in the browser cookie. An expiry index supports opportunistic pruning during login.

### `telegram_updates`

Stores processed Telegram `update_id` values. Inserting an update ID before command execution prevents webhook retries from applying a command twice.

### `next_match`

A boolean primary key constrained to `TRUE` enforces at most one row. The row contains a date, local wall-clock time, location, and update timestamp. Deleting the row represents “no next match.”

## Transaction boundaries

- Match recording: locks four players, calculates ratings, updates players, and inserts one match.
- Latest rollback: locks the newest match, restores ratings, and deletes that row.
- Player rename: locks the player and affected history, then updates every stored name reference.
- Player creation, deletion, manual Elo adjustment, sessions, next match, and Telegram context each use individual statements where a larger transaction is unnecessary.

All multi-row consistency operations use serializable transactions. Player locks are acquired in sorted order during match recording to reduce deadlock risk.

## Schema changes

For an additive, backward-compatible change, extend `EnsureSchema` with idempotent `CREATE ... IF NOT EXISTS` or `ALTER ... ADD COLUMN IF NOT EXISTS` statements. For destructive, long-running, or data-transforming changes, use an explicit reviewed migration procedure rather than hiding it in startup.

Before any production schema change:

1. Back up the database.
2. Test against a disposable database containing representative data.
3. Confirm old and new application versions can tolerate the rollout sequence.
4. Document rollback and recovery in `docs/deployment.md` or `docs/operations.md`.
