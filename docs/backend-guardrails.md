# Backend guardrails

## Street View boundary and round start

- Metadata succeeds **only** for HTTP 200, valid JSON, `status: "OK"`, and an explicitly supplied finite latitude/longitude in range. A real `(0, 0)` is valid; missing coordinates or any error must never be interpreted as that location.
- One metadata request has a **2-second timeout**, a **64 KiB body limit**, and no redirects. Request URLs, transport error strings and Google's `error_message` are never logged/returned: they may contain the API key.
- One round-location selection shares **6 seconds, 8 HTTP calls, and 1,024 polygon candidate points**, whichever is exhausted first. Select another area after four metadata attempts. There is no nested/unbounded retry budget.
- Only `ZERO_RESULTS`, network/body-read failures/timeouts, `UNKNOWN_ERROR`, and HTTP 408/500/502/503/504 may retry. Transient faults wait 100 ms between attempts; empty coverage resamples immediately. Auth/permission/billing/quota failures (HTTP 401/403/429, `REQUEST_DENIED`, `OVER_QUERY_LIMIT`, `OVER_DAILY_LIMIT`) fail on the first response. Invalid JSON, incomplete coordinates, unexpected statuses and other HTTP failures also fail without retry.
- `startGame` selects a valid location before changing round number, locations, results, player lives/powerups, active state or timer. Failure sends an error only to the initiating client; the lobby remains available for retry (including a next-round retry with previous results intact). No `START_ROUND` success is broadcast. Generation and ownership checks exclude stale commits; see [lobby lifecycle](lobby-lifecycle.md).
- The manual `cmd/main.go api` helper uses the same single-request validation/timeout, exits on errors other than empty coverage, and retains its separate bounded per-polygon exploration loop. Do not execute it without Google/spending approval.

## Socket error contract

Successful envelopes and payloads are unchanged. Command errors from `PlayerMessageHandler` use:

```json
{
  "status": "ERR",
  "type": "REQUEST_ERROR",
  "payload": {
    "command": "start",
    "code": "LOCATION_PROVIDER_QUOTA",
    "message": "Street View quota is exhausted. Ask the server operator to check quota and billing before retrying."
  }
}
```

`command` is the original command (empty when it cannot be decoded), `code` is a stable machine-readable code, and `message` is curated actionable text. Unknown internal errors become `REQUEST_FAILED`; raw parser/provider/DB details are not interpolated. Clients must dispatch error status/`REQUEST_ERROR`, display `message`, and release the pending action associated with `command` (especially start/guess loading states). Do not automatically retry quota/auth failures.

| Codes | Action / meaning |
| --- | --- |
| `LOCATION_NOT_FOUND` | Coverage/search-attempt limit; retry or change countries. |
| `LOCATION_GENERATION_TIMEOUT`, `LOCATION_PROVIDER_UNAVAILABLE` | Selection deadline/transient provider failure; retry later. |
| `LOCATION_PROVIDER_DENIED`, `LOCATION_PROVIDER_QUOTA` | Operator must check API access, restrictions, billing/quota; no automatic retries. |
| `LOCATION_PROVIDER_UNCONFIGURED` | Operator must configure the backend key; keyless local mode intentionally cannot start Maps rounds. |
| `LOCATION_PROVIDER_INVALID_RESPONSE` | Invalid/non-OK provider response; retry later or contact operator. |
| `LOCATION_DATA_INVALID` | Missing/invalid selection data or point-sampling limit; change countries/contact operator. |
| `INVALID_REQUEST`, `INVALID_COMMAND`, `INVALID_LOCATION` | Fix request JSON/command or provide both valid coordinate components. |
| `INVALID_COUNTRY_CODE`, `INVALID_LOBBY_SETTINGS`, `LOBBY_CAPACITY_TOO_SMALL` | Fix settings; invalid updates do not partially mutate configuration. |
| `NOT_ADMIN`, `GAME_IN_PROGRESS`, `ALREADY_ACTIVE`, `ROUND_STARTING` | Wrong role or action not valid during this state; wait/resume current action. |
| `GAME_NOT_ACTIVE`, `TIMES_UP`, `NO_MORE_ATTEMPTS`, `ALREADY_FINISHED` | Wait for the appropriate round/action. |
| `PLAYER_NOT_IN_LOBBY`, `INVALID_GAME_STATE`, `REQUEST_FAILED` | Reconnect before retrying; contact operator if persistent. |
| `NO_COUNTRY` | Choose a point inside a country. |
| `CANT_USE_LAST_ROUND`, `NOT_AVAILABLE`, `WRONG_TARGET`, `WRONG_TYPE` | Correct powerup timing/availability/target/type. |

Locations must contain both non-null `lat` and `lng`: latitude [-90,90], longitude [-180,180], no NaN/Infinity. Explicit zeros remain valid. Missing location pointers, absent membership, pre-round guesses, result/index gaps, unknown commands/countries and unavailable powerup slots are rejected before relevant dereferences. Player membership is installed before its socket reader starts.

Supplied lobby settings accept mode 1/2, players 1–12, attempts/rounds 1–100, round time 5–3,600 seconds, score factor 1–500 and exactly two powerup switches. Numeric zero/omission keeps its previous default/no-update meaning. Unknown countries are rejected on both HTTP creation (400) and socket update. Capacity/admission includes retained membership and pending reservations; a full lobby rejects admission with HTTP 409 `LOBBY_FULL`. HTTP lobby bodies are capped at 64 KiB and auth bodies at 32 KiB; socket text messages are limited to 16 KiB.

## JWT/session compatibility and logging

- Newly issued JWTs carry signed `token_use: "access"` or `"refresh"`. Both validators require HS256, expiry and nonempty subject; access requires access purpose/no refresh ID, refresh requires refresh purpose/nonempty ID before any DB query. Malformed parse results cannot be nil-dereferenced.
- Refresh validation additionally matches ID **and user ID**, non-revoked state and unexpired stored expiry. A conditional non-revoked-to-revoked update must affect exactly one row before refresh rotation continues. Already revoked refresh tokens cannot authenticate as access tokens: access validation rejects **all** refresh-purpose tokens independently of DB state.
- **Legacy JWTs without purpose are rejected on both paths. Existing users must sign in again; guests may need to register again and cannot rely on recovering their old in-memory lobby identity.** No schema/data migration is required; existing refresh rows can expire normally. A restart also loses in-memory lobbies. Do not weaken purpose validation to preserve old sessions, or roll back to the old validator simply to restore legacy tokens.
- Raw HTTP `Authorization: <access JWT>` and WebSocket offered subprotocols `['json', accessToken]` are unchanged, as are token-pair response fields, lifetimes and signing-key startup configuration.
- Logout/revocation still does not revoke already issued access JWTs immediately (15-minute lifetime plus 10-second validation leeway), nor reauthenticate/disconnect existing WebSockets. Refresh revoke-and-reissue is not a single transaction: a later failure may require re-login. Broader session lifecycle work remains separate.
- Auth header/token/claim logs, token-bearing HTTP response debug dumps, DB DSN logging and raw DB error details in touched paths are removed. GORM automatic SQL/error logging is disabled to prevent credentials, hashes and token rows from leaking; safe application-level context remains. Operators lose raw SQL diagnostics and should use deliberately redacted diagnostics rather than re-enable credential-bearing logs. Database startup checks migration failure before serving; see [database startup](database-startup.md).

Readiness, sequencing and additional command errors are documented in the
[socket contract](socket-contract.md). Country normalization and scoring policy
validation are documented in [region scoring](region-scoring.md).
