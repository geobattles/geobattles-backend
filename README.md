# GeoBattles backend

GeoBattles is an open-source geography-guessing game with real-time multiplayer, Street View panoramas and limited-life Battle Royale rounds. This repository contains the Go API, WebSocket game server and PostgreSQL persistence. The [Nuxt frontend](https://github.com/geobattles/geobattles-frontend) is maintained separately.

## Contributing

Use the Go version in [.go-version](.go-version), Git, Bash and Make. No database, Google key or Python runtime is needed for static checks:

```sh
go mod download
make check
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for the supported native and devcontainer workflows. Application unit tests remain opt-in; the frontend repository contains separate opt-in keyless socket integration and browser checks. Static checks do not start the application.

## Local development

Docker Compose v2+ is needed for the isolated local database:

```sh
make setup
make db-up
make dev
```

`make setup` generates private local credentials without printing or overwriting existing configuration. `make dev` validates the local database/listener and withholds the backend Google key. Use `make dev-maps` only for approved manual play or budgeted gameplay checks. Preserve existing `.env` files; never reuse production credentials or point development at production.

Initialize the frontend separately using its contributor instructions. For a trusted same-network preview, see [local LAN deployment](docs/local-lan-deployment.md). That HTTP-only stack is not a production template.

## Docker and production releases

The deployment model remains a Go backend on port **8080**, a Nuxt SSR frontend on port **3000**, PostgreSQL and a reverse proxy. Both app images run non-root; backend assets are bundled under `/app/assets`. Build with an explicit nonsecret version:

```sh
docker build --build-arg VERSION="$(git rev-parse HEAD)" -t geobattles-backend:local-check .
```

Building does not start the server. Starting it validates signing configuration, connects to PostgreSQL and runs `AutoMigrate`.

**Follow the [production release checklist](docs/production-release.md) before deployment.** Release the matching frontend/backend pair, use HTTPS/WSS, keep application/database ports private and preserve a matching rollback pair. Keep one backend replica: active lobbies and timers are in memory and do not survive restart. The current DB connector uses port 5432 and disables TLS; incompatible managed databases require a reviewed connector change. Do not treat a floating `latest` tag as a reviewed release or recreate production data during upgrades.

Branch and pull-request CI only validates. Image publication is a separate manual workflow restricted to `master`, requiring maintainer approval. Checks do not authorize deployment or publication.

## Google configuration

Use separate browser and backend keys. Restrict the frontend key to required browser APIs and website referrers; restrict the backend key to Street View Static API and the server's outbound IP where supported. Browser keys are public configuration, not server secrets. Set API quotas and billing alerts; alerts alone are not a spending cap. Builds and keyless checks need no Google requests.

## Documentation and security

Start with the [documentation index](docs/README.md).

- [Architecture](docs/architecture.md), [decisions](docs/decisions.md) and [roadmap](docs/roadmap.md)
- [Local development](docs/local-development.md) and [database startup](docs/database-startup.md)
- [Socket contract](docs/socket-contract.md), [lobby lifecycle](docs/lobby-lifecycle.md) and [region scoring](docs/region-scoring.md)
- [Security reporting](SECURITY.md)

GeoBattles is licensed under [GPLv3](LICENSE).
