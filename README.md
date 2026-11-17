# GeoBattles

GeoBattles is an open-source interactive geolocation guessing game where players compete to identify locations from Google Street View panoramas.

## 🌍 About the Game

GeoBattles features a **Battle Royale** mode where players can play solo or compete against others to guess locations, with limited lives.

The game offers real-time multiplayer, live statistics and interactive maps, with world, continent and country selection.

This repository contains the **Go backend**: the HTTP API, WebSocket game server and PostgreSQL persistence. The [Nuxt frontend](https://github.com/geobattles/geobattles-frontend) is maintained separately.

## 🚀 Getting Started

### Production Setup

Run the frontend, backend and PostgreSQL with Docker Compose, behind an HTTPS reverse proxy. The example uses the prebuilt `:latest` images for a simple setup. For repeatable production updates and rollback, you can replace `:latest` with compatible version tags or image digests. Images can also be built from source as described below.

Before deploying, follow the [production release checklist](docs/production-release.md) for configuration, database backup, verification and rollback. The example below uses a private same-host database network, matching the current connector's fixed PostgreSQL port 5432 and disabled database TLS.

<details>
<summary>📝 Google API Setup (required for gameplay)</summary>

Open the [Google Cloud console](https://console.cloud.google.com), enable the required APIs and create separate keys:

- **Frontend:** Maps JavaScript API, restricted to your website's referrers.
- **Backend:** Street View Static API, restricted to the server's outbound IP where supported.

> [!WARNING]
> Restrict each key to its required API and configure quotas and billing alerts. Browser keys are public configuration, not server secrets; billing alerts alone are not a spending cap. Builds and static checks do not need Google requests.

</details>

1. Create a project folder, for example `geobattles`.
2. Prepare a private `.env` file in that folder with these Compose inputs. Never commit it or share its values:

   | Variable                   | Value                                                                                                              |
   | -------------------------- | ------------------------------------------------------------------------------------------------------------------ |
   | `GMAPS_API`                | Restricted backend Street View key                                                                                 |
   | `NUXT_PUBLIC_GMAPS_API`    | Restricted browser Maps key                                                                                        |
   | `NUXT_PUBLIC_BACKEND_HOST` | Public HTTPS API origin, for example `https://api.example.com`                                                     |
   | `DB_PASSWORD`              | Random database password; use 64 lowercase hexadecimal characters to avoid the current connector's DSN limitations |
   | `JWT_KEY`                  | Strong, stable signing secret of at least 32 characters                                                            |

3. Create `docker-compose.yaml`:

   ```yaml
   name: geobattles

   services:
     backend:
       image: ghcr.io/geobattles/geobattles-backend:latest
       restart: unless-stopped
       environment:
         GMAPS_API: ${GMAPS_API:?Set the backend Street View key}
         JWT_KEY: ${JWT_KEY:?Set a strong stable signing key}
         DB_HOST: database
         DB_USER: geobattles
         DB_NAME: db
         DB_PASSWORD: ${DB_PASSWORD:?Set the database password}
         LISTEN_ADDR: 0.0.0.0:8080
         LOG_LEVEL: INFO
       depends_on:
         database:
           condition: service_healthy
       healthcheck:
         test:
           [
             "CMD",
             "wget",
             "-q",
             "-O",
             "/dev/null",
             "http://127.0.0.1:8080/readyz",
           ]
         interval: 10s
         timeout: 3s
         retries: 10
         start_period: 20s
       networks: [web, database]

     frontend:
       image: ghcr.io/geobattles/geobattles-frontend:latest
       restart: unless-stopped
       environment:
         NUXT_PUBLIC_GMAPS_API: ${NUXT_PUBLIC_GMAPS_API:?Set the browser Maps key}
         NUXT_PUBLIC_BACKEND_HOST: ${NUXT_PUBLIC_BACKEND_HOST:?Set the public HTTPS API origin}
         NUXT_BACKEND_SERVER_HOST: http://backend:8080
       depends_on:
         backend:
           condition: service_healthy
       networks: [web]

     database:
       image: postgres:17-alpine@sha256:d3244e8ce3f165ee3c93d9c4a4b1d0fe48b3a89e2287b717fdcbbde92d405065
       restart: unless-stopped
       environment:
         POSTGRES_USER: geobattles
         POSTGRES_PASSWORD: ${DB_PASSWORD:?Set the database password}
         POSTGRES_DB: db
       volumes:
         - ./.db:/var/lib/postgresql/data
       healthcheck:
         test:
           ["CMD-SHELL", 'pg_isready -U "$$POSTGRES_USER" -d "$$POSTGRES_DB"']
         interval: 3s
         timeout: 3s
         retries: 20
       networks: [database]

   networks:
     web:
       name: geobattles_web
     database:
       internal: true
   ```

4. Validate without printing resolved secrets, then start the stack when ready:

   ```sh
   docker compose --env-file .env config --quiet
   docker compose --env-file .env up -d --wait
   ```

Only the reverse proxy should publish ports. Attach it to the `geobattles_web` network so it can reach `backend:8080` and `frontend:3000`; the application and database ports above are not published to the host.

Starting the backend validates its signing configuration and runs database migrations. The example retains the original `.db` data directory and SQL database name `db`; preserve existing data and secrets rather than replacing or reinitializing them during upgrades. Run **one backend replica**: active lobbies and timers live in memory and disappear on restart, so drain occupied rooms before replacing it.

### SSL Configuration

Terminate HTTPS/WSS at your reverse proxy. For example, a [SWAG](https://github.com/linuxserver/docker-swag) container attached to `geobattles_web` can use the following server blocks. Replace the example domains and configure certificates in the proxy; its SSL include path is SWAG-specific.

<details>
<summary>nginx server blocks</summary>

```nginx
# Backend API and WebSockets
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name api.example.com;

    include /config/nginx/ssl.conf;

    location / {
        proxy_pass http://backend:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 3600s;
    }
}

# Nuxt frontend
server {
    listen 443 ssl;
    listen [::]:443 ssl;
    server_name game.example.com;

    include /config/nginx/ssl.conf;

    location / {
        proxy_pass http://frontend:3000;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

</details>

Preserve the application's raw-JWT `Authorization` header and WebSocket subprotocols; do not replace them with proxy Basic Auth. A same-origin `/api` gateway is another option; see the [release guide](docs/production-release.md#2-supply-runtime-configuration-privately) for its routing and SSR requirements.

## 🛠️ Development setup

### Requirements

- Go **1.27.4**, pinned in [.go-version](.go-version), plus Git, Bash and Make.
- Docker Compose v2+ for the isolated local PostgreSQL database or container stack.
- Restricted Google keys for gameplay; no key, database or Python runtime is needed for static checks.

### Setup Steps

1. Clone the backend:

   ```sh
   git clone https://github.com/geobattles/geobattles-backend geo-backend
   cd geo-backend
   ```

2. Choose your development method.

#### With Go installed locally

```sh
go mod download
make check
make setup
make db-up
make dev
```

`make check` checks formatting, modules, vet and compilation without starting the application. `make setup` creates private local credentials only when `.env` is absent; it never overwrites your configuration. `make dev` validates the isolated development database and listens on `127.0.0.1:8080`, with the backend Google key withheld.

For Maps-enabled manual play, configure the restricted keys, stop the keyless backend and use `make dev-maps`. Starting rounds can spend Google quota. Set up the frontend separately following its contributor guide; see [local development](docs/local-development.md) for both repositories and safe shutdown commands.

#### With Docker

Build the backend image from this repository without starting services:

```sh
docker build --build-arg VERSION="$(git rev-parse HEAD)" -t geobattles-backend:local .
```

For the complete local frontend/backend/database stack, keep sibling clones named `geo-backend` and `geo-frontend`, and prepare their private `.env` files separately. From the backend repository:

```sh
env -u DB_PASSWORD docker compose --env-file .env -f deploy/compose.lan.yml config --quiet
env -u DB_PASSWORD docker compose --env-file .env -f deploy/compose.lan.yml up -d --build --wait
```

The versioned files are [deploy/compose.lan.yml](deploy/compose.lan.yml) and [deploy/Caddyfile.lan](deploy/Caddyfile.lan). Open `http://localhost:8088`, or the host's LAN address from another device. This is an **HTTP development preview, not a production template**; see [LAN deployment](docs/local-lan-deployment.md) for configuration, versioning and restart precautions.

> [!NOTE]
> See [example.env](example.env) for backend configuration names and placeholders. Preserve existing `.env` files and never use production credentials or databases for development.

## Contributing and documentation

- [Contributing](CONTRIBUTING.md): supported toolchains, checks and devcontainer workflow.
- [Documentation index](docs/README.md): architecture, API/socket contracts, region scoring and operational guides.
- [Runtime diagnostics](docs/runtime-diagnostics.md): `/healthz`, `/readyz` and `/version`.
- [Security reporting](SECURITY.md).

GeoBattles is licensed under [GPLv3](LICENSE).
