# Local development

## Requirements

- Go version in `.go-version`, Make, Bash, Git and Docker Compose v2+.
- Frontend: nvm and its repository `.nvmrc`.
- No Python, pip environment, cloud database or new backend dependency is required.

Install Go from https://go.dev/dl/ and verify the official SHA256 before extracting a user-local toolchain. In WSL, start Docker Desktop with integration enabled for this distribution; verify `docker version` before database commands.

## Initialize and run

From the backend repository root:

```bash
go mod download
make setup
make db-up
make check
make dev
```

`make setup` runs a small Go command that reads `example.env`, generates random DB/JWT credentials, and exclusively creates `.env` with mode 0600. Existing configuration is retained, never overwritten or rotated. It does not initialize the frontend configuration or touch a database. Never post `.env` or resolved Compose credential output to chat/issues/logs.

Compose creates project `geobattles-dev`, a dedicated volume and digest-pinned PostgreSQL 17. Database/user are `geobattles_dev`; only `127.0.0.1:5432` is published. There is no automatic container restart policy. Make's database commands remove ambient `DB_PASSWORD` before Compose reads `--env-file`; local DB/user are fixed in YAML. If port 5432 is occupied, resolve it deliberately rather than killing unrelated services.

`make dev` builds and runs the server with `-dev`. Go reads `.env` without executing it as shell code, validates local DB host/user/name, a 64-character lowercase hexadecimal password, JWT key of at least 32 characters, INFO logging and `LISTEN_ADDR=127.0.0.1:8080`, then applies only application configuration keys. File values take precedence for dev mode. It blanks the backend Google key unless explicitly opted in.

Configuration/JWT initialization happens in explicit startup, before database migration and asset loading—not in package init. A missing/short signing key rejects startup. Normal startup retains process-environment precedence over optional `.env` and the legacy listener default; use `-dev` for local isolation. Other authentication/gameplay risks remain separate work.

## Maps-enabled manual play

For a requested manual gameplay or Android PWA play check, use both configured, restricted Maps keys. For the standard backend on port 8080, run:

```bash
make dev-maps
```

This uses `-dev -allow-paid-maps` and enables the backend Street View key. Do not run it alongside another backend on 8080. Starting rounds can spend Google quota. Permission for manual play does not authorize automated rounds or provider probes.

For the frontend, preserve an existing `.env`. On a fresh checkout copy its `.env.example` privately and configure the browser key plus `NUXT_PUBLIC_BACKEND_HOST=http://127.0.0.1:8080`. Then:

```bash
cd ../geo-frontend
nvm use
npm ci
npm run dev
```

Open http://127.0.0.1:3000/. Google browser-key referrer restrictions must allow the chosen local URL. Use separate API-restricted browser Maps JavaScript and backend Street View keys, with quotas. Browser keys are necessarily visible to browsers; JWT/DB secrets must never be placed in NUXT_PUBLIC_* values. Restart the frontend dev server after editing its `.env`.

## Manual checks and shutdown

These endpoints do not start rounds:

```bash
curl --fail http://127.0.0.1:8080/countryList
curl --fail http://127.0.0.1:8080/lobby
make db-status
```

Ctrl+C stops the foreground app. `make db-stop` stops PostgreSQL while preserving data. `make db-down` removes its container/network but retains the named volume. Never delete the volume without deliberate approval. Editing `.env` does not rotate an existing database's stored password.

Startup still auto-migrates the local database; compilation alone does not. Application unit tests and new test frameworks require explicit approval; separate frontend integration/browser tooling does not grant blanket permission. Use builds, formatting/static/type checks and scoped manual verification, reporting limitations honestly. Publication, production operations and spending beyond approved local use remain approval-gated.
