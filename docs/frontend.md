# Frontend

The browser client in `web/` is an ES-module progressive web app with no compilation or package-manager step.

## Files

| File | Responsibility |
| --- | --- |
| `index.html` | Semantic page structure, navigation, dialogs/sheets, and module entry point |
| `styles.css` | Responsive visual system, components, states, and accessibility styles |
| `api.js` | Same-origin `/api/v1/*` request wrapper and endpoint methods |
| `state.js` | Shared client state and selection helpers |
| `app.js` | Authentication, data loading, rendering, events, and admin workflows |
| `ui-enhancements.js` | Isolated progressive interaction enhancements |
| `service-worker.js` | Static caching behavior and application update handling |
| `manifest.webmanifest` | Install metadata |
| `icons.svg` | Reusable interface icon symbols |

## Views and permissions

Ranking and history are public. Selecting a player opens a public profile. Recording and all management controls require the `admin` role. The settings-menu next-match editor remains available to admins even when no schedule exists; the public next-match card is omitted entirely when the API returns `null`.

Client-side visibility is not authorization. Every write is independently checked by the Go handler.

## 6 Loco tournament preview

Open **6 Loco tournament preview** in the ranking menu, or visit `/static/tournament.html`. You can also open `web/tournament.html` directly from a local checkout. This optional planner offers one proposed rotation on a single court. Each of six players plays four matches with four different partners and rests twice. It opens with fictional names. Edit the roster, switch between English and Spanish, or copy the rounds for the group. Names stay in page memory and reset on reload. The page runs independently of the API, sessions, rankings, and match recording.

## State and data flow

On load, the client checks session status, resolves the authenticated role, and loads rankings, recent matches, and the next match. `state.js` holds the current role, data, selected teams, scores, active view, loading flags, and errors. Mutations call the API first and then refresh affected public state.

`api.js` always sends cookies with `credentials: "same-origin"`, serializes object bodies as JSON, and turns non-success responses into errors containing the HTTP status. Add new browser calls there and keep their paths under `/api/v1/*`.

## Rendering and safety

Prefer DOM APIs and `textContent` for untrusted values. When generated HTML is appropriate, pass dynamic strings through the existing escaping helper. Preserve dialog focus restoration, Escape-to-close behavior, focus trapping, disabled/loading states, and meaningful labels.

## Service worker

The service worker may cache versioned static assets, but API and session requests must remain network-driven. When changing cached frontend assets, update the cache version so installed clients receive the new files. Never cache authenticated JSON responses.

## Verification

Frontend changes still require the Go test suite because handlers serve and authorize the UI contract. Also run JavaScript syntax checks:

```bash
node --check web/api.js
node --check web/state.js
node --check web/app.js
node --check web/ui-enhancements.js
node --check web/service-worker.js
```

Manually verify a narrow mobile viewport and a desktop viewport, keyboard-only navigation, viewer/admin visibility, expired-session behavior, empty states, and both configured and absent next-match states.
