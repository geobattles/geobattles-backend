## Purpose

<!-- What changes, why, and the explicit acceptance criteria? -->

## Scope and compatibility

<!-- Note routes/payloads, database behavior, configuration, dependencies, and rollback implications; write "unchanged" where applicable. -->

## Verification evidence

<!-- Paste exact commands and outcomes, not only "works". Record failures/blockers and anything not exercised. -->

- [ ] Used Go from `.go-version` and ran `make check` (or `./scripts/check.sh`).
- [ ] No application unit tests, new test framework or race-test job added/run without explicit approval; separate integration/browser checks stayed within approved local and provider scope.
- [ ] Static success is not described as behavioral, integration, concurrency, or security coverage.

## Safety and documentation

- [ ] No secrets, authorization logs, local binaries, database files or internal work journals included.
- [ ] No unapproved database/runtime actions, paid API calls, deployment, or image publication.
- [ ] Architecture/decisions/roadmap updated where relevant; no unrelated changes hidden in this PR.

## Remaining risks

<!-- Known source findings, unverified behavior, and required follow-up/approval. -->
