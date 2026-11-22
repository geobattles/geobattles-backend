# Powerups

Double Score (`type:0`) and Duel (`type:1`) are server-owned next-round effects.
Each enabled inventory bit is granted once at successful round-1 commit, not
per round. There is no first-round activation; one-round games have no activation
opportunity. Both items can be used for the same upcoming round.

## Decision window and advancement

Nonfinal ROUND_RESULT opens a maximum **20,000 ms** window when both powerups
are enabled in lobby settings, or **10,000 ms** otherwise, in MID-ROUND, targeting
the immediately following round. The duration depends on enabled settings, not
remaining player inventory. Only connected members with a live owning
socket and usable inventory delay advancement. Players can spend items or send
`powerup_ready` to irreversibly finish choosing. Exhausting usable items also
locks that member's choices, including if an item is subsequently refunded.

The window closes at its monotonic server deadline or when all eligible members
are done. Equality is closed. Validation/commit under the lobby mutex determines
acceptance, not browser timestamps or when a socket frame arrived. Delayed timer
callbacks grant no extra time. Inventory and effects commit atomically; rejection
never spends inventory.

Host `start` during MID-ROUND requires the displayed gameId and completed round.
It locks the host's choices and queues advancement; it cannot bypass another
eligible chooser. Once closed, an existing host intent reserves location
selection once. Expiry without an intent stays on results, waiting for the host.
No usable items means no decision delay. This is separate from `setReady`, which
continues to apply only to initial games/replays in WAITING/FINISHED.

Membership loss/removal cancels queued/reserved starts under the existing
selection rules. Host loss requires the current host to request again. Failed/cancelled
selection clears intent but does not reopen the window or undo accepted effects;
the host may retry against the same closed window. Old generation timers and
provider completions cannot affect newer rounds or disposed lobbies.

## Commands and retries

```ts
type UsePowerup = {
  command: 'use_powerup'; gameId: string; round: number; requestId: string;
  powerup: { type: 0 | 1; target?: string };
};
type PowerupReady = {
  command: 'powerup_ready'; gameId: string; round: number; requestId: string;
};
type MidRoundStart = { command: 'start'; gameId: string; round: number };
```

Here `round` is the **completed** intermission round, not the affected round.
Require explicit integer round/type fields; missing/null type never means zero.
Request IDs contain 1–64 ASCII letters/digits/underscore/hyphen. Source is always
the exact authenticated owning member; any supplied source is ignored. Double
Score rejects a nonempty target. Duel requires another connected owning member
ID, not a display name. A done opponent remains a valid Duel target.

Accepted use/ready receipts are retained per membership/game/requestId. Exact
retries return the same receipt identity and **current** state, even after cutoff,
advancement or refund. Different normalized content with that ID rejects with
REQUEST_ID_CONFLICT. Do not silently retry a pending request after reconnect:
first reconcile the authoritative ledger/done state. An intentional replacement
after refund needs a new ID. A new ready ID after choosing done is rejected.
Rejected arbitrary requests are not cached.

At most `2 * maxPlayers * numRounds` effects and `3 * maxPlayers * numRounds`
accepted use/ready receipts are retained per game. Caps reject without spending
or changing done state; existing accepted retries remain valid. Successful replay
clears the ledger/window/receipts; failed replay preserves the finished state.
An old game's request never applies to a replay.

## Effects, scoring and membership

Effects transition QUEUED → ACTIVE at the affected round's commit → APPLIED at
once-only finalization. They never drift to another round during provider
failures. At most one unresolved/applied effect occupies a membership/type/round
slot. Refunded/cancelled records remain in the bounded ledger.

- **Double Score:** `dblScr = baseScr`. No-guess doubles zero. Placement and Duel
  bonuses are not multiplied. `total = baseScr + placeScr + dblScr + duelScr`.
- **Duel:** unilateral, no target acceptance or target inventory cost. Each
  activation transfers 1,000 from loser to winner. Multiple incoming and
  reciprocal Duels settle independently and stack; totals may be negative.
- In distance modes, an attempted guess beats missing, lower best distance wins,
  equal valid distances/neither attempting draw. A real zero-distance attempt is
  valid. In mode 2, a correct country (`attempt>0 && cc==='XX'`) beats wrong or
  missing; faster correct time wins; equal correct times/neither correct draw.
  This does not otherwise complete Country Battle.
