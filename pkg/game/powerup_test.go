package game

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
)

func powerLobby(t *testing.T) *Lobby {
	t.Helper()
	c := defaultConf(LobbyConf{NumRounds: 3, MaxPlayers: 4})
	l := &Lobby{conf: &c, hub: websocket.NewHub(), players: map[string]*Player{}, leases: map[string]*playerLease{}, reservations: map[string]*Admission{}, phase: PhasePlaying}
	l.resetResultsLocked()
	l.gameID = "match"
	l.round = 1
	for i, id := range []string{"a", "b", "c"} {
		l.players[id] = &Player{Connected: true, Powerups: []bool{true, true}}
		client := websocket.NewClient(id, id, l.hub, nil, l, PlayerMessageHandler, nil)
		l.leases[id] = &playerLease{client: client, present: uint64(i + 1)}
		l.totalResults[id] = &Result{}
	}
	l.admin = "a"
	l.endResults[1] = map[string]*Result{"a": {}, "b": {}, "c": {}}
	t.Cleanup(func() {
		if l.roundTimer != nil {
			l.roundTimer.Stop()
		}
		if l.powerWindow != nil && l.powerWindow.timer != nil {
			l.powerWindow.timer.Stop()
		}
		if l.startCancel != nil {
			l.startCancel()
		}
	})
	return l
}

func choice(id string, kind int, target string) ClientReq {
	game, round := "match", 1
	p, _ := json.Marshal(Powerup{Type: kind, Target: target})
	return ClientReq{Cmd: "use_powerup", GameID: &game, Round: &round, RequestID: &id, Powerup: p}
}
func readyChoice(id string) ClientReq {
	r := choice(id, 0, "")
	r.Cmd = "powerup_ready"
	r.Powerup = nil
	return r
}
func activate(l *Lobby, source string, r ClientReq) error {
	var e lobbyEffects
	return l.powerupCommandLocked(&e, l.leases[source].client, r)
}
func openChoices(l *Lobby) { l.phase = PhaseMidRound; l.openPowerupWindowLocked() }
func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil || err.Error() != code {
		t.Fatalf("want %s, got %v", code, err)
	}
}

func TestPowerupRejectsLivePlayWithoutSpending(t *testing.T) {
	l := powerLobby(t)
	requireCode(t, activate(l, "a", choice("one", 0, "")), "POWERUP_WINDOW_CLOSED")
	if !l.players["a"].Powerups[0] || len(l.powerEffects) != 0 {
		t.Fatal("live-play activation spent inventory")
	}
}

func TestDuelEqualDistancesAreDraw(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	if err := activate(l, "a", choice("duel", 1, "b")); err != nil {
		t.Fatal(err)
	}
	l.setupRoundLocked(logic.Coords{}, "")
	l.endResults[2]["a"] = &Result{Attempt: 1, Dist: 25}
	l.endResults[2]["b"] = &Result{Attempt: 1, Dist: 25}
	_ = l.processPowerups()
	if l.endResults[2]["a"].DuelScore != 0 || l.endResults[2]["b"].DuelScore != 0 || l.powerEffects[0].Outcome != "DRAW" {
		t.Fatal("equal valid distances must draw")
	}
}

func TestPowerupAtomicRetryAcrossAdvancement(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	r := choice("double", 0, "")
	if err := activate(l, "a", r); err != nil {
		t.Fatal(err)
	}
	id := l.powerEffects[0].ID
	if l.players["a"].Powerups[0] || l.powerEffects[0].Round != 2 {
		t.Fatal("activation not committed")
	}
	if err := activate(l, "a", r); err != nil {
		t.Fatal(err)
	}
	l.setupRoundLocked(logic.Coords{}, "")
	if err := activate(l, "a", r); err != nil {
		t.Fatal(err)
	}
	if len(l.powerEffects) != 1 || l.powerEffects[0].ID != id || l.powerEffects[0].Status != "ACTIVE" {
		t.Fatal("retry reapplied or replaced effect")
	}
	conflict := choice("double", 1, "b")
	requireCode(t, activate(l, "a", conflict), "REQUEST_ID_CONFLICT")
	requireCode(t, activate(l, "a", choice("stale", 0, "")), "STALE_POWERUP_CONTEXT")
}

