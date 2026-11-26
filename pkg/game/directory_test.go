package game

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func directoryRoom() *Lobby {
	return &Lobby{
		id: "DEMO", phase: PhaseWaiting, admin: "host",
		conf:    &LobbyConf{Name: "Alpine Atlas", MaxPlayers: 2, NumRounds: 20},
		players: map[string]*Player{"host": {Name: "Atlas", Connected: true}},
		leases:  map[string]*playerLease{"host": {}}, reservations: map[string]*Admission{},
	}
}

func TestDirectorySeats(t *testing.T) {
	l := directoryRoom()
	l.reservations["guest"] = &Admission{}
	row, _ := l.summary("")
	if row.OpenToNew || row.Connected != 1 || row.Occupied != 2 || row.Retained {
		t.Fatalf("pending new reservation: %+v", row)
	}
	delete(l.reservations, "guest")
	lease := &playerLease{expires: time.Now().Add(time.Minute)}
	l.players["guest"], l.leases["guest"] = &Player{Name: "Guest"}, lease
	l.reservations["guest"] = &Admission{previous: lease}
	row, _ = l.summary("guest")
	if !row.Retained || row.OpenToNew || row.Occupied != 2 || row.Connected != 1 || row.HostName != "Atlas" {
		t.Fatalf("returning member: %+v", row)
	}
	delete(l.reservations, "guest")
	lease.expires = time.Now().Add(-time.Second)
	row, ok := l.summary("guest")
	if !ok || row.Retained || l.admissionStatusLocked("guest", time.Now(), false).Code != "REJOIN_EXPIRED" {
		t.Fatal("expired seat in a live waiting room")
	}
	l.phase = PhasePlaying
	row, _ = l.summary("guest")
	if !row.Retained || row.OpenToNew {
		t.Fatal("away participant in an active match lost the return seat")
	}
	l.phase = PhaseFinished
	delete(l.players, "guest")
	delete(l.leases, "guest")
	row, _ = l.summary("guest")
	if row.Retained || !row.OpenToNew {
		t.Fatal("fresh admission and retained membership must be distinct")
	}
	l.starting = true
	row, _ = l.summary("")
	if row.OpenToNew || !row.Starting {
		t.Fatal("new admission during round start")
	}
	l.idleDeadline = time.Now().Add(-time.Second)
	if _, ok := l.summary("host"); ok {
		t.Fatal("expired room is still discoverable")
	}
	if _, ok := FindLobbySummary("never-a-room-code"); ok {
		t.Fatal("unknown room found")
	}
}

func TestDirectorySizeIndependentOfHistory(t *testing.T) {
	l := directoryRoom()
	l.resetResultsLocked()
	encode := func(value any) []byte {
		t.Helper()
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	directory := func() []byte {
		row, _ := l.summary("")
		return encode(map[string]LobbySummary{l.id: row})
	}
	legacy := func() []byte { return encode(map[string]LobbySnapshot{l.id: l.publicSnapshotLocked()}) }
	empty, legacyEmpty := directory(), legacy()
	for round := 1; round <= 20; round++ {
		l.rawResults[round] = map[string][]Result{"host": {{Attempt: 1, BaseScore: 5000}}, "guest": {{Attempt: 1}}}
		l.endResults[round] = map[string]*Result{"host": {Total: 5000}, "guest": {}}
	}
	full := directory()
	if !bytes.Equal(empty, full) {
		t.Fatal("match history changed the directory payload")
	}
	if !bytes.Equal(legacyEmpty, legacy()) {
		t.Fatal("match history changed the legacy lobby list payload")
	}
}
