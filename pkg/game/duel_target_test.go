package game

import (
	"testing"

	"github.com/geobattles/geobattles-backend/pkg/logic"
)

func TestDuelUsesExplicitOpponentWithoutSubstitution(t *testing.T) {
	l := powerLobby(t)
	openChoices(l)

	// Invalid confirmation cannot spend inventory or create a default Duel.
	for _, target := range []string{"", "a", "missing"} {
		requireCode(t, activate(l, "a", choice("invalid-"+target, 1, target)), "WRONG_TARGET")
		if !l.players["a"].Powerups[1] || len(l.powerEffects) != 0 || len(l.powerLogs[2]) != 0 || len(l.powerReceipts) != 0 {
			t.Fatal("invalid target consumed Duel or created an effect/receipt")
		}
	}
	l.players["c"].Connected = false
	requireCode(t, activate(l, "a", choice("disconnected", 1, "c")), "WRONG_TARGET")
	if !l.players["a"].Powerups[1] || len(l.powerEffects) != 0 {
		t.Fatal("disconnected target spent Duel")
	}
	l.players["c"].Connected = true

	selected := choice("selected-c", 1, "c")
	if err := activate(l, "a", selected); err != nil {
		t.Fatal(err)
	}
	if err := activate(l, "a", selected); err != nil {
		t.Fatal(err)
	}
	requireCode(t, activate(l, "a", choice("selected-c", 1, "b")), "REQUEST_ID_CONFLICT")
	requireCode(t, activate(l, "a", choice("second-click", 1, "b")), "NOT_AVAILABLE")
	if l.players["a"].Powerups[1] || !l.players["a"].Powerups[0] || len(l.powerEffects) != 1 || len(l.powerReceipts) != 1 || len(l.powerLogs[2]) != 1 {
		t.Fatal("confirmation/retry did not consume only one Duel")
	}
	if l.powerEffects[0].Target != "c" || l.powerLogs[2][0].Target != "c" {
		t.Fatal("selected opponent was replaced")
	}

	l.setupRoundLocked(logic.Coords{}, "")
	// A beats C, but loses to B: substituting B would reverse the outcome.
	l.endResults[2]["a"] = &Result{Attempt: 1, Dist: 10}
	l.endResults[2]["b"] = &Result{Attempt: 1, Dist: 1}
	l.endResults[2]["c"] = &Result{Attempt: 1, Dist: 20}
	for range 2 {
		if err := l.processPowerups(); err != nil {
			t.Fatal(err)
		}
	}
	if l.endResults[2]["a"].DuelScore != 1000 || l.endResults[2]["b"].DuelScore != 0 || l.endResults[2]["c"].DuelScore != -1000 {
		t.Fatal("Duel settled against another opponent or applied more than once")
	}
	if l.powerEffects[0].Status != "APPLIED" || l.powerEffects[0].Outcome != "SOURCE_WON" || l.powerEffects[0].Target != "c" {
		t.Fatal("final effect did not preserve the confirmed opponent")
	}
}
