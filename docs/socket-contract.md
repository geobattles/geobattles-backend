# Socket contract: Ready Check, recovery and match identity

This contract complements
[lobby-lifecycle.md](lobby-lifecycle.md). Shapes below are specifications, not
runtime captures. Country Battle is not completed.

## Transport and ordering

- HTTP authorization is the existing **raw JWT**. WebSocket clients offer
  `["json", accessToken]`; the server negotiates `json`.
- Game envelopes are `{status, type, seq, payload?}`. `seq` is a positive,
  monotonically increasing integer per live lobby, assigned under the lobby
  mutex before encoding/enqueue. It does not reset for replay. Every broadcast
  and transaction-bound unicast (including `REQUEST_ERROR`, `CC`, powerup replies
  and recovery) consumes one number. Broadcast recipients receive the same seq.
- Gaps are expected because other players receive private replies. There is no
  durable event replay. Ignore stale/duplicate seq within a connection generation;
  reset that fence for a fresh socket generation and replace state from its
  authoritative JOINED/REJOIN snapshot. Do not require seq to start at 1 or be
  contiguous. Legacy/stateless errors and heartbeat messages may omit seq.
- The server still sends `{"command":"ping"}`; legacy clients can reply with
  `{"command":"pong"}`. It additionally sends RFC6455 control pings every 3s;
  browsers answer those without JavaScript. Either control pong or JSON pong
  refreshes the 10s read deadline. A JSON pong with a positive numeric `probe`
  opts into a `{"command":"heartbeat_ack","probe":...}` reply for browser latency
  sampling. Heartbeats carry no seq.
- Readers accept JSON **text** frames, with a 16KiB per-message limit, and no unbounded inbound
  work queue. Writers alone write data frames, using a 5s write deadline and a
  private 64-event queue. A full queue detaches the peer instead of dropping an
  event and continuing a stale stream.
- Close sends a best-effort control frame with a 250ms deadline, then closes the
  underlying connection; callbacks execute once and outside transport/game
  locks. `1000` is used for explicit exit commands and acknowledgements of
  normal peer close, `1001` for ordinary transport teardown, `1003` for binary
  commands, `1009` for Gorilla's oversize rejection, and `1013` for a slow peer
  or an upgrade whose admission expired. Delivery of a close frame is not
  guaranteed over a lost connection. **Close code alone never means membership
  was removed.**

## Authoritative round time

- `START_ROUND.payload.timeRemaining` is a nonnegative integer in **milliseconds**,
  sampled from the live round deadline under the lobby mutex after successful
  start commit. It uses the same `remainingLocked()` calculation as
  `LobbySnapshot.timeRemaining` and `REJOIN_ROUND.payload.timeRemaining`, not a
  fresh duration reconstructed from settings. Sub-millisecond fractions truncate.
- The existing deadline remains configured `roundTime` plus three seconds. The
  server's one-shot timer, deadline checks on guesses and once-only round
  finalization are unchanged; no tick broadcasts or new commands are added.
- Zero is explicitly serialized, including on START_ROUND; it means no time
  remains, not permission to restart a client countdown. The shared response
  payload also emits `timeRemaining: 0` on other message types where it is unused.
  Do not use that default as a clock update: lobby messages carry the authoritative
  value in `payload.lobby`, and REJOIN_ROUND retains its dedicated payload.
- Older backends may omit the START_ROUND field. Clients may fall back to the
  legacy settings-derived duration **only when the field is absent**, never for
  zero. An existing countdown for the same game/round must not move forward when
  applying later snapshots. A new game/round can initialize a new countdown.
- Remaining time is sampled while preparing the server event, not an absolute
  timestamp or round-trip clock synchronization. Network/queue delay is not
  compensated; the server remains authoritative for guess acceptance and round
  completion.

## Ready Check

Region selection adds `conf.scoringMode` and the read-only `conf.scoreScaleKm`;
see [region-scoring.md](region-scoring.md) for exact catalog, normalization,
automatic/manual curves and update semantics. Settings still preserve Ready
flags. No `regionMode` or Advanced-specific socket command is introduced.

```ts
// New command; ready:false is a real request, not an omitted value.
type SetReady = { command: 'setReady'; ready: boolean }
type Player = {
  name: string; connected: boolean; ready: boolean; color: string
  away?: boolean; powerups?: boolean[]; lives?: number
}
```

- All players, including the administrator, initially have `ready:false`.
  Successful setReady is acknowledged to everyone by the existing
  `OK / UPDATED_LOBBY / {lobby: LobbySnapshot}`. It is not an optimistic toggle;
  sending the same value again is valid and receives an authoritative snapshot.
