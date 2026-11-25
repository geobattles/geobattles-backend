# Lobby lifecycle and frontend wire contract

The [socket contract](socket-contract.md) specifies readiness, ordering, round time, match identity and journal archives. [Powerups](powerups.md) specifies the separate intermission decision barrier, effect ledger and receipts. HTTP uses raw-JWT authorization and WebSockets negotiate `json`.

## Exact payload shapes

Notation below describes type shapes. `Record<number, T>` has decimal-string JSON keys. Empty maps are `{}`, empty arrays `[]`. Optional fields are omitted, not `null`; powerupState's targetRound/closesAt explicitly use null when absent.

```ts
type Phase = 'WAITING' | 'PLAYING' | 'MID-ROUND' | 'FINISHED'
type Coords = { lat: number; lng: number } // both axes ALWAYS emitted, including 0
type Player = { name: string; connected: boolean; ready: boolean; color: string; rematchSeat?: boolean;
  away?: boolean; powerups?: boolean[]; lives?: number } // legacy omission of zero lives retained
type Result = { location: Coords; distance: number; baseScr: number;
  attempt: number; lives: number; placeScr?: number; dblScr?: number;
  duelScr?: number; total?: number; cc?: string; time?: number }
type Powerup = { type: number; source: string; target?: string }
type LobbyConf = { name: string; mode: number; maxPlayers: number;
  numAttempt: number; numRounds: number; roundTime: number; scoreFactor?: number;
  scoringMode: 'auto' | 'manual'; scoreScaleKm: number; // authoritative decay half-distance in km
  ccList: string[]; powerups?: boolean[]; placeBonus?: boolean; dynLives: boolean }
type LobbySnapshot = {
  ID: string; gameId?: string; admin: string; conf: LobbyConf; numPlayers: number;
  endedEarly?: boolean; spectating?: boolean; spectatorCount?: number; openToSpectators?: boolean;
  playerList: Record<string, Player>;
  currentRound: number; round: number; phase: Phase;
  results: Record<number, Record<string, Result[]>>;
  endResults: Record<number, Record<string, Result>>;
  revealedAnswers?: Record<number, Coords>;
  totalResults: Record<string, Result>;
  fullroundRes: Record<string, Result[]>;
  roundRes: Record<string, Result>;
  timeRemaining: number; // milliseconds, explicitly 0 when expired/non-playing
  powerLog: Powerup[];
  powerupState?: PowerupState; // member-only, see powerups.md
  location?: Coords; // MEMBER snapshots: absent in WAITING; current/last otherwise
}
type PublicLobbySnapshot = Omit<LobbySnapshot, 'location' | 'revealedAnswers' | 'powerupState'>
```

Public GET `/lobby` returns `Record<string, PublicLobbySnapshot>` keyed by lobby ID. Authenticated POST `/lobby` returns one `PublicLobbySnapshot`. Both omit `location`, `revealedAnswers`, private powerup state and live round results. Player WebSocket snapshots retain the complete current/last location and all phase/results fields. Individual lobby snapshots are consistent. In player and HTTP snapshots, `numPlayers` includes retained away and disconnected players.

Server envelopes are `{status, type, seq?, payload?}`. The table lists message-specific fields; shared response payloads also serialize `timeRemaining: 0` when unused. Only start/rejoin and nested lobby time values are authoritative clock updates. Successful payloads:

| TYPE | Payload |
| --- | --- |
| `JOINED_LOBBY` | `{lobby: LobbySnapshot, user: string}`. Broadcast to players for a player admission. Unicast with the spectator projection for a watcher admission or recovery. |
| `REJOIN_ROUND` | `{gameId?, phase, round, lobby: LobbySnapshot, players: Record<string,Player>, timeRemaining: number, fullroundRes: Record<string,Result[]>, roundRes: Record<string,Result>, totalResults: Record<string,Result>, powerLog: Powerup[], location?: Coords}` |
| `START_ROUND` | `{gameId: string, location: Coords, players: Record<string,Player>, powerLog: Powerup[], round: number, phase: 'PLAYING', timeRemaining: number}` |
| `NEW_RESULT` | `{gameId: string, user: string, playerRes: Result, round: number, phase: 'PLAYING'}` |
| `ROUND_FINISHED` | no payload, `status: 'WRN'` |
| `ROUND_RESULT` | `{gameId: string, round, phase: 'MID-ROUND'\|'FINISHED', players, fullroundRes, roundRes, totalResults, powerLog, location: Coords, polygon?: GeoJSON}` (`polygon` only existing mode2 path) |
| `GAME_END` | `{gameId: string, results: Record<number,Record<string,Result[]>>, endResults, revealedAnswers, totalResults, players, round, phase: 'FINISHED', endedEarly?: boolean}` |
| `UPDATED_LOBBY` | `{lobby: LobbySnapshot}` |
| `LEFT_LOBBY` | `{lobby: LobbySnapshot, user: string}` |
| `DUEL_FROM` | `{user,gameId,round: affectedRound,effectId}`, `status: 'WRN'` |
| `POWERUP_USED` | `{gameId,requestId,effect,powerupState,players}` |
| `POWERUP_READY` | `{gameId,requestId,powerupState}` |
| `POWERUP_STATE` | `{gameId,round,phase,players,powerupState}` |
| `CC` | `{cc: string, polygon?: GeoJSON}` |