func TestPowerupRejectsInvalidRequestsWithoutMutation(t *testing.T) {
	cases := []struct {
		name string
		edit func(*Lobby, *ClientReq)
		code string
	}{
		{"missing type", func(l *Lobby, r *ClientReq) { r.Powerup = json.RawMessage(`{}`) }, "INVALID_REQUEST"},
		{"null type", func(l *Lobby, r *ClientReq) { r.Powerup = json.RawMessage(`{"type":null}`) }, "INVALID_REQUEST"},
		{"fractional type", func(l *Lobby, r *ClientReq) { r.Powerup = json.RawMessage(`{"type":0.5}`) }, "INVALID_REQUEST"},
		{"null target", func(l *Lobby, r *ClientReq) { r.Powerup = json.RawMessage(`{"type":0,"target":null}`) }, "INVALID_REQUEST"},
		{"unknown type", func(l *Lobby, r *ClientReq) { r.Powerup = json.RawMessage(`{"type":7}`) }, "WRONG_TYPE"},
		{"missing game", func(l *Lobby, r *ClientReq) { r.GameID = nil }, "INVALID_REQUEST"},
		{"missing round", func(l *Lobby, r *ClientReq) { r.Round = nil }, "INVALID_REQUEST"},
		{"missing id", func(l *Lobby, r *ClientReq) { r.RequestID = nil }, "INVALID_REQUEST"},
		{"invalid id", func(l *Lobby, r *ClientReq) { v := "bad id"; r.RequestID = &v }, "INVALID_REQUEST"},
		{"old game", func(l *Lobby, r *ClientReq) { v := "old"; r.GameID = &v }, "STALE_POWERUP_CONTEXT"},
		{"old round", func(l *Lobby, r *ClientReq) { v := 2; r.Round = &v }, "STALE_POWERUP_CONTEXT"},
		{"double target", func(l *Lobby, r *ClientReq) { *r = choice("x", 0, "b") }, "WRONG_TARGET"},
		{"self duel", func(l *Lobby, r *ClientReq) { *r = choice("x", 1, "a") }, "WRONG_TARGET"},
		{"missing target", func(l *Lobby, r *ClientReq) { *r = choice("x", 1, "") }, "WRONG_TARGET"},
		{"offline target", func(l *Lobby, r *ClientReq) { *r = choice("x", 1, "b"); l.players["b"].Connected = false }, "WRONG_TARGET"},
		{"disabled", func(l *Lobby, r *ClientReq) { (*l.conf.Powerups)[0] = false }, "NOT_AVAILABLE"},
		{"spent", func(l *Lobby, r *ClientReq) { l.players["a"].Powerups[0] = false }, "NOT_AVAILABLE"},
		{"done", func(l *Lobby, r *ClientReq) { l.powerWindow.done[1] = true }, "POWERUP_DECISION_DONE"},
		{"waiting", func(l *Lobby, r *ClientReq) { l.phase = PhaseWaiting }, "POWERUP_WINDOW_CLOSED"},
		{"finished", func(l *Lobby, r *ClientReq) { l.phase = PhaseFinished }, "POWERUP_WINDOW_CLOSED"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			l := powerLobby(t)
			openChoices(l)
			r := choice("x", 0, "")
			tc.edit(l, &r)
			before := append([]bool{}, l.players["a"].Powerups...)
			requireCode(t, activate(l, "a", r), tc.code)
			if fmt.Sprint(before) != fmt.Sprint(l.players["a"].Powerups) || len(l.powerEffects) != 0 || len(l.powerReceipts) != 0 {
				t.Fatal("rejection mutated inventory/ledger/receipts")
			}
		})
	}
}

func TestCutoffReconcilesEvenWhenTimerDelayed(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	l.powerWindow.timer.Stop()
	l.powerWindow.deadline = time.Now()
	requireCode(t, activate(l, "a", choice("late", 0, "")), "POWERUP_WINDOW_CLOSED")
	if l.powerWindow.open || !l.players["a"].Powerups[0] {
		t.Fatal("cutoff did not close without spend")
	}
	var e lobbyEffects
	if err := l.requestPowerupStartLocked(&e, l.leases["a"].client, choice("", 0, "")); err != nil {
		t.Fatal(err)
	}
	// Country catalog may be unloaded in this provider-free package harness; the
	// separate queued-start test loads bundled data and verifies reservation.
	if l.powerWindow.open {
		t.Fatal("start reopened cutoff")
	}
}

func TestReadyIdempotencyAndIrreversibility(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	r := readyChoice("ready")
	if err := activate(l, "a", r); err != nil {
		t.Fatal(err)
	}
	if err := activate(l, "a", r); err != nil {
		t.Fatal(err)
	}
	requireCode(t, activate(l, "a", readyChoice("ready-again")), "POWERUP_DECISION_DONE")
	requireCode(t, activate(l, "a", choice("after-ready", 0, "")), "POWERUP_DECISION_DONE")
	l.players["a"].Connected = false
	var e lobbyEffects
	l.reconcilePowerupWindowLocked(&e)
	l.players["a"].Connected = true
	l.reconcilePowerupWindowLocked(&e)
	if !l.powerupStateLocked().DecisionReady["a"] {
		t.Fatal("reconnect forgot done choice")
	}
}

