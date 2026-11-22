package game

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
	ws "github.com/gorilla/websocket"
)

// Real loopback WebSocket pair; no app startup, auth/DB, or provider required.
func powerSocket(t *testing.T, l *Lobby, id string) (*websocket.Client, *ws.Conn) {
	t.Helper()
	serverConn := make(chan *ws.Conn, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := (&ws.Upgrader{}).Upgrade(w, r, nil)
		if err == nil {
			serverConn <- c
		}
	}))
	peer, _, err := ws.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	conn := <-serverConn
	c := websocket.NewClient(id, id, l.hub, conn, l, PlayerMessageHandler, nil)
	l.leases[id].client = c
	l.hub.RegisterClient(c)
	go c.Write()
	t.Cleanup(func() { c.Close(); peer.Close(); srv.Close() })
	return c, peer
}

func nextPowerEvent(t *testing.T, peer *ws.Conn) modelsEvent {
	t.Helper()
	_ = peer.SetReadDeadline(time.Now().Add(3 * time.Second))
	var event modelsEvent
	if err := peer.ReadJSON(&event); err != nil {
		t.Fatal(err)
	}
	return event
}

type modelsEvent struct {
	Status  string          `json:"status"`
	Type    string          `json:"type"`
	Seq     uint64          `json:"seq"`
	Payload json.RawMessage `json:"payload"`
}

func loadPowerGeography(t *testing.T) {
	t.Helper()
	t.Chdir("../..")
	if err := logic.InitCountryDB(); err != nil {
		t.Fatal(err)
	}
}

func TestMalformedPowerupWireErrorCorrelatesRequest(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	c, peer := powerSocket(t, l, "a")
	PlayerMessageHandler(c, []byte(`{"command":"use_powerup","gameId":"match","round":1.5,"requestId":"typed-id","powerup":{"type":0}}`))
	event := nextPowerEvent(t, peer)
	var p struct {
		Code      string `json:"code"`
		RequestID string `json:"requestId"`
		GameID    string `json:"gameId"`
	}
	if err := json.Unmarshal(event.Payload, &p); err != nil {
		t.Fatal(err)
	}
	if event.Type != "REQUEST_ERROR" || p.Code != "INVALID_REQUEST" || p.RequestID != "typed-id" || p.GameID != "match" {
		t.Fatalf("uncorrelated safe error: %s", event.Payload)
	}
	if !l.players["a"].Powerups[0] {
		t.Fatal("malformed request spent inventory")
	}
}

func TestClosedRecoveryRestoresDoneFlag(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	if err := activate(l, "a", readyChoice("done")); err != nil {
		t.Fatal(err)
	}
	var e lobbyEffects
	l.players["a"].Connected = false
	l.reconcilePowerupWindowLocked(&e)
	l.powerWindow.deadline = time.Now()
	l.reconcilePowerupWindowLocked(&e)
	l.players["a"].Connected = true
	l.reconcilePowerupWindowLocked(&e)
	done, ok := l.resyncLocked().Lobby.PowerupState.DecisionReady["a"]
	if !ok || !done || l.powerWindow.open {
		t.Fatal("closed recovery forgot authoritative done state")
	}
}

func TestAdmissionAtCutoffDoesNotStrandQueuedStart(t *testing.T) {
	t.Setenv("GMAPS_API", "")
	l := powerLobby(t)
	openChoices(l)
	_, hostPeer := powerSocket(t, l, "a")
	var e lobbyEffects
	if err := l.requestPowerupStartLocked(&e, l.leases["a"].client, choice("", 0, "")); err != nil {
		t.Fatal(err)
	}
	// A retained player reconnects while the deadline timer is delayed.
	l.players["b"].Connected = false
	previous := l.leases["b"]
	previous.expires = time.Now().Add(time.Minute)
	previous.grace = time.AfterFunc(time.Minute, func() {})
	l.reconcilePowerupWindowLocked(&e)
	l.powerWindow.timer.Stop()
	l.powerWindow.deadline = time.Now()
	a := &Admission{lobby: l, id: "b", name: "b", generation: 99, previous: previous, deadline: time.Now().Add(time.Minute)}
	a.timer = time.AfterFunc(time.Minute, func() {})
	l.reservations["b"] = a
	joined := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&ws.Upgrader{}).Upgrade(w, r, nil)
		if err == nil {
			a.Join(conn)
			close(joined)
		}
	}))
	peer, _, err := ws.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	<-joined
	l.mu.Lock()
	stranded := l.powerStart != nil && !l.starting
	l.mu.Unlock()
	for {
		event := nextPowerEvent(t, hostPeer)
		if event.Type == "REQUEST_ERROR" {
			break
		}
	}
	// Use the ordinary close path and stop the recovery timer it creates.
	peer.Close()
	srv.Close()
	l.mu.Lock()
	c := l.leases["b"].client
	l.mu.Unlock()
	c.Close()
	l.mu.Lock()
	if lease := l.leases["b"]; lease != nil && lease.grace != nil {
		lease.grace.Stop()
	}
	l.mu.Unlock()
	if stranded {
		t.Fatal("admission closed/stopped window but never launched or cancelled queued host start")
	}
}

