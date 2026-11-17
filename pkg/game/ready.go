package game

import "errors"

// Ready belongs to a retained membership, not an optimistic client toggle.
// Initial/replay selection freezes it; ordinary next-round starts never gate it.
func (l *Lobby) setReadyLocked(id string, ready *bool) error {
	if ready == nil {
		return errors.New("INVALID_READY")
	}
	if l.starting {
		return errors.New("ROUND_STARTING")
	}
	if l.phase != PhaseWaiting && l.phase != PhaseFinished {
		return errors.New("GAME_IN_PROGRESS")
	}
	p := l.players[id]
	if p == nil || !p.Connected {
		return errors.New("PLAYER_NOT_IN_LOBBY")
	}
	p.Ready = *ready
	return nil
}

func (l *Lobby) resetReadyLocked() {
	for _, p := range l.players {
		p.Ready = false
	}
}

func (l *Lobby) allReadyLocked() bool {
	if len(l.players) == 0 {
		return false
	}
	for id, p := range l.players {
		lease := l.leases[id]
		if !p.Connected || !p.Ready || lease == nil || lease.client == nil || lease.client.Closed() {
			return false
		}
	}
	return true
}