func TestDuelRefundAndRetryDoesNotRetarget(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	r := choice("duel", 1, "b")
	if err := activate(l, "a", r); err != nil {
		t.Fatal(err)
	}
	var e lobbyEffects
	l.removePowerupMemberLocked(&e, "b")
	delete(l.players, "b")
	delete(l.leases, "b")
	l.reconcilePowerupWindowLocked(&e)
	if !l.players["a"].Powerups[1] || l.powerEffects[0].Status != "REFUNDED" || len(l.powerLogs[2]) != 0 {
		t.Fatal("target removal did not refund promptly")
	}
	if err := activate(l, "a", r); err != nil {
		t.Fatal(err)
	}
	if len(l.powerEffects) != 1 || !l.players["a"].Powerups[1] {
		t.Fatal("refunded retry spent again")
	}
	if err := activate(l, "a", choice("replacement", 1, "c")); err != nil {
		t.Fatal(err)
	}
	l.removePowerupMemberLocked(&e, "a")
	if l.powerEffects[1].Status != "CANCELLED" {
		t.Fatal("source removal failed to cancel")
	}
}

func TestBothItemsExhaustionKeepsDoneAfterRefund(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	if err := activate(l, "a", choice("double", 0, "")); err != nil {
		t.Fatal(err)
	}
	if err := activate(l, "a", choice("duel", 1, "b")); err != nil {
		t.Fatal(err)
	}
	var e lobbyEffects
	l.removePowerupMemberLocked(&e, "b")
	delete(l.players, "b")
	delete(l.leases, "b")
	l.reconcilePowerupWindowLocked(&e)
	requireCode(t, activate(l, "a", choice("replacement", 1, "c")), "POWERUP_DECISION_DONE")
}

func TestDuelOutcomeRules(t *testing.T) {
	cases := []struct {
		name string
		mode int
		s, t Result
		want int
	}{
		{"perfect", 1, Result{Attempt: 1, Dist: 0}, Result{Attempt: 1, Dist: 1}, 1},
		{"valid beats missing", 1, Result{Attempt: 1, Dist: 100}, Result{}, 1},
		{"missing loses", 1, Result{}, Result{Attempt: 1, Dist: 100}, -1},
		{"neither", 1, Result{}, Result{}, 0},
		{"distance tie", 1, Result{Attempt: 1, Dist: 2}, Result{Attempt: 1, Dist: 2}, 0},
		{"farther", 1, Result{Attempt: 1, Dist: 3}, Result{Attempt: 1, Dist: 2}, -1},
		{"correct country", 2, Result{Attempt: 1, CC: "XX", Time: 20}, Result{Attempt: 1, CC: "GB", Time: 10}, 1},
		{"wrong countries", 2, Result{Attempt: 1, CC: "GB"}, Result{Attempt: 1, CC: "FR"}, 0},
		{"equal country time", 2, Result{Attempt: 1, CC: "XX", Time: 10}, Result{Attempt: 1, CC: "XX", Time: 10}, 0},
		{"faster country", 2, Result{Attempt: 1, CC: "XX", Time: 5}, Result{Attempt: 1, CC: "XX", Time: 10}, 1},
		{"slower country", 2, Result{Attempt: 1, CC: "XX", Time: 15}, Result{Attempt: 1, CC: "XX", Time: 10}, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := duelWinner(tc.mode, &tc.s, &tc.t); got != tc.want {
				t.Fatalf("got %d want %d", got, tc.want)
			}
		})
	}
}

func TestScoringStacksAndAppliesExactlyOnce(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	for _, v := range []struct {
		source, id string
		kind       int
		target     string
	}{{"a", "double", 0, ""}, {"a", "ab", 1, "b"}, {"b", "ba", 1, "a"}, {"c", "ca", 1, "a"}} {
		if err := activate(l, v.source, choice(v.id, v.kind, v.target)); err != nil {
			t.Fatal(err)
		}
	}
	l.setupRoundLocked(logic.Coords{}, "")
	l.endResults[2]["a"] = &Result{BaseScore: 2000, PlaceScore: 200, Attempt: 1, Dist: 1}
	l.endResults[2]["b"] = &Result{Attempt: 1, Dist: 2}
	l.endResults[2]["c"] = &Result{}
	_ = l.processPowerups()
	_ = l.processPowerups()
	_ = l.processTotal()
	a, b, c := l.endResults[2]["a"], l.endResults[2]["b"], l.endResults[2]["c"]
	if a.DoubleScore != 2000 || a.DuelScore != 3000 || a.Total != 7200 || b.DuelScore != -2000 || c.DuelScore != -1000 {
		t.Fatalf("wrong stacking: a=%+v b=%+v c=%+v", a, b, c)
	}
	for _, r := range l.powerEffects {
		if r.Status != "APPLIED" {
			t.Fatal("effect not finalized")
		}
	}
}

