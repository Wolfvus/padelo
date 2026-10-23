# Development guide

## Prerequisites

- Go 1.26.10 or newer
- PostgreSQL reachable through a disposable development URL
- Node.js for optional JavaScript syntax checks

Padelo has no Python runtime and no frontend dependency installation.

## Local environment

```bash
cp .env.example .env
```

Set `DATABASE_URL` to an empty local database, keep `APP_ENV=development`, and make `PUBLIC_ORIGIN` exactly match the browser origin. Never use production credentials for local work.

```bash
set -a
source .env
set +a
go run ./cmd/padelo
```

Startup creates missing schema objects. Add players through the admin UI or API; there is deliberately no seed script.

## Required checks

Run these after Go or frontend changes:

```bash
gofmt -w ./cmd ./internal
go mod tidy
go test ./...
go test -race ./...
go vet ./...
go build ./...
git diff --check
```

For frontend changes also run the `node --check` commands in [frontend.md](frontend.md) and perform browser smoke testing. The committed module files must not change unexpectedly after `go mod tidy`.

## Organization conventions

- Keep process wiring in `cmd/padelo` and application logic in `internal`.
- Group HTTP and store files by domain using descriptive filenames.
- Keep Elo math pure and independently testable.
- Keep browser endpoint calls centralized in `web/api.js`.
- Do not duplicate authorization rules in the client or trust client visibility.
- Add focused tests beside the package under test.
- Update documentation in the same change as behavior or operational assumptions.

## Adding an endpoint

1. Define the route in `internal/httpapp/app.go` under `/api/v1/*`.
2. Put validation and response handling in the domain handler file.
3. Put SQL in the matching `internal/store` domain file.
4. Apply `requireAdmin` to every administrative operation.
5. Confirm mutating browser requests remain covered by the origin guard.
6. Add the frontend method to `web/api.js` if needed.
7. Add tests and update `docs/api.md`.

Use strict JSON decoding for browser endpoints, set conservative body-size limits, and return errors through the common `{"detail":"..."}` response shape.

## Changing the schema

Read [data-model.md](data-model.md) before changing `EnsureSchema`. Startup SQL must remain idempotent. Test schema changes against a disposable PostgreSQL database and document any production sequencing or backup requirement.