START_ROUND, ROUND_RESULT and GAME_END additionally carry `powerupState`;
REJOIN_ROUND carries it through `payload.lobby`. See [powerups](powerups.md)
for its exact fields, null semantics and context-bound request errors.

## Spectating and rematches

Send `spectate=1` with the lobby ID on WebSocket upgrades and advisory
`/lobby/admission` requests. A retained player always recovers their player
seat. New clients can watch an active match or a full rematch. Older clients
receive the existing admission behavior. A room can hold 12 spectators in
addition to its configured player capacity. Detached spectators have the same
90-second return window and can reconnect during the round. Explicit watcher
leave releases their place immediately.

Spectator `JOINED_LOBBY` and `UPDATED_LOBBY` snapshots set `spectating: true`.
They carry the round clock, connected competitor and watcher counts, settled
totals and the best guesses and answers from finalized rounds. Their player
roster preserves names for departed competitors. Current results, raw attempts,
inventories, powerup decisions and current locations are excluded. Player-only
events use a separate audience. Spectator commands other than heartbeat and
leave are rejected with `SPECTATOR_READ_ONLY`.

When a match ends or a rematch seat opens, connected spectators get available
seats in admission order. Promotion sends a player `UPDATED_LOBBY` on the
same socket. A promoted player's `rematchSeat` flag lasts until the next
match starts. They opt into that match through Ready Check.
Spectators above the player limit stay connected and can watch the next match.

## Ending a match early

The current host can send `{command:'end_game', gameId, round}` in
`PLAYING` or `MID-ROUND`. When a round is live, the server settles the
submitted guesses and then finishes the match. Between rounds it keeps the
settled scores and cancels any pending next-round start and powerups.
`GAME_END` and later snapshots set `endedEarly: true`. The saved results
contain only the rounds played, and the configured length applies to the next
match.

Except the noted `WRN` types, these statuses are `OK`. `REJOIN_ROUND` is unicast to a returning member, enqueued **after** that member's `JOINED_LOBBY` in all four phases, including WAITING (round/timeRemaining 0). For a fresh member, use the complete JOINED snapshot. On successful replay `START_ROUND.round` is 1; never infer round by incrementing a client-local counter. Final successful guess `NEW_RESULT` is enqueued before `ROUND_FINISHED`, `ROUND_RESULT`, `GAME_END`. Timer/leave completion has no extra `NEW_RESULT`.

`REQUEST_ERROR` remains `status:'ERR', payload:{command,code,message}`. Lifecycle error codes: `LOBBY_BUSY` (an upgrade reservation is pending; retry), `START_CANCELLED` (membership changed during selection). Existing `ROUND_STARTING`, provider and validation codes remain. Admission conflicts return HTTP 409 (including connected/pending duplicate identity, full, game-in-progress, expired seat, start-in-progress); missing/closed lobby returns HTTP 400. A duplicate does not take over the live connection. Join checks the absolute 10s reservation deadline and exact reservation/previous seat under lobby.mu, independently of timer scheduling. A reservation that expires during an already-upgraded handshake rolls back and closes that exact new socket.

## Recovery/leave rules for the client

Country catalog and region-dependent scoring are specified in
[region-scoring.md](region-scoring.md). New configurations default to `auto`;
manual scoring requires explicit `scoringMode:'manual'`. `scoreScaleKm` is
read-only and recomputed by the server. These scalar fields are included in all
cloned public/member configuration projections, including recovery and replay.