func TestDoubleNoGuessAddsZero(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	if err := activate(l, "a", choice("double", 0, "")); err != nil {
		t.Fatal(err)
	}
	l.setupRoundLocked(logic.Coords{}, "")
	_ = l.processPowerups()
	if l.endResults[2]["a"].DoubleScore != 0 || l.powerEffects[0].Status != "APPLIED" {
		t.Fatal("no-guess double incorrect")
	}
}

func TestRapidDoneActivationOrdering(t *testing.T) {
	for i := 0; i < 30; i++ {
		l := powerLobby(t)
		openChoices(l)
		var wg sync.WaitGroup
		for _, r := range []ClientReq{readyChoice("ready"), choice("double", 0, "")} {
			wg.Add(1)
			go func(r ClientReq) { defer wg.Done(); l.mu.Lock(); defer l.mu.Unlock(); _ = activate(l, "a", r) }(r)
		}
		wg.Wait()
		if len(l.powerEffects) > 1 {
			t.Fatal("more than one activation")
		}
		if l.players["a"].Powerups[0] != (len(l.powerEffects) == 0) {
			t.Fatal("atomic spend violated")
		}
	}
}

func TestStateIsDetachedAndPublicProjectionOmitsIt(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	if err := activate(l, "a", choice("double", 0, "")); err != nil {
		t.Fatal(err)
	}
	s := l.snapshotLocked()
	s.PowerupState.Effects[0].Status = "CORRUPTED"
	s.PowerupState.DecisionReady["a"] = true
	s.Players["a"].Powerups[1] = false
	if l.powerEffects[0].Status != "QUEUED" || l.powerWindow.done[1] || !l.players["a"].Powerups[1] {
		t.Fatal("snapshot aliases live state")
	}
	if l.publicSnapshotLocked().PowerupState != nil {
		t.Fatal("public projection exposed member powerup authority")
	}
}

func TestRapidTransitionsFenceOldActivationsAndTimers(t *testing.T) {
	l := powerLobby(t)
	l.conf.NumRounds = 6
	l.resetResultsLocked()
	l.gameID = "match"
	l.setupRoundLocked(logic.Coords{}, "")
	var e lobbyEffects
	for round := 1; round < 6; round++ {
		l.roundGen = uint64(round)
		l.finalizeLocked(&e, l.roundGen)
		if !l.powerWindow.open {
			t.Fatal("expected chooser window")
		}
		generation := l.powerWindow.generation
		r := choice(fmt.Sprintf("round-%d", round), 0, "")
		r.Round = &round
		if round == 1 {
			if err := activate(l, "a", r); err != nil {
				t.Fatal(err)
			}
		}
		l.setupRoundLocked(logic.Coords{}, "")
		before := len(l.powerEffects)
		if round == 1 {
			if err := activate(l, "a", r); err != nil {
				t.Fatal(err)
			}
		} else {
			requireCode(t, activate(l, "a", r), "STALE_POWERUP_CONTEXT")
		}
		l.powerupExpired(generation)
		if len(l.powerEffects) != before || l.phase != PhasePlaying || l.powerWindow.open || l.round != round+1 {
			t.Fatal("old request/timer mutated advanced round")
		}
	}
}

func TestConcurrentDuplicateRequestsSpendOnlyOnce(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	r := choice("same-request", 0, "")
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			l.mu.Lock()
			defer l.mu.Unlock()
			if err := activate(l, "a", r); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if len(l.powerEffects) != 1 || len(l.powerReceipts) != 1 || l.players["a"].Powerups[0] {
		t.Fatal("concurrent retry double-spent or duplicated receipt")
	}
}

func TestReceiptAndLedgerLimitsDoNotSpend(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	l.powerEffects = make([]powerupRecord, 2*l.conf.MaxPlayers*l.conf.NumRounds)
	requireCode(t, activate(l, "a", choice("limit", 0, "")), "POWERUP_LIMIT")
	if !l.players["a"].Powerups[0] {
		t.Fatal("ledger cap spent inventory")
	}
	l.powerEffects = nil
	for i := 0; i < 3*l.conf.MaxPlayers*l.conf.NumRounds; i++ {
		l.powerReceipts[powerupReceiptKey{member: 999, request: fmt.Sprint(i)}] = powerupReceipt{}
	}
	requireCode(t, activate(l, "a", readyChoice("limit-ready")), "POWERUP_LIMIT")
	if l.powerWindow.done[1] {
		t.Fatal("receipt cap locked player")
	}
}
