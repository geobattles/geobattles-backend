package game

import "time"

// LobbySummary contains only the fields used to discover a room. Retained is
// specific to the authenticated caller and is omitted from public responses.
type LobbySummary struct {
	ID           string `json:"ID"`
	Name         string `json:"name"`
	HostName     string `json:"hostName,omitempty"`
	Phase        string `json:"phase"`
	CurrentRound int    `json:"currentRound"`
	NumRounds    int    `json:"numRounds"`
	Connected    int    `json:"connected"`
	Occupied     int    `json:"occupied"`
	MaxPlayers   int    `json:"maxPlayers"`
	Starting     bool   `json:"starting,omitempty"`
	OpenToNew    bool   `json:"openToNew"`
	Retained     bool   `json:"retained,omitempty"`
}

func (l *Lobby) summary(caller string) (LobbySummary, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if l.closed || l.idleExpiredLocked(now) {
		return LobbySummary{}, false
	}
	occupied := l.occupiedSeatsLocked()
	s := LobbySummary{
		ID: l.id, Name: l.conf.Name, Phase: l.phase, CurrentRound: l.round, NumRounds: l.conf.NumRounds,
		Occupied: occupied, MaxPlayers: l.conf.MaxPlayers, Starting: l.starting,
		OpenToNew: !l.starting && l.newPlayerStatusLocked(occupied) == "OK",
	}
	if host := l.players[l.admin]; host != nil {
		s.HostName = host.Name
	}
	for _, p := range l.players {
		if p.Connected {
			s.Connected++
		}
	}
	if lease := l.leases[caller]; caller != "" && lease != nil {
		s.Retained = !l.seatExpiredLocked(lease, now)
	}
	return s, true
}

func LobbyDirectory(caller string) map[string]LobbySummary {
	lobbies := registeredLobbies()
	out := make(map[string]LobbySummary, len(lobbies))
	for _, l := range lobbies {
		if row, ok := l.summary(caller); ok {
			out[row.ID] = row
		}
	}
	return out
}

func FindLobbySummary(code string) (LobbySummary, bool) {
	if l := lookupLobby(code); l != nil {
		return l.summary("")
	}
	return LobbySummary{}, false
}
