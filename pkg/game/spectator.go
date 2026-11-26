package game

import (
	"time"

	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/models"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
	ws "github.com/gorilla/websocket"
)

const maxSpectators = 12

func (l *Lobby) matchInProgressLocked() bool {
	return l.phase == PhasePlaying || l.phase == PhaseMidRound
}

func (l *Lobby) isSpectatorLocked(id string) bool {
	_, watching := l.spectators[id]
	return watching
}

func (l *Lobby) occupiedSpectatorsLocked() int {
	occupied := len(l.spectators)
	for _, a := range l.reservations {
		if a.spectating && a.previous == nil {
			occupied++
		}
	}
	return occupied
}

func (l *Lobby) hasPlayerReservationsLocked() bool {
	for _, a := range l.reservations {
		if !a.spectating {
			return true
		}
	}
	return false
}

func (l *Lobby) canOfferSpectatingLocked() bool {
	return l.matchInProgressLocked() || (l.phase == PhaseFinished && l.occupiedSeatsLocked() >= l.conf.MaxPlayers)
}

func (l *Lobby) spectatorCountLocked() int {
	count := 0
	for id := range l.spectators {
		if l.liveLeaseLocked(id) {
			count++
		}
	}
	return count
}

func (l *Lobby) joinSpectatorLocked(e *lobbyEffects, a *Admission, c *websocket.Client) {
	delete(l.reservations, a.id)
	a.timer.Stop()
	present := uint64(0)
	if a.previous != nil {
		a.previous.stopTimers()
		present = a.previous.present
	} else {
		l.nextMember++
		present = l.nextMember
	}
	l.spectators[a.id] = a.name
	l.leases[a.id] = &playerLease{client: c, generation: a.generation, present: present}
	if !l.hub.RegisterClient(c) {
		e.close(c)
		return
	}
	l.sendFrameLocked(e, c, models.ResponseBase{
		Status: "OK", Type: "JOINED_LOBBY", Payload: models.ResponsePayload{User: a.id, Lobby: l.spectatorSnapshotLocked()},
	})
	l.publishPromotionsLocked(e)
	l.publishSpectatorsLocked(e)
}

func (l *Lobby) removeSpectatorLocked(id string) {
	l.leases[id].stopTimers()
	if pending := l.reservations[id]; pending != nil {
		pending.timer.Stop()
		delete(l.reservations, id)
	}
	delete(l.spectators, id)
	delete(l.leases, id)
}

func (l *Lobby) leaveSpectatorLocked(e *lobbyEffects, c *websocket.Client, explicit bool) {
	l.hub.UnregisterClient(c)
	if explicit {
		l.removeSpectatorLocked(c.ID)
		e.closeWith(c, ws.CloseNormalClosure, "Left lobby")
	} else {
		lease := l.leases[c.ID]
		lease.client = nil
		lease.expires = time.Now().Add(returnWindow)
		l.scheduleSeatExpiryLocked(c.ID, lease)
	}
	l.publishSpectatorsLocked(e)
}

// Give available rematch seats to connected watchers in admission order. A
// pending upgrade or an offline player retains their existing reservation.
func (l *Lobby) publishPromotionsLocked(e *lobbyEffects) {
	if l.starting || l.phase != PhaseFinished {
		return
	}
	promoted := false
	for l.occupiedSeatsLocked() < l.conf.MaxPlayers {
		oldest := ""
		for id := range l.spectators {
			if l.liveLeaseLocked(id) && (oldest == "" || l.leases[id].present < l.leases[oldest].present) {
				oldest = id
			}
		}
		if oldest == "" {
			break
		}
		l.players[oldest] = &Player{Name: l.spectators[oldest], Connected: true, Color: l.playerColorLocked(), RematchSeat: true}
		delete(l.spectators, oldest)
		promoted = true
	}
	if promoted {
		l.selectAdminLocked()
		if l.admin != "" {
			l.clearIdleLocked()
		}
		l.broadcastLocked(e, models.ResponseBase{Status: "OK", Type: "UPDATED_LOBBY", Payload: models.ResponsePayload{Lobby: l.snapshotLocked()}})
	}
}

// Construct the public match view from allowed fields. It has no current
// answer, raw attempts, inventories, queued actions, or unscored rows.
func (l *Lobby) spectatorSnapshotLocked() LobbySnapshot {
	lastRevealed := l.round
	if l.phase == PhasePlaying {
		lastRevealed--
	}
	out := LobbySnapshot{
		ID: l.id, GameID: l.gameID, Admin: l.admin, Spectating: true, SpectatorCount: l.spectatorCountLocked(), EndedEarly: l.endedEarly,
		Conf:         LobbyConf{Name: l.conf.Name, Mode: l.conf.Mode, CCList: []string{}, NumRounds: l.conf.NumRounds, MaxPlayers: l.conf.MaxPlayers, RoundTime: l.conf.RoundTime},
		CurrentRound: l.round, Round: l.round, Phase: l.phase, TimeRemaining: l.remainingLocked(),
		Players: make(map[string]Player), TotalResults: make(map[string]Result),
		Results: make(map[int]map[string][]Result), EndResults: make(map[int]map[string]Result), RevealedAnswers: make(map[int]logic.Coords),
		FullRoundRes: make(map[string][]Result), RoundRes: make(map[string]Result),
	}
	for id, identity := range l.matchRoster {
		player := Player{Name: identity.Name, Color: identity.Color}
		if p := l.players[id]; p != nil && p.Connected && l.liveLeaseLocked(id) {
			out.NumPlayers++
			player.Connected = true
		}
		out.Players[id] = player
	}
	for round, rows := range l.endResults {
		if round > 0 && round <= lastRevealed && round <= len(l.locations) {
			out.EndResults[round] = copyResults(rows)
			out.RevealedAnswers[round] = l.locations[round-1]
			for id := range rows {
				if result := l.totalResults[id]; result != nil {
					out.TotalResults[id] = Result{Total: result.Total}
				}
			}
		}
	}
	return out
}

func (l *Lobby) publishSpectatorsLocked(e *lobbyEffects) {
	if l.closed || l.spectatorCountLocked() == 0 {
		return
	}
	l.broadcastFrameLocked(e, models.ResponseBase{
		Status: "OK", Type: "UPDATED_LOBBY", Payload: models.ResponsePayload{Lobby: l.spectatorSnapshotLocked()},
	}, func(c *websocket.Client) bool { return l.isSpectatorLocked(c.ID) })
}
