# Padelo contributor notes

- The backend is a Go `net/http` service. Do not reintroduce the retired Python/FastAPI backend.
- Keep frontend API calls under `/api/v1/*`; server-side authorization is the source of truth.
- Keep the vanilla HTML/CSS/JavaScript frontend in `web/` unless a UI task explicitly changes that decision.
- PostgreSQL/Neon is the only persistence layer. Never point tests, experiments, or destructive scripts at production.
- Public reads are rankings, match history, player profiles, and the next match. Player management, Elo changes, match recording, rollback, and next-match scheduling are admin-only.
- Admin sessions must remain opaque, random, and revocable server-side cookies. Do not replace them with browser storage tokens.
- Mutating browser routes must retain same-origin validation and server-side admin authorization.
- Telegram handling must verify `X-Telegram-Bot-Api-Secret-Token` and the allowed chat ID before processing commands.
- Match recording, player renaming, and rollback operations must remain transactional.
- Use descriptive, responsibility-based filenames; do not add numbered source-file names.
- Update the relevant file in `docs/` when behavior, configuration, schema, deployment, or operational procedures change.
- After Go changes, run `gofmt -w ./cmd ./internal`, `go test ./...`, `go test -race ./...`, `go vet ./...`, and `go build ./...`.
