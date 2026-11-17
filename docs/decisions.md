# Design decisions

## Keep the existing service boundaries

The Go backend, PostgreSQL/GORM persistence and separate Nuxt frontend remain independent components. Prefer focused changes to framework or persistence rewrites. Preserve wire/configuration compatibility unless both clients and server are changed together.

## Separate verification, runtime and publication

Go **1.27.4** in `.go-version` pins the verification toolchain; `go.mod` separately declares the language minimum. `make check` covers shell syntax, module integrity, formatting, vet and compilation without starting services. Application unit tests, new frameworks and race-test jobs are opt-in; frontend integration/browser checks are separate and keyless by default.

Pull requests and branch pushes validate with read-only permissions. Image publication is manual and restricted to `master`. Passing checks does not authorize deployment, production database work or paid provider calls.

## Keep configuration in Go and orchestration in Make

Load and validate configuration, then configure JWT signing before database access. Go creates private local credentials without replacing existing files; Make coordinates build/run/Compose. Development mode restricts the database/listener and suppresses the backend Maps key unless explicitly enabled. Never evaluate `.env` as shell code.

## Own live games in one process

A private registry holds lobby pointers; each lobby mutex owns its mutable state. Detached snapshots and immutable queued events keep network writes outside game locks. Recovery is bounded by membership/grace lifetime, not durable across restarts. Completed-game persistence is one bounded best-effort attempt, not exactly-once delivery. See [lobby lifecycle](lobby-lifecycle.md).

## Preserve explicit protocol and provider boundaries

HTTP authorization uses raw JWTs; WebSockets offer `json` and an access token. Tokens require signed purpose, expiry and subject. Command failures use curated `REQUEST_ERROR` messages, never raw provider/driver errors. Round selection is bounded and occurs before committing game state. See [guardrails](backend-guardrails.md) and the [socket contract](socket-contract.md).

## Derive automatic scoring from geography

New lobbies use region-adaptive automatic scoring. Manual scoring requires an explicit policy and retains its existing factor formula. The server normalizes countries and derives the effective scale; clients cannot supply an authoritative scale. Settings changes preserve readiness. See [region scoring](region-scoring.md).
