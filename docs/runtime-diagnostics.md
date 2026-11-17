# Runtime diagnostics

These public, credential-free endpoints distinguish a responding backend, basic
readiness, and the build being served. They do not create accounts/lobbies, change
schema, inspect sessions, or call Google. Existing startup and gameplay contracts
are unchanged.

## HTTP contract

| Request | Status | JSON body | Meaning |
| --- | --- | --- | --- |
| `GET /healthz` | `200` | `{"status":"ok"}` | This process can answer HTTP. No dependency probe. |
| `GET /readyz` | `200` | `{"status":"ready"}` | Startup gates passed and the current PostgreSQL pool ping succeeded. |
| `GET /readyz` | `503` | `{"status":"unavailable"}` | Startup assets/pool are unavailable, or the database ping failed/timed out. |
| `GET /version` | `200` | `{"version":"dev"}` | Public build identifier; `dev` is the build default. |

All three routes set `Cache-Control: no-store`,
`Content-Type: application/json; charset=utf-8`, and
`X-Content-Type-Options: nosniff`, including failure and preflight responses.
`HEAD` performs the same check as `GET` but sends no body. `OPTIONS` returns `204`
without a body or database ping. Other methods return `405`,
`Allow: GET, HEAD, OPTIONS`, and `{"status":"method_not_allowed"}`.

Browser access allows any origin without credentials. Use a simple `GET` without
an authorization/custom header; no account/token is required. A recovery UI may
fetch with `cache: 'no-store'`. Do not treat a network error as a confirmed `503`:
it can also mean the client is offline or the backend/proxy is unreachable.

## Readiness boundary

Routes are installed only after the existing synchronous database/migration and
asset initializers have run. The startup gate captures successful pool access,
nonempty country and full-polygon catalogs, and success from the reverse-data
initializer. A database connection/migration failure still exits before an HTTP
listener exists; diagnostics do not replace or change that startup lifecycle.

Each `GET`/`HEAD /readyz` that passes the startup gate calls the existing
`database/sql.DB.PingContext` with a **one-second timeout**, derived from the
request context so client cancellation can end it sooner. The handler does not
open a separate pool, perform application SQL, retry, call a provider, or include
raw errors in responses/logs. A later database outage makes readiness fail without
making liveness fail, and a recovered database can become ready on the next probe.
A failed startup asset gate remains unavailable until a successful restart.

Readiness uses startup status and minimal catalog checks, not a fresh validation
of every geographical asset on each request.
Readiness also does not prove Maps authorization/quota/coverage, writable tables,
WebSocket admission/recovery, or successful multiplayer gameplay. Use readiness
for availability reporting/routing, not automatic restart on every database outage.
These endpoints do not themselves install a restart policy.

## Build identity and privacy

The version response has exactly one allowlisted field: `version`. Its value is
captured before serving from `main.version`, defaulting to `dev`. It is not read
from runtime environment variables and does not change during the process lifetime.
No build-settings/module dump, hostname, paths, database DSN, environment values,
credentials, player/session details, or provider error is exposed.

The Docker builder accepts `ARG VERSION=dev` and links it with
`-ldflags="-s -w -X main.version=${VERSION}"`. Pass a unique **public** release or
build identifier through Compose `build.args.VERSION` (or Docker `--build-arg
VERSION=local-build-id`). Prefer a short identifier using letters, digits, `.`,
`_`, `+`, and `-`; do not put spaces, secrets, or private infrastructure details in
it. A runtime Compose `environment.VERSION` entry does not set build identity.
Rebuilding/recreating the container is necessary to change the served identifier.

A native build can set the same field, without starting the app:

```sh
# Run from geo-backend with Go 1.27.4; output stays in the ignored local directory.
mkdir -p .local/bin
GOTOOLCHAIN=local GOWORK=off go build -mod=readonly -trimpath \
  -ldflags='-X main.version=local-build-id' \
  -o .local/bin/geobattles-backend .
```

Existing `make build`/`make dev` builds keep the `dev` default. A unique identifier
is supplied by the caller, not inferred from Git: a revision alone does not
identify intentional uncommitted changes. Compare `/version` against the exact
expected build value when diagnosing stale local containers/proxy routing. The
identifier is diagnostic metadata, not cryptographic proof of deployment.

## Verification scope

Use `make check` with the pinned Go 1.27.4 for formatting, module integrity, vet,
and compilation; it does not start the app or require PostgreSQL/Google. Runtime
acceptance belongs to the separately approved service owner: after an approved
restart, read the three routes and confirm the exact expected version and headers.
A runtime `503`/timeout check needs a separately controlled database outage; do not
disrupt an active game or start another database merely to demonstrate it.