- Only the exact owning connected socket can change its own ready flag.
  Missing/null ready returns `INVALID_READY`; nonboolean or malformed JSON
  returns `INVALID_REQUEST`. Other validation/error conventions are unchanged.
- setReady is accepted only in `WAITING` and `FINISHED`. During selection it
  returns `ROUND_STARTING`; during a game it returns `GAME_IN_PROGRESS`.
- An administrator’s initial start or replay start requires at least one live
  member and readiness from **every live member**; otherwise it
  returns `REQUEST_ERROR {command:'start', code:'PLAYERS_NOT_READY', message}`
  before provider work. Normal authorization/active/reservation conflicts retain
  their existing priority. Readiness is rechecked at successful selection commit.
- **MID-ROUND advancement does not use Ready Check.** It uses the separate
  [powerup decision window](powerups.md): context-bound host start queues until
  eligible connected choosers finish or the 10s cutoff closes. Offline retained
  members do not delay that window.
- New membership and permanent removal by command reset everyone’s ready
  flag. Lobby settings updates preserve every member's existing ready flag,
  including changes to mode, capacity, attempts, rounds, round time, score factor,
  country list, powerups, placement bonus and dynamic lives. No-op and title-only
  updates also preserve readiness. The existing numeric omitted/zero no-update
  meanings, settings validation and start gates remain unchanged.
- Transport loss or resumable leave clears only that member’s ready flag. Recovery is **not** an
  automatic ready acknowledgement and leaves that member unready; it does not
  reset the other members. This preserves the distinction between connection
  loss and membership removal.
- Final game completion clears all ready flags before the final ROUND_RESULT
  players are copied. A FINISHED recovery snapshot therefore has unready players
  until they explicitly acknowledge a replay. Ready Check never resets lives,
  scores, powerups, round deadlines or retained completed summaries.

## Stable gameId

A crypto-random UUIDv4 from the existing uuid dependency is created **only** at
successful initial/replay round-1 commit, after provider selection and ownership
checks. Failed, rejected or cancelled starts keep the prior ID and summary.
Normal next rounds, reconnects and completion preserve the ID. A lobby that has
never started has no gameId field. It is an in-memory match identifier, **not**
the database-generated game ID or a durable restart/retry guarantee.

| Existing shape | Additive field |
| --- | --- |
| `LobbySnapshot` (HTTP and member socket snapshots) | `gameId?: string` |
| `REJOIN_ROUND.payload` | `gameId?: string` (also in payload.lobby) |
| `START_ROUND.payload` | `gameId: string` |
| `NEW_RESULT.payload` | `gameId: string` |
| `ROUND_RESULT.payload` | `gameId: string` |
| `GAME_END.payload` | `gameId: string` |

JOINED_LOBBY / UPDATED_LOBBY / LEFT_LOBBY carry it in their nested lobby. The
payload-less ROUND_FINISHED remains payload-less. Frontends should allow absent
fields when talking to an older server. Use gameId to deduplicate a local match
journal, not a round number or mutable lobby title. For recovery of a completed
game, use the snapshot's complete results/endResults/revealedAnswers/totalResults
and the same gameId.

## Authoritative journal archive

These are additional fields on the base shapes in `lobby-lifecycle.md`, not
runtime captures. Round keys are 1-based decimal-string JSON object keys;
`Coords` and `Result` retain their existing shapes (both coordinate axes always
serialize, including zero).

```ts
// Member LobbySnapshot, nested in JOINED_LOBBY / REJOIN_ROUND and other member
// lobby snapshots. No new top-level field is required on REJOIN_ROUND.
type MemberJournalArchive = {
  revealedAnswers?: Record<number, Coords>
  // Already present on LobbySnapshot:
  endResults: Record<number, Record<string, Result>>
}

// Additive GAME_END.payload fields; existing results/totalResults/gameId/
// round/phase remain unchanged.
type FinalJournalArchive = {
  revealedAnswers?: Record<number, Coords>
  endResults?: Record<number, Record<string, Result>>
}
```

- `revealedAnswers` contains detached coordinate values for **finalized rounds
  only**, joined to existing `endResults` entries and valid archived locations.
  An `endResults` entry alone is not proof of completion: it is allocated when
  a round starts. Earlier rounds have finalized before advancement; the current
  round is included only in MID-ROUND/FINISHED after its generation finalizer.
