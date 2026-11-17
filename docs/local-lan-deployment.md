# Versioned local LAN preview

Use `deploy/compose.lan.yml` and `deploy/Caddyfile.lan` for a trusted-network development preview. Keep sibling clones named `geo-backend` and `geo-frontend` under one workspace. This HTTP stack is not a production deployment template.

## Before starting or replacing services

- Use private development `.env` files. `make setup` creates backend credentials when absent; it never overwrites them. Configure the frontend separately from its `.env.example`. Never commit either private file.
- This stack uses the isolated `geobattles_dev` database/user and an internal database network. Backend/database ports are not published; only the gateway listens on LAN port 8088 (or `GEOBATTLES_LAN_PORT`). Do not supply production configuration.
- Startup migrates the development database. Maps-enabled manual play and automated provider checks require their own explicit spending scope.
- Inspect occupied lobbies before restarting the backend. All in-memory lobbies disappear on restart, including waiting/finished rooms with connected players. Obtain reset permission where needed. Never remove the persistent database volume as cleanup.

From the **backend repository**:

```sh
# Validation does not start services; --quiet avoids printing resolved secrets.
env -u DB_PASSWORD docker compose --env-file .env -f deploy/compose.lan.yml config --quiet

# Only under the relevant local-start/restart approval:
env -u DB_PASSWORD docker compose --env-file .env -f deploy/compose.lan.yml up -d --build --wait
```

Use `GEOBATTLES_BUILD_VERSION` for a shared nonsecret build identifier when preparing both images. Compose uses project `geobattles-lan` and its dedicated volume; its SQL database/user are named `geobattles_dev`, not `geobattles-lan`. Compare running image IDs and `/api/version` with the build manifest; read back `/api/healthz`, `/api/readyz`, frontend hydration and actual occupied-lobby state after an approved change. Public HTTP200 alone is not gameplay acceptance.

The gateway strips `/api` for HTTP and WebSocket traffic and preserves native upgrade behavior. Access logging is disabled to avoid retaining private query/subprotocol data. `http://localhost:8088` is the local URL; phones use the host's approved LAN address. This is an HTTP development preview, not public HTTPS production hosting.

Production, image publication and remote writes remain separately approval-gated.