func TestHostStartWaitsForOtherChoicesThenReservesOnce(t *testing.T) {
	loadPowerGeography(t)
	l := powerLobby(t)
	openChoices(l)
	var e lobbyEffects
	host := l.leases["a"].client
	if err := l.requestPowerupStartLocked(&e, host, choice("", 0, "")); err != nil {
		t.Fatal(err)
	}
	if l.starting || e.selection != nil || l.powerStart != host || !l.powerWindow.open {
		t.Fatal("host skipped chooser barrier")
	}
	if err := activate(l, "b", choice("double", 0, "")); err != nil {
		t.Fatal(err)
	}
	if err := activate(l, "b", readyChoice("done-b")); err != nil {
		t.Fatal(err)
	}
	if err := l.powerupCommandLocked(&e, l.leases["c"].client, readyChoice("done-c")); err != nil {
		t.Fatal(err)
	}
	if !l.starting || e.selection == nil || l.powerWindow.open {
		t.Fatal("last decision did not reserve queued start")
	}
	generation := l.generation
	var retry lobbyEffects
	if err := l.requestPowerupStartLocked(&retry, host, choice("", 0, "")); err != nil {
		t.Fatal(err)
	}
	if retry.selection != nil || l.generation != generation {
		t.Fatal("duplicate start reserved provider work twice")
	}
	l.cancelStartLocked(&e)
	if l.powerStart != nil || l.starting || l.powerWindow.open {
		t.Fatal("cancelled selection reopened window or retained intent")
	}
}

func TestProviderFailureRetainsEffectsAndClosedRetry(t *testing.T) {
	loadPowerGeography(t)
	t.Setenv("GMAPS_API", "")
	l := powerLobby(t)
	openChoices(l)
	powerSocket(t, l, "a")
	if err := activate(l, "b", choice("double", 0, "")); err != nil {
		t.Fatal(err)
	}
	var e lobbyEffects
	l.powerWindow.deadline = time.Now()
	l.reconcilePowerupWindowLocked(&e)
	if err := l.requestPowerupStartLocked(&e, l.leases["a"].client, choice("", 0, "")); err != nil || e.selection == nil {
		t.Fatalf("reserve: %v", err)
	}
	l.completeStart(e.selection)
	if l.starting || l.powerStart != nil || l.powerWindow.open || l.powerEffects[0].Status != "QUEUED" || l.players["b"].Powerups[0] {
		t.Fatal("provider failure changed choices or retained start intent")
	}
	var retry lobbyEffects
	if err := l.requestPowerupStartLocked(&retry, l.leases["a"].client, choice("", 0, "")); err != nil || retry.selection == nil {
		t.Fatalf("closed-window retry failed: %v", err)
	}
}

func TestWindowExpiryWithoutIntentAndOldTimers(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	old := l.powerWindow.generation
	l.powerWindow.deadline = time.Now()
	l.powerupExpired(old)
	if l.powerWindow.open || l.starting {
		t.Fatal("expiry auto-started without host")
	}
	l.round = 2
	l.openPowerupWindowLocked()
	l.powerupExpired(old)
	if !l.powerWindow.open {
		t.Fatal("old timer closed newer round")
	}
	l.stopPowerupWindowLocked()
	l.closed = true
	l.powerupExpired(l.powerWindow.generation)
	if l.starting {
		t.Fatal("disposed lobby started")
	}
}