- WAITING has no archive (the optional field is omitted when empty). PLAYING may
  include completed earlier rounds, **never the current PLAYING answer**.
  MID-ROUND/FINISHED include all completed rounds. This does not remove or change
  the existing member-only `location` field, which still carries the current or
  last answer outside WAITING.
- `GAME_END` always supplies both archives on this server, copied from the same
  detached final snapshot as `results` and `totalResults`. The scored
  `endResults[round][playerId]` values include finalized bonuses/totals. A client
  no longer needs every preceding ROUND_RESULT to assemble a final debrief.
- JOINED_LOBBY and REJOIN_ROUND obtain the archive through `payload.lobby`;
  other member lobby snapshots use the same projection. Recovery is replacement
  state, not permission to increment totals or replay scoring.
- Failed/cancelled replay selection preserves the previous answers/results and
  gameId. Successful replay uses the existing results/locations reset; round 1
  starts without the prior archive. No extra reset or business logic is added.
- HTTP GET `/lobby` and POST `/lobby` **omit both** `location` and
  `revealedAnswers` in every phase. Public discovery is not a journal source.
- Older servers may omit the new fields. Only persist a completed journal when
  authoritative answers and scored round results cover every completed round;
  do not synthesize them from raw guesses or aggregate totals. Deduplicate by
  gameId and continue to scope persisted records to the authenticated player.

The archive remains in-memory and follows the existing membership/grace/lobby
lifetime; it does not provide process-restart recovery or durable history.

## Ownership, host handoff and recovery

- Identity and capacity are reserved before upgrade for at most 10s. Commit
  rechecks the exact reservation, deadlines and previous seat. A connected or
  pending duplicate gets HTTP 409 `ALREADY_CONNECTED` and the old socket keeps
  its lease. Failed upgrades roll back only their own reservation.
- Authenticated `GET /lobby/admission?id=<code>` is a no-store advisory check for
  the calling identity. It returns `{code, retained?, retryAfterMs?}` with no
  roster, scores, answers or reservation. `ADMISSION_PENDING` distinguishes an
  upgrade still in flight. `OK`, `ROUND_STARTING`, `ALREADY_CONNECTED`,
  `REJOIN_EXPIRED`, `GAME_IN_PROGRESS`, `LOBBY_FULL`, and `LOBBY_NOT_FOUND`
  cover the other outcomes. Only the upgrade can admit a player.
- The authenticated creator ID belongs to the lobby. The longest-present live
  member fills a vacant host slot, and the creator reclaims `admin` on any
  successful return. Other returning members leave a live admin in place. A
  room with no live members has an empty `admin` field.
- Transport loss and the `leave` command keep the player’s seat and join order.
  A resumable leave broadcasts `connected:false, away:true`. Transport loss
  broadcasts `connected:false`. The clock continues. A lost connection can
  delay early completion for up to 30s from detection, during which time the
  player can come back and finish their guesses. Deliberate leave immediately
  stops blocking early completion.
- Seats remain claimable through an in-progress match while the room has live
  members. In WAITING or FINISHED the seat times out 90s after departure.
  A successful replay removes absent seats before starting the new match. With
  everyone absent the room expires after 90s. `disconnect` still permanently
  removes membership and closes its exact socket.
- JOINED_LOBBY is always enqueued before REJOIN_ROUND for a recovering member,
  including WAITING/FINISHED. The existing complete snapshots restore phase,
  authoritative round, players (including readiness), raw/best/total results, powerups,
  finalized answer archive, current/last location and remaining milliseconds,
  including explicit zero. Apply
  replacement state; do not increment round or add the totals again.
- Loss, exit commands and disposal cancel pending provider work and invalidates its
  generation. An open exact initiating socket receives START_CANCELLED before
  a newer start can begin. A stale completion cannot commit/reset results or
  resurrect a disposed lobby. Exact lease/generation checks make late callbacks
  from an old socket harmless after recovery.
- After the idle cutoff or last retained seat is removed and admissions resolve,
  dispose the lobby, stop its round, seat, admission and idle timers, cancel selection,
  close its hub and remove only that exact registry entry. Never-joined lobbies
  expire after 30s. There is no process-restart recovery or global rate/lobby cap.

HTTP discovery/creation omit both the answer `location` and `revealedAnswers`;
authenticated member recovery retains the current/last answer as before and
adds finalized answers for the journal. [Powerup state](powerups.md) recovers
inventory, queued/active effects, done/start status and settlement outcomes;
Duel ties/neither-valid guesses draw. The one bounded best-effort completed-game
database write remains unchanged. Final guess ordering
stays NEW_RESULT -> ROUND_FINISHED -> ROUND_RESULT -> GAME_END.