- Transient loss preserves effects/inventory and existing guesses. Permanent
  source removal cancels its unresolved effects without a refund to a nonexistent
  member. Permanent target removal promptly refunds a retained source's unresolved
  Duel once. Membership generations prevent a fresh admission of the same ID from
  inheriting prior effects. Settled effects never refund/reverse on later leave.
- Refund cannot reopen a closed window or undo a done choice. An unlocked source
  can use a refunded item again only while the original window remains open.
  Final-round effects settle before GAME_END.

## Authoritative state and events

```ts
type PowerupEffect = {
  id: string; requestId: string; gameId: string; round: number;
  type: 0 | 1; source: string; target?: string;
  status: 'QUEUED' | 'ACTIVE' | 'APPLIED' | 'CANCELLED' | 'REFUNDED';
  outcome?: 'SOURCE_WON' | 'TARGET_WON' | 'DRAW';
  reason?: 'SOURCE_LEFT' | 'TARGET_LEFT';
};
type PowerupState = {
  gameId?: string; round: number; targetRound: number | null;
  status: 'OPEN' | 'CLOSED'; closesAt: number | null; serverNow: number;
  remainingMs: number; decisionReady: Record<string, boolean>;
  startRequested: boolean; starting: boolean; effects: PowerupEffect[];
};
```

`round` on an effect is the affected round. State `round` is the current/completed
lobby round. `targetRound` is round+1 only in MID-ROUND. `closesAt`/`serverNow` are
Unix epoch milliseconds; remainingMs derives from the server monotonic deadline
and is zero when closed. No tick broadcasts occur. Count down locally relative
to receipt time without extending an existing window; server closure always wins.
`decisionReady` projects connected participants, including their irreversible done
flags after reconnect. `effects` is the full stable-order bounded match ledger.

Member LobbySnapshot, START_ROUND, ROUND_RESULT and GAME_END include
`powerupState`; REJOIN_ROUND includes it through `payload.lobby`. WAITING,
PLAYING and FINISHED report CLOSED. Public HTTP discovery/creation omit it.
The legacy `powerLog` remains the current-round `{type,source,target?}` array,
not a queued-effects ledger. Player inventory remains `[doubleAvailable,duelAvailable]`.

| Status / type | Dedicated payload |
| --- | --- |
| OK / POWERUP_STATE | `{gameId,round,phase,players,powerupState}` |
| OK / POWERUP_USED | `{gameId,requestId,effect,powerupState,players}` |
| OK / POWERUP_READY | `{gameId,requestId,powerupState}` |
| WRN / DUEL_FROM | `{user:source,gameId,round:affectedRound,effectId}` |

Changes to choices/inventory/window/participants/start intent are broadcast with
POWERUP_STATE; ROUND_RESULT supplies the initial opening. State is enqueued before
the source's receipt and before any triggered selection/START_ROUND. Duel notice
is sent once per new activation, never on retry; the ledger recovers missed
notifications. DTOs are detached/encoded under the same mutex and increasing seq
rules as other game events. Dedicated events do not update the live round clock,
render maps or independently advance the client phase.

Errors remain ERR / REQUEST_ERROR with `{command,code,message}` plus independently
well-formed `requestId?`, `gameId?`, `round?`. Codes are INVALID_REQUEST,
STALE_POWERUP_CONTEXT, POWERUP_WINDOW_CLOSED, POWERUP_DECISION_DONE, NOT_AVAILABLE,
WRONG_TARGET, WRONG_TYPE, REQUEST_ID_CONFLICT, POWERUP_LIMIT, plus existing
ownership/start errors. Shape is checked before accepted retries/conflicts,
context, window, done, inventory/target/limit and commit. A cutoff-triggered
closure is broadcast before the rejection. Missing state from an older backend
means unsupported controls, not permission to infer an activation window.

## Local verification

`go test ./...` runs focused standard-library Go regressions for these powerups,
including real loopback WebSocket envelope/ordering checks, without starting the
application or connecting to a database. Provider-failure cases explicitly blank
the Google key and exercise the unconfigured path, not a substitute provider.
`go test ./pkg/game -count=20` repeats timing/transition coverage. These commands
are distinct from `make check`, which remains static/vet/build only. Full paired
frontend, browser, database and Maps-backed multiplayer acceptance is separate.
Recovery remains in-memory only; no schema migration, durable receipt store or
process-restart guarantee is added.
