# Architecture

## Overview

Padelo is one deployable Go process backed by PostgreSQL. It serves the browser application, handles JSON APIs, performs Elo updates, and optionally accepts Telegram webhooks.

```text
Browser ─────── HTTPS ──────┐
                            ├─ Go net/http service ── pgx pool ── PostgreSQL
Telegram ─ webhook + secret ┘
```

There is no frontend build pipeline and no second application server. PostgreSQL is the system of record for players, ratings, matches, sessions, Telegram state, and the scheduled next match.

## Startup and shutdown

`cmd/padelo/main.go` performs startup in this order:

1. Load and validate environment configuration.
2. Create and ping the PostgreSQL connection pool.
3. create missing schema objects with `Store.EnsureSchema`.
4. Build the HTTP handler and start the server with explicit timeouts.
5. On `SIGINT` or `SIGTERM`, stop accepting requests and close cleanly.

Configuration errors and database connectivity errors fail startup. Schema creation is additive and does not seed or reset application data.

## Package responsibilities

| Path | Responsibility |
| --- | --- |
| `cmd/padelo` | Process wiring, logging, HTTP lifecycle, and shutdown |
| `internal/config` | Environment parsing, defaults, and production validation |
| `internal/elo` | Pure team-rating calculations and change records |
| `internal/httpapp` | Routes, middleware, validation, JSON responses, and Telegram command dispatch |
| `internal/security` | Opaque tokens, constant-time comparison, origin checks, and login throttling |
| `internal/store` | Schema ownership, SQL queries, and transactional state changes |
| `web` | Browser UI, client state, API adapter, styles, and service worker |

Files inside `httpapp` and `store` are grouped by domain rather than execution order. Keep that convention when adding features.

## Request processing

Every request passes through recovery, security headers, host validation, and same-origin validation before route handling. The host guard exempts health checks and Telegram webhooks because infrastructure and Telegram may use different request-host behavior. The origin guard applies to mutating browser API requests; Telegram is authenticated separately.

Authorization is always enforced in handlers. Hiding admin UI is only a presentation concern and is not a security boundary.

## Data consistency

Match recording locks all four player rows in a stable name order inside a serializable transaction. Rating updates and match insertion commit together. Rollback locks the latest match, restores each recorded pre-match rating, and deletes the match in the same transaction. Player renaming updates player, match, Elo-change, and saved Telegram-context references transactionally.

The browser batch-score input is processed as up to five distinct matches in order. Each completed match commits independently; the batch is not one database transaction.

## Elo calculation

Each team rating is the average of its two players. The standard logistic expectation uses a 400-point scale. Winners receive a positive update and losers the inverse, with a default K-factor of 32 and a bounded score-margin multiplier. Ratings and reported changes are rounded to two decimal places.
