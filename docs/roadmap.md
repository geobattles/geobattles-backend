# Backend improvement areas

These are design and verification topics, not a release certification or authorization to implement them.

- **Session lifecycle:** assess immediate access-token/session revocation and atomic refresh revoke-and-reissue. Existing WebSockets are not reauthenticated when a token is revoked.
- **Abuse and information boundaries:** review origin policy, global lobby/rate limits and public player/result visibility. Member snapshots intentionally carry the current answer; competitive anti-cheat needs a separate design.
- **Runtime reliability:** exercise active-round recovery, simultaneous guess/deadline/leave finalization, slow peers, replay and successful completed-game persistence against the exact release pair. Compilation and keyless lobby checks are not substitutes.
- **Operations:** review graceful shutdown, controlled migrations, PostgreSQL TLS/DSN configuration and production ingress limits. Rehearse database restore and compatible image rollback before deployment.
- **Dependencies and assets:** assess advisories by exposure and review bundled geography provenance; keep compatible upgrades separate from broad rewrites.
- **Gameplay scope:** Country Battle remains unfinished; do not advertise it as complete without an explicit product decision and acceptance.

Use [CONTRIBUTING.md](../CONTRIBUTING.md) for verification boundaries and the [production release checklist](production-release.md) for deployment gates.