- Transport loss and the resumable `leave` command retain the player’s color,
  lives, powerups, and raw/best/total results in the current match. `away:true`
  marks an explicit resumable leave. The permanent `disconnect` command
  removes the member from the active roster. Use that roster and the `away`
  flag to display presence. Member snapshots carry `matchRoster` with display
  names and colors for historical results, including removed players.
- Each round's score rows contain only players connected when the round opened
  or admitted before its deadline. Finishing the round preserves their guesses
  and any earned points after they leave. A player who is absent for an entire
  round has no result for it and does not hold up completion. Returning during
  an active round preserves any existing guesses, lives and the remaining clock.
- Apply the restored snapshot as replacement state. It carries phase, round,
  roster, attempts, powerups, results, location and remaining milliseconds.
  Zero is a valid time value. Retained seat and empty-room cutoffs are described
  in the [socket contract](socket-contract.md#ownership-host-handoff-and-recovery).
- A completed summary remains until successful replay or room cleanup. Failed
  or cancelled starts preserve it. Lobby state is in memory.
- No guess is represented by `attempt === 0`, **not** `distance === 0`. Perfect guesses have `attempt > 0 && distance === 0`. Before showing/ranking a distance or marker, require `attempt > 0`, distance field presence, and `Number.isFinite(distance)`; use both coordinate axes even when 0. Historical stored rows are not backfilled. Equal valid guesses retain the existing tie/placement policy.

## Ownership and bounded effects

Private registry mutex only copies/inserts/deletes exact lobby pointers; it is always released before lobby.mu. Lobby.mu owns configuration, membership, exact connection leases/admission reservations, phases, generations, timers and scores. All event DTOs are copied and JSON-encoded within the transaction, then nonblocking-enqueued to 64-event private per-peer queues under a short hub mutex. No live state is encoded later. Slow/closed peers are detached and disconnected after all locks, forcing resync rather than silently skipping events. Only writer goroutines perform data writes; no hub goroutine is retained.

Location work uses a captured context/config and reserved generation outside locks. Closing the room, losing the host, or ending the match cancels pending location selection. A superseded provider completion cannot commit. Pending player admissions reject start/settings with `LOBBY_BUSY`. Spectator upgrades run independently of location selection. Round finalization is one generation-guarded transaction shared by timer, guess, leave, expiry and early ending. Persistence receives one detached snapshot (at most 100 rounds × 12 best results) and one best-effort DB write outside locks with a 5s context.

## Operational limits

- Recovery is single-process and in-memory. Country Battle remains unfinished; historical stored results are not backfilled.
- The legacy public list retains discovery metadata and its player roster for older clients. Score and guess fields stay empty. The directory uses counts and room metadata. Player socket snapshots include the current or last answer.
- Bounded means existing per-game player/round/attempt limits and per-peer 64-event queues, not global lobby/rate limits. A slow peer reconnects via snapshot; there is no durable event replay. JSON copying/encoding remains inside the lobby transaction and can become costly near maximum history limits.
- Persistence is at-most-once attempted per completed in-memory game, not guaranteed durable delivery; DB failure/timeouts are logged safely with no automatic retry. Process exit loses retained state.
- Compile success is not multiplayer/provider/database/concurrency acceptance; exercise the intended release pair within the approved runtime scope.

## Directory and return eligibility

`GET /lobby/directory` returns summaries keyed by room code. Each row contains
`ID`, `name`, optional `hostName`, `phase`, `currentRound`, `numRounds`,
`connected`, `occupied`, `maxPlayers`, `openToNew`, optional `starting`, and optional `openToSpectators`.
Occupied seats include disconnected members and in-flight new admissions. A
raw JWT in the Authorization header adds `retained: true` only to the caller's
own rooms. Responses use private, no-store caching and vary by Authorization.

`GET /lobby/directory/{code}` returns one public row, or HTTP 404 with
`{"code":"LOBBY_NOT_FOUND"}`. Home verifies its saved code using authenticated
`GET /lobby/admission?id=…`. A successful new admission has no retained flag.
The WebSocket handshake makes the final decision.

The original `GET /lobby` remains available for older clients, the frontend
socket's legacy admission fallback, the socket smoke suite, and the browser
suite's public-lobby inspection. Roll out the additive
backend routes before the frontend. Retire the legacy response separately
after those consumers have migrated.