func TestInventoryGrantReplayResetAndFinalRoundSettlement(t *testing.T) {
	l := powerLobby(t)
	l.resetResultsLocked()
	l.gameID = "new"
	(*l.conf.Powerups)[1] = false
	l.setupRoundLocked(logic.Coords{}, "")
	if !l.players["a"].Powerups[0] || l.players["a"].Powerups[1] {
		t.Fatal("enabled inventory not granted")
	}
	l.players["a"].Powerups[0] = false
	l.setupRoundLocked(logic.Coords{}, "")
	if l.players["a"].Powerups[0] {
		t.Fatal("inventory regenerated per round")
	}
	l.players["a"].Powerups[0] = true
	l.phase = PhaseMidRound
	l.openPowerupWindowLocked()
	r := choice("final-double", 0, "")
	g, round := "new", 2
	r.GameID = &g
	r.Round = &round
	if err := activate(l, "a", r); err != nil {
		t.Fatal(err)
	}
	l.setupRoundLocked(logic.Coords{}, "")
	l.roundGen = 55
	l.endResults[3]["a"] = &Result{BaseScore: 2000, Attempt: 1, Dist: 1}
	var e lobbyEffects
	l.finalizeLocked(&e, 55)
	total := l.totalResults["a"].Total
	l.finalizeLocked(&e, 55)
	if l.phase != PhaseFinished || l.powerEffects[0].Status != "APPLIED" || l.endResults[3]["a"].DoubleScore != 2000 || total != l.totalResults["a"].Total {
		t.Fatal("final round did not settle once before finish")
	}
	l.resetResultsLocked()
	l.gameID = "replay"
	l.setupRoundLocked(logic.Coords{}, "")
	if len(l.powerEffects) != 0 || len(l.powerReceipts) != 0 || l.powerWindow != nil || !l.players["a"].Powerups[0] {
		t.Fatal("successful replay did not reset inventory/ledger/window")
	}
	requireCode(t, activate(l, "a", r), "STALE_POWERUP_CONTEXT")
}

func TestNoUsableItemsClosesImmediately(t *testing.T) {
	l := powerLobby(t)
	for _, p := range l.players {
		p.Powerups = []bool{false, false}
	}
	openChoices(l)
	if l.powerWindow.open || l.powerWindow.timer != nil || l.starting {
		t.Fatal("empty inventory introduced wait or auto-start")
	}
}

func TestFailedReplayPreservesFinishedPowerups(t *testing.T) {
	loadPowerGeography(t)
	t.Setenv("GMAPS_API", "")
	l := powerLobby(t)
	openChoices(l)
	powerSocket(t, l, "a")
	if err := activate(l, "b", choice("double", 0, "")); err != nil {
		t.Fatal(err)
	}
	l.phase = PhaseFinished
	l.stopPowerupWindowLocked()
	for _, p := range l.players {
		p.Ready = true
	}
	work, err := l.beginStartLocked(l.leases["a"].client)
	if err != nil {
		t.Fatal(err)
	}
	l.completeStart(work)
	if l.gameID != "match" || len(l.powerEffects) != 1 || len(l.powerReceipts) != 1 || l.players["b"].Powerups[0] || l.phase != PhaseFinished {
		t.Fatal("failed replay discarded finished powerup state")
	}
}

func TestPostSettlementLeaveCannotRefundOrReverse(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	if err := activate(l, "a", choice("duel", 1, "b")); err != nil {
		t.Fatal(err)
	}
	l.setupRoundLocked(logic.Coords{}, "")
	l.endResults[2]["a"] = &Result{Attempt: 1, Dist: 0}
	_ = l.processPowerups()
	var e lobbyEffects
	l.removePowerupMemberLocked(&e, "b")
	if l.players["a"].Powerups[1] || l.powerEffects[0].Status != "APPLIED" || l.endResults[2]["a"].DuelScore != 1000 {
		t.Fatal("leave reversed settled duel")
	}
}

func TestTransientDisconnectPreservesDuelAndOldLeaseCannotSpend(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	if err := activate(l, "a", choice("duel", 1, "b")); err != nil {
		t.Fatal(err)
	}
	l.clientLost(l.leases["b"].client)
	if lease := l.leases["b"]; lease.grace != nil {
		defer lease.grace.Stop()
	}
	if l.players["a"].Powerups[1] || l.powerEffects[0].Status != "QUEUED" {
		t.Fatal("transient loss refunded duel")
	}
	old := l.leases["a"].client
	l.leases["a"].client = websocket.NewClient("a", "a", l.hub, nil, l, PlayerMessageHandler, nil)
	var e lobbyEffects
	requireCode(t, l.powerupCommandLocked(&e, old, choice("old-lease", 0, "")), "PLAYER_NOT_IN_LOBBY")
	l.setupRoundLocked(logic.Coords{}, "")
	l.endResults[2]["b"] = &Result{Attempt: 1, Dist: 0}
	_ = l.processPowerups()
	if l.powerEffects[0].Outcome != "TARGET_WON" {
		t.Fatal("retained offline target lost valid guess")
	}
}

