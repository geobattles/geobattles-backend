# Lobby lifecycle and frontend wire contract

The [socket contract](socket-contract.md) specifies readiness, ordering, round time, match identity and journal archives. [Powerups](powerups.md) specifies the separate intermission decision barrier, effect ledger and receipts. HTTP uses raw-JWT authorization and WebSockets negotiate `json`.

## Exact payload shapes

Notation below describes type shapes. `Record<number, T>` has decimal-string JSON keys. Empty maps are `{}`, empty arrays `[]`. Optional fields are omitted, not `null`; powerupState's targetRound/closesAt explicitly use null when absent.

```ts
type Phase = 'WAITING' | 'PLAYING' | 'MID-ROUND' | 'FINISHED'
type Coords = { lat: number; lng: number } // both axes ALWAYS emitted, including 0
type Player = { name: string; connected: boolean; ready: boolean; color: string;
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

Public GET `/lobby` returns `Record<string, PublicLobbySnapshot>` keyed by lobby ID. Authenticated POST `/lobby` returns one `PublicLobbySnapshot`; creation does not authorize member recovery. Both HTTP paths omit `location` and `revealedAnswers` in every phase. Authorized member WebSocket snapshots below retain the complete current/last location and all phase/results fields. Individual lobby snapshots are consistent; listing is not one cross-lobby instant. `numPlayers` includes retained away and disconnected members (capacity is membership + new admission reservations, not just connected sockets).

Server envelopes are `{status, type, seq?, payload?}`. The table lists message-specific fields; shared response payloads also serialize `timeRemaining: 0` when unused. Only start/rejoin and nested lobby time values are authoritative clock updates. Successful payloads:

| TYPE | Payload |
| --- | --- |
| `JOINED_LOBBY` | `{lobby: LobbySnapshot, user: string}` (unchanged base shape, broadcast) |
| `REJOIN_ROUND` | `{gameId?, phase, round, lobby: LobbySnapshot, players: Record<string,Player>, timeRemaining: number, fullroundRes: Record<string,Result[]>, roundRes: Record<string,Result>, totalResults: Record<string,Result>, powerLog: Powerup[], location?: Coords}` |
| `START_ROUND` | `{gameId: string, location: Coords, players: Record<string,Player>, powerLog: Powerup[], round: number, phase: 'PLAYING', timeRemaining: number}` |
| `NEW_RESULT` | `{gameId: string, user: string, playerRes: Result, round: number, phase: 'PLAYING'}` |
| `ROUND_FINISHED` | no payload, `status: 'WRN'` |
| `ROUND_RESULT` | `{gameId: string, round, phase: 'MID-ROUND'\|'FINISHED', players, fullroundRes, roundRes, totalResults, powerLog, location: Coords, polygon?: GeoJSON}` (`polygon` only existing mode2 path) |
| `GAME_END` | `{gameId: string, results: Record<number,Record<string,Result[]>>, endResults, revealedAnswers, totalResults, round, phase: 'FINISHED'}` |
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
  removes the member from the active roster. An absent seat only counts toward
  placement bonuses for rounds that player actually entered. Never infer intent from a
  WebSocket close code.
- Apply the restored snapshot as replacement state. It carries phase, round,
  roster, attempts, powerups, results, location and remaining milliseconds.
  Zero is a valid time value. Retained seat and empty-room cutoffs are described
  in the [socket contract](socket-contract.md#ownership-host-handoff-and-recovery).
- A completed summary remains until successful replay or room cleanup. Failed
  or cancelled starts preserve it. Lobby state is in memory.
- No guess is represented by `attempt === 0`, **not** `distance === 0`. Perfect guesses have `attempt > 0 && distance === 0`. Before showing/ranking a distance or marker, require `attempt > 0`, distance field presence, and `Number.isFinite(distance)`; use both coordinate axes even when 0. Historical stored rows are not backfilled. Equal valid guesses retain the existing tie/placement policy.

## Ownership and bounded effects

Private registry mutex only copies/inserts/deletes exact lobby pointers; it is always released before lobby.mu. Lobby.mu owns configuration, membership, exact connection leases/admission reservations, phases, generations, timers and scores. All event DTOs are copied and JSON-encoded within the transaction, then nonblocking-enqueued to 64-event private per-peer queues under a short hub mutex. No live state is encoded later. Slow/closed peers are detached and disconnected after all locks, forcing resync rather than silently skipping events. Only writer goroutines perform data writes; no hub goroutine is retained.

Location work uses a captured context/config and reserved generation outside locks. Leave/loss/expiry/closure cancels it; stale completion cannot commit. Cancellation enqueues `START_CANCELLED` to a still-owned/open initiator within the cancelling transaction, before another start can begin; a superseded provider completion stays silent. Closure or departure of that initiator does not require a reply to the dead connection. Pending admissions reject start/settings with `LOBBY_BUSY`; admissions reject while starting. Round finalization is one generation-guarded transaction shared by timer/guess/leave/expiry. Persistence receives one detached snapshot (at most 100 rounds x 12 best results) and one best-effort DB write outside locks with a 5s context. No durable retry/idempotency key is introduced.

## Operational limits

- Recovery is single-process and in-memory. Country Battle remains unfinished; historical stored results are not backfilled.
- Public discovery includes player/guess/result information but neither answer field. Authorized member snapshots still include the current/last answer; this is not an anti-cheat boundary.
- Bounded means existing per-game player/round/attempt limits and per-peer 64-event queues, not global lobby/rate limits. A slow peer reconnects via snapshot; there is no durable event replay. JSON copying/encoding remains inside the lobby transaction and can become costly near maximum history limits.
- Persistence is at-most-once attempted per completed in-memory game, not guaranteed durable delivery; DB failure/timeouts are logged safely with no automatic retry. Process exit loses retained state.
- Compile success is not multiplayer/provider/database/concurrency acceptance; exercise the intended release pair within the approved runtime scope.
