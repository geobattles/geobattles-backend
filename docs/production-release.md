# Production release checklist

This is a release gate, not a production Compose configuration or evidence that a deployment has run. Inspect the actual hosting, proxy, PostgreSQL version/schema and secrets management before adapting it. The checked-in LAN stack intentionally uses HTTP and development credentials; never expose it as an Internet production recipe.

## 1. Freeze a compatible pair

- Review and commit both repositories, run their documented checks, and record both commit IDs and final image digests. Deploy the frontend requiring `/countryCatalog` with its matching backend.
- Image publication is manual/master-only; dispatch still needs owner approval. The backend passes the complete source SHA as Docker `VERSION`; the frontend embeds its source version. Component SHAs differ: record the release pair rather than assuming they are equal.
- Review unresolved dependency advisories and bundled-asset provenance. Frontend findings are tracked in its `docs/dependency-security.md`; a successful build is not a security clearance.
- Maintain secret scanning/push protection, private vulnerability reporting and master protection with required CI checks. Local YAML does not enforce remote settings. Never publish credentials, private backups, browser artifacts or unrelated archive tags.

## 2. Supply runtime configuration privately

Backend settings are `DB_HOST`, `DB_USER`, `DB_NAME`, `DB_PASSWORD`, `JWT_KEY`, `GMAPS_API`, `LISTEN_ADDR` and `LOG_LEVEL`. Preserve a strong stable signing key; missing/short keys fail before DB access. Do not use development setup commands to generate or rotate production configuration. Legacy tokens require re-login; plan for guest-session effects.

The current connector hardcodes port 5432 and `sslmode=disable`, and interpolates an unquoted DSN. It does not read DATABASE_URL or configurable TLS/port options. Use an appropriate private same-host PostgreSQL network, or change/review the connector before using an incompatible managed service. DSN-special characters in configuration also need correct handling; do not weaken provider security to work around connector limitations.

Frontend settings:

- `NUXT_PUBLIC_BACKEND_HOST`: browser-reachable HTTPS API origin, or `/api` behind a same-origin gateway.
- `NUXT_BACKEND_SERVER_HOST`: optional private SSR API origin, such as `http://backend:8080` on an isolated container network. It falls back to the public host; supply it for a relative public prefix.
- `NUXT_PUBLIC_GMAPS_API`: restricted browser key; every `NUXT_PUBLIC_*` value is public.

A same-origin gateway must strip `/api` consistently and forward WebSocket upgrades/subprotocols. Never expose container DNS to browsers, use mixed HTTP content on an HTTPS page, or replace the application's raw-JWT Authorization header with edge Basic Auth. Host the app at a reviewed origin/root; PWA subpath hosting needs additional scope/routing work.

## 3. Protect ingress and provider usage

- Terminate HTTPS/WSS at a configured proxy; do not expose app/DB ports directly. Review allowed origins, request/body/connection limits, guest/auth abuse controls, logging and alerts.
- Restrict browser/server Maps keys separately by API and referrer/outbound-IP as appropriate. Establish quotas and billing alerts; billing alerts are not hard spending caps. Never put keys into image build arguments or publish resolved Compose output.
- Keep one long-running backend replica. In-memory game ownership does not support interchangeable replicas, overlapping rolling updates or scale-to-zero.

## 4. Rehearse database and application acceptance

Compare model definitions, ORM/driver dependencies and the deployed schema for every release. Region/scoring settings are in-memory lobby state, not SQL fields; that does not establish compatibility for other changes.

AutoMigrate runs on startup and fails closed. Production schema drift or DDL permissions can block startup. There is no encompassing rollback transaction: partial DDL is not undone by process failure.

Obtain a backup and prove restore to an authorized isolated production-like database before launch. Rehearse startup there; never use production as a smoke-test database, recreate its tables or delete its volume. Verify health/readiness/version, HTTPS/SSR login, catalog, WebSockets and keyless lobby flows. Separately approve/budget a real multiplayer Maps game, reconnect/replay, phone checks and secure-context PWA install/update. HTTP LAN and provider-free checks do not establish this acceptance.

## 5. Cut over and retain rollback

1. Record current image digests and private configuration through the deployment system without exposing values. Keep a matching rollback pair and a separately verified database recovery plan.
2. Announce/drain active games before restarting the backend; waiting/finished rooms may still contain players. A backend restart loses in-memory lobbies.
3. Deploy only the approved immutable frontend/backend pair. Preserve unrelated services, database volume and configuration. Do not use stack/volume deletion as rollback.
4. Read back the running image identities and routed `/healthz`, `/readyz`, `/version` responses (with the gateway prefix if configured), and exercise the intended routes.
5. Restore the known-good image pair if needed within the approved maintenance window. Image rollback does not reverse database changes or recreate lost games.

Production deployment, image publication, public posts and paid provider checks remain separately approved operations.
