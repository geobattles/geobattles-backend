package game

import (
	"testing"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
)

func TestSpectatorSnapshotOnlyRevealsSettledRounds(t *testing.T) {
	l := powerLobby(t)
	l.locations = []logic.Coords{{Lat: 46, Lng: 15}, {Lat: 52, Lng: 5}}
	l.round, l.active = 2, true
	l.deadline = time.Now().Add(time.Minute)
	l.rawResults[2] = map[string][]Result{"a": {{Attempt: 1, BaseScore: 4321}}}
	l.endResults[2] = map[string]*Result{"a": {Attempt: 1, BaseScore: 4321}}
	l.endResults[1]["a"].Total = 5000
	l.totalResults["a"].Total = 5000
	for id, p := range l.players {
		l.rememberCompetitorLocked(id, p)
	}
	l.powerEffects = []powerupRecord{{PowerupEffect: PowerupEffect{Type: 1, Round: 2, Status: "ACTIVE", Source: "a", Target: "b"}}}
	l.powerLogs[2] = []Powerup{{Type: 1, Source: "a", Target: "b"}}

	s := l.spectatorSnapshotLocked()
	if !s.Spectating || s.Location != nil || s.PowerupState != nil || len(s.PowerLog) != 0 || len(s.Results) != 0 || len(s.FullRoundRes) != 0 || len(s.RoundRes) != 0 {
		t.Fatal("spectator received private state")
	}
	if len(s.RevealedAnswers) != 1 || len(s.EndResults) != 1 || s.RevealedAnswers[1] != l.locations[0] || s.EndResults[1]["a"].Total != 5000 ||
		s.TotalResults["a"].Total != 5000 || s.TimeRemaining <= 0 {
		t.Fatal("spectator could not recover the completed round and live clock")
	}
	for _, p := range s.Players {
		if len(p.Powerups) > 0 || p.Lives != 0 {
			t.Fatal("spectator received private player state")
		}
	}
	public := l.publicSnapshotLocked()
	if len(public.Results[2]) != 0 || len(public.EndResults[2]) != 0 || len(public.FullRoundRes) != 0 || len(public.RoundRes) != 0 || len(public.PowerLog) != 0 {
		t.Fatal("public directory exposed unsettled round data")
	}
}

func TestSpectatorAdmissionsAreSeparateFromPlayerSeats(t *testing.T) {
	l := powerLobby(t)
	l.conf.MaxPlayers = len(l.players)
	now := time.Now()
	if s := l.admissionStatusLocked("late", now, false); s.Code != "GAME_IN_PROGRESS" {
		t.Fatalf("legacy caller got a role it cannot render: %+v", s)
	}
	if s := l.admissionStatusLocked("late", now, true); s.Code != "OK" || !s.Spectating {
		t.Fatalf("late arrival could not watch a full match: %+v", s)
	}
	for i := range maxSpectators {
		l.reservations[string(rune('d'+i))] = &Admission{spectating: true}
	}
	if s := l.admissionStatusLocked("late", now, true); s.Code != "SPECTATORS_FULL" {
		t.Fatalf("watch capacity ignored outstanding handshakes: %+v", s)
	}
	if l.occupiedSeatsLocked() != len(l.players) || l.hasPlayerReservationsLocked() {
		t.Fatal("watch reservations interfered with the active roster")
	}
	l.players["b"].Connected = false
	l.leases["b"].client = nil
	if s := l.admissionStatusLocked("b", now, true); s.Code != "OK" || !s.Retained || s.Spectating {
		t.Fatalf("retained competitor lost their role: %+v", s)
	}
}

func TestRematchPromotesConnectedSpectatorsInOrder(t *testing.T) {
	l := powerLobby(t)
	l.phase = PhaseFinished
	l.spectators = map[string]string{"older": "First", "newer": "Second", "offline": "Offline"}
	for i, id := range []string{"offline", "older", "newer"} {
		l.leases[id] = &playerLease{present: uint64(4 + i)}
		if id != "offline" {
			l.leases[id].client = websocket.NewClient(id, id, l.hub, nil, l, PlayerMessageHandler, nil)
		}
	}
	l.players["a"].Ready = true
	var e lobbyEffects
	l.publishPromotionsLocked(&e)
	if p := l.players["older"]; p == nil || p.Ready || l.isSpectatorLocked("older") || !l.isSpectatorLocked("newer") ||
		l.players["offline"] != nil || !l.players["a"].Ready || l.totalResults["older"] != nil {
		t.Fatal("promotion changed readiness, scores or admission order")
	}
}

func TestEarlyFinishSettlesOnlyPlayedRounds(t *testing.T) {
	for _, phase := range []string{PhasePlaying, PhaseMidRound} {
		t.Run(phase, func(t *testing.T) {
			l := powerLobby(t)
			l.phase, l.active, l.roundGen = phase, phase == PhasePlaying, 2
			l.locations = []logic.Coords{{Lat: 46, Lng: 15}}
			l.endResults[1]["a"] = &Result{Attempt: 1, BaseScore: 1234, Total: 1234}
			if !l.active {
				l.finalizedGen = l.roundGen
				l.totalResults["a"].Total = 1234
				l.openPowerupWindowLocked()
				l.powerEffects = append(l.powerEffects, powerupRecord{PowerupEffect: PowerupEffect{Round: 2, Status: "QUEUED"}})
			}
			var e lobbyEffects
			l.endMatchEarlyLocked(&e)
			if l.phase != PhaseFinished || !l.endedEarly || l.active || l.conf.NumRounds != 3 || l.totalResults["a"].Total != 1234 ||
				e.save == nil || len(e.save.Rounds) != 1 || len(l.snapshotLocked().RevealedAnswers) != 1 {
				t.Fatal("early finish changed the planned length, scored twice, or lost the played round")
			}
			if len(l.powerEffects) > 0 && l.powerEffects[0].Status != "CANCELLED" {
				t.Fatal("unplayed round kept a queued powerup")
			}
			var stale lobbyEffects
			l.finalizeLocked(&stale, l.roundGen)
			if stale.save != nil || l.totalResults["a"].Total != 1234 {
				t.Fatal("stale round timer settled an already completed match")
			}
		})
	}
}
