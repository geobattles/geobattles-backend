# Automated contributor guidance

Read [CONTRIBUTING.md](CONTRIBUTING.md) and the [documentation index](docs/README.md) before editing. Inspect the worktree and preserve unrelated changes.

- Use focused feature branches from public `master`. Do not commit, push, merge, publish images, deploy, spend provider quota or start/restart services unless requested.
- Preserve routes, JSON fields, raw-JWT HTTP authorization, WebSocket subprotocols, configuration precedence and database behavior unless a coordinated change is approved.
- Use Go **1.27.4** from `.go-version` and `make check` (or `./scripts/check.sh`). These checks cover shell syntax, module integrity, formatting, vet and compilation, not application execution. `make build` writes only ignored `.local/bin/` output.
- Application unit tests, new test frameworks and race-test jobs require explicit approval. Existing frontend socket integration/browser E2E tooling is separate from routine checks; it requires approved local services and is keyless by default. That tooling does not grant blanket permission for application unit tests or paid Maps checks.
- Do not change dependencies or run `go get` / `go mod tidy` incidentally. Report exact checks, actual results and unverified behavior; never present compilation as runtime or security acceptance.
- Never start the application as a build check. Startup connects to PostgreSQL and auto-migrates. Runtime work must use an approved isolated development database, never production.
- Preserve private configuration. Do not print, commit or attach `.env*`, tokens, credentials, raw authorization logs or resolved Compose configuration. `make setup` creates private local credentials only when absent; `make dev` withholds the backend Maps key.
- `make dev-maps`, round starts and the API helper can spend Google quota. Permission for someone's manual play is not permission for automated rounds or API experiments.
- Keep validation non-publishing and read-only. Image publication is manual and restricted to `master`; do not dispatch or push merely to obtain checks.
- Keep documentation about current contracts and supported operations. Do not put agent handoffs, task journals, approval histories or scratch evidence into the repository.
