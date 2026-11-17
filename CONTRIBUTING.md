# Contributing to the backend

## Start a focused change

Create a feature branch from public `master`, for example `feat/lobby-improvement` or `fix/socket-recovery`, and submit a pull request against `master`. Keep unrelated changes separate and preserve existing work.

Read the [architecture](docs/architecture.md) and relevant [protocol and operations documentation](docs/README.md). Coordinate frontend/backend changes to routes, payloads, raw-JWT authorization, WebSocket subprotocols, configuration or database behavior.

## Static development loop

Install Git, Bash, Make and Go **1.27.4**, pinned in [.go-version](.go-version). Docker Compose v2+ is needed only for local runtime work; static checks need no database, API key, `.env` or Python runtime.

```sh
go version                    # must report go1.27.4
go mod download               # populate dependency cache
make check
make build                    # optional: .local/bin/geobattles-backend
```

Without Make, use `./scripts/check.sh`. Checks cover shell syntax, module integrity, Go formatting, `go vet ./...` and `go build ./...`, including root, `cmd/` and `pkg/` sources. They do not execute application entry points. Cold dependency downloads may need network access; checks use the pinned local toolchain and read-only module mode. Do not run `go get` or `go mod tidy` as unrelated cleanup.

**Application unit tests and new test frameworks remain opt-in.** The frontend repository provides separate socket integration and browser E2E tooling. Run it only with approved isolated local services, keyless by default; Maps-backed checks need explicit spending scope and a budget. This is not blanket permission for application unit tests or race-test jobs. Static success is not behavioral, concurrency, integration or security acceptance.

## Optional devcontainer

The checked-in VS Code devcontainer pins Go 1.27.4 and includes Bash, Git and Make. It runs non-root and downloads modules after creation, without starting services or forwarding ports. Run `make check` in its terminal.

For runtime work, use the host-side Make/Compose workflow below. The container deliberately has no Docker-socket mount or automatic PostgreSQL service. Do not weaken development database guards to work around container networking.

## Local runtime

From the repository root, when local services are wanted:

```sh
make setup                    # creates private .env only if absent
make db-up                    # isolated PostgreSQL; preserves data
make dev                      # loopback server, backend Maps key withheld
```

See [local development](docs/local-development.md) for configuration, frontend setup, shutdown and separately approved `make dev-maps` use. Setup does not overwrite credentials. Startup validates configuration and JWT signing **before** database connection and migration. Never point development tools at production or delete database volumes as routine cleanup.

## Pull request checklist

- Explain purpose, compatibility and rollback implications; include exact verification commands and outcomes, failures and unverified behavior.
- Review the complete diff, including untracked files. Stage explicit paths and inspect `git diff --cached --check` and the staged file list before committing.
- Keep credentials, local backups, generated output, database files and runtime/browser logs out of the change. Never use blanket `git clean -fdx` to resolve a dirty checkout.
- Update durable contracts/runbooks when behavior changes. Keep implementation journals and one-off execution evidence outside public repository history.
- Report vulnerabilities privately through [SECURITY.md](SECURITY.md).

Branch and pull-request CI validates without publishing. Image publication is a separate manual workflow restricted to `master` and requires maintainer approval. Deployment and paid provider checks are separate operations; follow the [production release gates](docs/production-release.md). Local checks do not require a push or workflow dispatch.
