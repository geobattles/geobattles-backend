package game

import (
	"errors"
	"sync"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
)

var registry = struct {
	sync.Mutex
	lobbies map[string]*Lobby
}{lobbies: make(map[string]*Lobby)}

// Lookup copies the pointer and releases the registry BEFORE any lobby lock.
func lookupLobby(id string) *Lobby {
	registry.Lock()
	l := registry.lobbies[id]
	registry.Unlock()
	return l
}

func deleteLobbyIfSame(l *Lobby) {
	registry.Lock()
	if registry.lobbies[l.id] == l {
		delete(registry.lobbies, l.id)
	}
	registry.Unlock()
}

// ListLobbies preserves the ID-keyed HTTP shape, not live registry/state pointers.
// It is a set of individually consistent snapshots, not a global transaction.
func ListLobbies() map[string]LobbySnapshot {
	registry.Lock()
	lobbies := make([]*Lobby, 0, len(registry.lobbies))
	for _, l := range registry.lobbies {
		lobbies = append(lobbies, l)
	}
	registry.Unlock()
	out := make(map[string]LobbySnapshot, len(lobbies))
	for _, l := range lobbies {
		l.mu.Lock()
		if !l.closed && !l.idleExpiredLocked(time.Now()) {
			out[l.id] = l.publicSnapshotLocked()
		}
		l.mu.Unlock()
	}
	return out
}

func CreateLobby(conf LobbyConf, creator string) (LobbySnapshot, error) {
	c, size, err := normalizeConf(conf)
	if err != nil {
		return LobbySnapshot{}, err
	}
	l := &Lobby{creator: creator, hub: websocket.NewHub(), players: make(map[string]*Player),
		leases: make(map[string]*playerLease), reservations: make(map[string]*Admission),
		ccSize: size, phase: PhaseWaiting}
	l.conf = &c
	l.resetResultsLocked()
	// Generate outside the short registry critical section; insertion is conditional.
	// Bound collision retries rather than replacing an existing lobby or spinning.
	for attempt := 0; attempt < 128; attempt++ {
		id := logic.GenerateRndID(4)
		l.id = id
		if conf.Name == "" {
			l.conf.Name = id
		}
		registry.Lock()
		_, exists := registry.lobbies[id]
		if !exists {
			registry.lobbies[id] = l
		}
		registry.Unlock()
		if !exists {
			l.mu.Lock()
			snapshot := l.publicSnapshotLocked()
			// A never-joined lobby should not retain an entry forever.
			if len(l.players) == 0 && !l.closed {
				l.emptyTimer = time.AfterFunc(reconnectGrace, l.expireEmpty)
			}
			l.mu.Unlock()
			return snapshot, nil
		}
	}
	l.hub.Close()
	return LobbySnapshot{}, errors.New("LOBBY_ID_UNAVAILABLE")
}