func TestDuelWireNotificationIsOnceAndReadyReceiptIsTyped(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	a, source := powerSocket(t, l, "a")
	_, target := powerSocket(t, l, "b")
	raw := []byte(`{"command":"use_powerup","gameId":"match","round":1,"requestId":"duel","powerup":{"type":1,"target":"b"}}`)
	PlayerMessageHandler(a, raw)
	if nextPowerEvent(t, source).Type != "POWERUP_STATE" || nextPowerEvent(t, source).Type != "POWERUP_USED" {
		t.Fatal("missing source receipt")
	}
	if nextPowerEvent(t, target).Type != "POWERUP_STATE" {
		t.Fatal("missing target state")
	}
	notice := nextPowerEvent(t, target)
	var n duelFromPayload
	_ = json.Unmarshal(notice.Payload, &n)
	if notice.Type != "DUEL_FROM" || n.User != "a" || n.Round != 2 || n.EffectID == "" {
		t.Fatal("invalid duel notice")
	}
	PlayerMessageHandler(a, raw)
	if nextPowerEvent(t, source).Type != "POWERUP_USED" {
		t.Fatal("missing retry receipt")
	}
	PlayerMessageHandler(a, []byte(`{"command":"powerup_ready","gameId":"match","round":1,"requestId":"ready"}`))
	if nextPowerEvent(t, source).Type != "POWERUP_STATE" || nextPowerEvent(t, source).Type != "POWERUP_READY" {
		t.Fatal("missing ready state/receipt")
	}
	// Target's next event must be ready state, not another DUEL_FROM on retry.
	if nextPowerEvent(t, target).Type != "POWERUP_STATE" {
		t.Fatal("retry emitted second duel notice")
	}
}

func TestInitialSelectionPublishesClosedStartingState(t *testing.T) {
	loadPowerGeography(t)
	t.Setenv("GMAPS_API", "")
	l := powerLobby(t)
	l.resetResultsLocked()
	l.gameID = ""
	l.phase = PhaseWaiting
	for _, p := range l.players {
		p.Ready = true
	}
	c, peer := powerSocket(t, l, "a")
	PlayerMessageHandler(c, []byte(`{"command":"start"}`))
	selected := nextPowerEvent(t, peer)
	var p powerupStatePayload
	_ = json.Unmarshal(selected.Payload, &p)
	if selected.Type != "POWERUP_STATE" || p.PowerupState.Status != "CLOSED" || !p.PowerupState.Starting {
		t.Fatalf("initial selection state not published: %s", selected.Type)
	}
	failed := nextPowerEvent(t, peer)
	_ = json.Unmarshal(failed.Payload, &p)
	if failed.Type != "POWERUP_STATE" || p.PowerupState.Starting {
		t.Fatal("failed initial selection retained starting state")
	}
	if nextPowerEvent(t, peer).Type != "REQUEST_ERROR" {
		t.Fatal("missing provider failure")
	}
}

func TestPowerupWireBroadcastReceiptAndRetry(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)
	c, peer := powerSocket(t, l, "a")
	raw := []byte(`{"command":"use_powerup","gameId":"match","round":1,"requestId":"double","powerup":{"type":0,"source":"b"}}`)
	PlayerMessageHandler(c, raw)
	state, receipt := nextPowerEvent(t, peer), nextPowerEvent(t, peer)
	if state.Type != "POWERUP_STATE" || receipt.Type != "POWERUP_USED" || state.Seq >= receipt.Seq {
		t.Fatal("wrong state-before-receipt ordering")
	}
	var p powerupUsedPayload
	if err := json.Unmarshal(receipt.Payload, &p); err != nil {
		t.Fatal(err)
	}
	if p.Effect.Source != "a" || p.RequestID != "double" || p.Players["a"].Powerups[0] || p.Effect.Status != "QUEUED" {
		t.Fatalf("wrong authoritative receipt: %+v", p)
	}
	PlayerMessageHandler(c, raw)
	retry := nextPowerEvent(t, peer)
	var duplicate powerupUsedPayload
	_ = json.Unmarshal(retry.Payload, &duplicate)
	if retry.Type != "POWERUP_USED" || duplicate.Effect.ID != p.Effect.ID || len(l.powerEffects) != 1 {
		t.Fatal("retry duplicated effect/broadcast")
	}
	l.mu.Lock()
	l.powerWindow.deadline = time.Now()
	l.mu.Unlock()
	PlayerMessageHandler(c, []byte(`{"command":"use_powerup","gameId":"match","round":1,"requestId":"late","powerup":{"type":1,"target":"b"}}`))
	closed, rejected := nextPowerEvent(t, peer), nextPowerEvent(t, peer)
	if closed.Type != "POWERUP_STATE" || rejected.Type != "REQUEST_ERROR" || closed.Seq >= rejected.Seq {
		t.Fatal("cutoff error preceded closing state")
	}
}
