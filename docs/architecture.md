# Backend architecture

## Runtime boundaries

```text
Nuxt frontend
  | HTTP + authenticated WebSocket
  v
main.go (LISTEN_ADDR / :8080, Gorilla mux)
  +-- pkg/api + pkg/middleware + pkg/auth  -> accounts / tokens
  +-- pkg/game <-> pkg/websocket          -> in-memory lobbies and rounds
  |      +-- pkg/logic + pkg/reverse      -> bundled geography + scoring
  |      +-- Google Street View metadata -> external network / quota
  +-- pkg/db + pkg/models (GORM)          -> PostgreSQL
```

This is a **single-process** server. Lobby membership, timers, current rounds, hubs, and active results live in memory. Restarts lose active games; shared PostgreSQL alone does not enable transparent horizontal scaling. The private registry only copies/inserts/deletes exact lobby pointers; its mutex is released before taking a lobby mutex. The lobby mutex owns configuration, membership/leases/admission reservations, phase, generation, timers and scores. See [lobby lifecycle](lobby-lifecycle.md) for lock order, immutable snapshots, bounded transport effects and the exact additive frontend contract. Validate concurrency and multiplayer behavior separately from compilation.

## Entry points and ownership

| Location | Responsibility / maintenance constraint |
| --- | --- |
| `main.go` and root configuration code | Explicit startup loads/validates dotenv, configures logging, then configures JWT signing before DB/assets; it registers routes and serves on `LISTEN_ADDR` (legacy fallback `0.0.0.0:8080`). `-dev` restricts local settings; importing/compiling no longer runs main startup side effects. |
| `pkg/api`, `pkg/middleware` | HTTP decoding/responses, CORS, authorization, WebSocket upgrade and lobby admission. |
| `pkg/auth` | HS256 access/refresh JWTs require explicit signed `token_use`, expiry/subject and purpose-specific ID shape. Signing key is explicitly configured after dotenv loading and must be at least 32 characters. Legacy sessions need re-login; full session lifecycle remains separate. |
| `pkg/db`, `pkg/models` | Global GORM handle; user/guest accounts, refresh-token revocation, games, rounds, results. `ConnectDB` auto-migrates at startup. |
| `pkg/game` | Private registry (`registry.go`), detached DTOs (`snapshot.go`), exact upgrade/connection/grace lifecycle (`lifecycle.go`), generation-guarded start/finalization (`round_lifecycle.go`), configuration/guesses/scoring, and post-lock socket-close/persistence effects (`effects.go`). |
| `pkg/websocket` | Per-client read/write loops and mutex-owned private hub membership. Nonblocking 64-event peer queues contain immutable encoded events; slow peers disconnect after locks rather than skip events. A done signal/once-close replaces immortal hub loops. This is transport, not a durable state store. |
| `pkg/logic`, `pkg/reverse` | Polygon selection, distances, validated coordinates/countries and Street View metadata. Only the latter is a Google request: 2s/request, one 6s/8-call/1,024-point location budget; failures propagate before round mutation. |
| `assets/` | Country metadata, selection polygons, and reverse-geocoding data loaded by relative path. Run the server from the repository root. |
| `cmd/main.go` | Manual geography/API exploration helper, not the server or an automated test suite. API mode can spend Google quota. |

## Current public contract

Routes below are registered in `main.go`; CORS preflight is omitted here.

| Access | Method | Route |
| --- | --- | --- |
| Public | POST | `/auth/register/user`, `/auth/register/guest`, `/auth/login`, `/auth/refresh` |
| Public | GET | `/countryList`, `/countryCatalog`, `/lobby`, `/healthz`, `/readyz`, `/version` |
| Access JWT | GET | `/auth/logout` |
| Access JWT | POST | `/updateUser`, `/lobby` |
| Access JWT | DELETE | `/lobby` (currently logs a warning and returns success without deleting) |
| Access JWT | WebSocket upgrade | `/lobbySocket?id=<lobby-id>` |

Protected HTTP currently takes a **raw JWT** in `Authorization`, not a `Bearer` prefix. The browser offers WebSocket subprotocols `['json', accessToken]`; middleware reads the second value, and the server negotiates `json`. Preserve these conventions until both clients and server can change together. Payload definitions are in `pkg/models/comms.go` and `pkg/game/types.go`; command handling is in `pkg/game/player.go`.

Typical flow: register/login -> create/discover lobby -> reserve admission/upgrade/join -> administrator updates settings/starts round -> select Street View location outside locks -> commit live generation -> timed guesses/broadcasts -> once-only round/game results. One detached bounded completed-game DB snapshot is written outside game/transport locks with a 5s context; delivery remains best-effort without durable retry. The last completed summary is retained until a successful replay start or lobby cleanup. Transport loss and `leave` keep a returnable seat with the game clock running. The `disconnect` command removes membership and closes its exact owning socket. Other commands are `setReady`, `start`, `submit_location`, `loc_to_cc`, `update_lobby_settings`, and `use_powerup`.

Command failures use one safe `ERR` / `REQUEST_ERROR` envelope with `{command, code, message}`. Successful TYPE names/base fields are retained, with additive authoritative phase/round/recovery state documented in [lobby lifecycle](lobby-lifecycle.md). Pending admissions reject start/settings with curated `LOBBY_BUSY`; membership cancellation during selection returns `START_CANCELLED` to a still-owning initiator. See the [socket contract](socket-contract.md) for readiness, ordered events and match archives, and [backend guardrails](backend-guardrails.md) for error codes, input bounds and token/session compatibility. Failed/cancelled selection leaves rounds/timers/player results, including a completed replay summary, unchanged.

## Configuration and verification boundaries

[Region scoring](region-scoring.md) adds an immutable validated country catalog,
geographical continent definitions and region-adaptive automatic scoring.
`countryList` stays `string[]`; new `countryCatalog` publishes center/span and
scoring constants for exact local previews. Geography derives from playable
rings, separately from existing sampling weights. `LobbyConf.scoringMode`
defaults to auto for new lobbies; explicit manual retains the legacy factor
formula. `scoreScaleKm` is server-derived and carried in detached snapshots.

PostgreSQL uses `DB_HOST`, `DB_USER`, `DB_PASSWORD`, and `DB_NAME`; current code fixes port 5432 and disables database TLS. `LISTEN_ADDR` controls HTTP binding; the local launcher requires `127.0.0.1:8080`. `GMAPS_API` is used for Street View metadata, `JWT_KEY` for token signing, and `LOG_LEVEL` for logging. Never copy real values into documentation or reports.

`make dev` runs the Go server with local-only `-dev` configuration; `make dev-maps` explicitly permits the configured backend Maps key. Go handles dotenv and validation, Make orchestrates build/Compose, and the standalone Go setup command creates private credentials without replacing existing files. No Python is required. Missing/short signing keys are rejected before DB side effects. Purpose validation and sanitized logging do not constitute a general security/concurrency audit.

`make check` compiles server/helper entry points but executes none: no DB, `.env`, Docker, Google key, or app runtime is required. It is not proof of authentication safety, panic resistance, race freedom, or successful multiplayer gameplay. See the [roadmap](roadmap.md) for unresolved risks and the [decisions](decisions.md) for design constraints.
