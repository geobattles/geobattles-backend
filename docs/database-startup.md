# Database startup

## Fail-closed initialization

`main.go` loads configuration, configures logging and validates JWT signing before database access. `db.ConnectDB() error` opens a candidate GORM handle with SQL logging set to `Silent`, then checks `AutoMigrate` for these models in order:

1. `models.User`
2. `models.Game`
3. `models.Round`
4. `models.Result`
5. `models.RefreshToken`

Only successful migration publishes the candidate as `db.DB`. Connection or migration failure closes the candidate's underlying SQL pool when available without replacing an existing global handle. Startup returns one of these fixed messages, never a wrapped driver error, DSN, SQL or credential:

- `Failed to connect to the database; check local database configuration and availability`
- `Failed to migrate the database schema; check local database permissions and schema compatibility`

The server logs the safe error and exits with status **1** before asset initialization, route registration or HTTP listening. `ConnectDB` is a startup operation, not a runtime reconnect API. Successful startup logs `Connected to DB`; migration detail is at debug level.

**Failure is not rollback.** There is no encompassing migration transaction. DDL completed before a later failure may remain committed; closing a pool does not undo it. Inspect schema state before retrying. Do not infer successful migration from a successful database connection alone.

## Configuration and operation

The connector reads `DB_HOST`, `DB_USER`, `DB_PASSWORD` and `DB_NAME`, fixes port **5432** and uses `sslmode=disable`. It does not support `DATABASE_URL` or configurable TLS/port settings. Its unquoted DSN also requires care with special characters. Review the connector before selecting an incompatible managed database; do not weaken provider security to accommodate it.

Use only the isolated development database for local runtime work. Builds and `make check` do not connect or migrate. Production requires a tested backup/restore plan and migration rehearsal against an authorized isolated copy; see [production release](production-release.md).

## Failure-path verification

When separately authorized, a migration-denial check should use a disposable login and empty administrator-owned schema in an isolated development database. Verify connection permission, absence of schema `CREATE`, and an exclusive search path with no `public` fallback before running the candidate on an unused loopback port. Use disposable private configuration and a blank Maps key; do not modify normal credentials or services.

Require exit **1**, the fixed migration error, no readiness message, no credential output and no HTTP listener. A refused request after exit alone cannot exclude a transient listener. Inspect any partial schema changes and remove only the exact temporary role/schema and their grants. This verifies startup failure handling, not rollback or multiplayer persistence.
