# Backend documentation

## Develop and contribute

- [Contributor workflow](../CONTRIBUTING.md): feature branches, toolchain and static checks
- [Local development](local-development.md): private configuration and isolated PostgreSQL
- [Architecture](architecture.md) and [design decisions](decisions.md)
- [Improvement areas](roadmap.md)

## API and game contracts

- [Lobby lifecycle and payloads](lobby-lifecycle.md): snapshots, ownership, recovery and persistence
- [Socket contract](socket-contract.md): readiness, ordering, heartbeat, match identity and archives
- [Powerups](powerups.md): next-round choices, start barrier, idempotency and Duel settlement
- [Backend guardrails](backend-guardrails.md): validation, provider failures and token compatibility
- [Region scoring](region-scoring.md): country catalog and automatic/manual curves
- [Location selection](location-selection.md): search areas, returned-point bounds and country coverage

## Operate safely

- [Database startup](database-startup.md): migration failure and connector limits
- [Runtime diagnostics](runtime-diagnostics.md): health, readiness and build identity
- [LAN preview](local-lan-deployment.md): trusted-network development stack
- [Production release](production-release.md): compatibility, acceptance and rollback gates
- [Security policy](../SECURITY.md)
