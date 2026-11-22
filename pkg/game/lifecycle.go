package game

import (
	"errors"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/models"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
	ws "github.com/gorilla/websocket"
)

const (
	reconnectGrace = 30 * time.Second
	admissionWait  = 10 * time.Second
)

// Admission is an exact single-use upgrade reservation. Identity/capacity are
// reserved BEFORE Upgrade; no lobby lock spans the HTTP/WebSocket handshake.
type Admission struct {
	lobby      *Lobby
	id         string
	name       string
	generation uint64
	previous   *playerLease
	deadline   time.Time
	timer      *time.Timer // owned by lobby.mu
}

func ReserveAdmission(lobbyID, id, name string) (*Admission, error) {
	l := lookupLobby(lobbyID)
	if l == nil {
		return nil, errors.New("LOBBY_NOT_FOUND")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil, errors.New("LOBBY_NOT_FOUND")
	}
	if l.starting {
		return nil, errors.New("ROUND_STARTING")
	}
	if l.reservations[id] != nil || (l.players[id] != nil && l.players[id].Connected) {
		return nil, errors.New("ALREADY_CONNECTED")
	}
	now := time.Now()
	previous := l.leases[id]
	if previous != nil && !previous.expires.IsZero() && !now.Before(previous.expires) {
		return nil, errors.New("REJOIN_EXPIRED")
	}
	if l.players[id] == nil {
		if l.phase != PhaseWaiting && l.phase != PhaseFinished {
			return nil, errors.New("GAME_IN_PROGRESS")
		}
		occupied := len(l.players)
		for _, reservation := range l.reservations {
			if reservation.previous == nil {
				occupied++
			}
		}
		if occupied >= l.conf.MaxPlayers {
			return nil, errors.New("LOBBY_FULL")
		}
	}
	l.nextLease++
	a := &Admission{lobby: l, id: id, name: name, generation: l.nextLease, previous: previous, deadline: now.Add(admissionWait)}
	l.reservations[id] = a
	a.timer = time.AfterFunc(time.Until(a.deadline), a.Cancel)
	return a, nil
}

func (a *Admission) Cancel() {
	l := a.lobby
	var e lobbyEffects
	l.mu.Lock()
	if l.reservations[a.id] == a {
		delete(l.reservations, a.id)
		a.timer.Stop()
		if len(l.players) == 0 && len(l.reservations) == 0 && l.emptyTimer == nil {
			l.closeLocked(&e)
		}
	}
	l.mu.Unlock()
	l.applyEffects(e)
}

// Join finishes the reservation, publishes membership before starting the reader,
// and rolls back/close the upgraded socket if the exact reservation expired.
func (a *Admission) Join(conn *ws.Conn) {
	l := a.lobby
	c := websocket.NewClient(a.id, a.name, l.hub, conn, l, PlayerMessageHandler, l.clientLost)
	var e lobbyEffects
	l.mu.Lock()
	now := time.Now()
	if l.closed || l.starting || l.reservations[a.id] != a || l.leases[a.id] != a.previous ||
		!now.Before(a.deadline) || (a.previous != nil && !now.Before(a.previous.expires)) {
		l.mu.Unlock()
		a.Cancel()
		c.CloseWithCode(ws.CloseTryAgainLater, "Admission expired; reconnect")
		return
	}
	// Close an expired decision window before admitting recovered eligibility.
	// Recovery cannot join a choice set whose original cutoff already passed.
	l.reconcilePowerupWindowLocked(&e)
	delete(l.reservations, a.id)
	a.timer.Stop()
	if l.emptyTimer != nil {
		l.emptyTimer.Stop()
		l.emptyTimer = nil
	}
	returning := a.previous != nil
	var present uint64
	if returning {
		a.previous.grace.Stop()
		present = a.previous.present
		p := l.players[a.id]
		p.Connected, p.Name, p.Ready = true, a.name, false
	} else {
		l.nextMember++
		present = l.nextMember
		l.players[a.id] = &Player{Name: a.name, Connected: true, Color: l.playerColorLocked(), Powerups: make([]bool, len(*l.conf.Powerups))}
		l.resetReadyLocked()
	}
	l.leases[a.id] = &playerLease{client: c, generation: a.generation, present: present}
	l.selectAdminLocked()
	l.reconcilePowerupWindowLocked(&e)
	if !l.hub.RegisterClient(c) {
		e.close(c)
	} else {
		l.broadcastLocked(&e, models.ResponseBase{Status: "OK", Type: "JOINED_LOBBY", Payload: models.ResponsePayload{Lobby: l.snapshotLocked(), User: a.id}})
		if returning {
			l.sendLocked(&e, c, models.ResponseBase{Status: "OK", Type: "REJOIN_ROUND", Payload: l.resyncLocked()})
		}
	}
	l.launchPowerupStartLocked(&e)
	l.mu.Unlock()
	l.applyEffects(e)
	go c.Write()
	go c.Read()
}

func (l *Lobby) ownsLocked(c *websocket.Client) bool {
	lease := l.leases[c.ID]
	return !l.closed && lease != nil && lease.client == c && l.players[c.ID] != nil && l.players[c.ID].Connected
}

// Invalidate and enqueue feedback in the same transaction, before any newer
// start can begin. Superseded provider completions must not send another error.
func (l *Lobby) cancelStartLocked(e *lobbyEffects) {
	hadPowerStart := l.powerStart != nil
	wasStarting := l.starting
	l.powerStart = nil
	if l.startCancel != nil {
		l.startCancel()
		l.startCancel = nil
	}
	initiator := l.startInitiator
	l.startInitiator = nil
	if l.starting {
		l.generation++ // invalidates any in-flight completion
		l.starting = false
		if initiator != nil && l.ownsLocked(initiator) && !initiator.Closed() {
			l.errorLocked(e, initiator, "start", "START_CANCELLED")
		}
	}
	if hadPowerStart || wasStarting {
		l.broadcastPowerupLocked(e)
	}
}

func (l *Lobby) selectAdminLocked() {
	if p := l.players[l.admin]; p != nil && p.Connected && l.liveLeaseLocked(l.admin) {
		return
	}
	l.admin = ""
	var oldest uint64
	for id, p := range l.players {
		if !p.Connected || !l.liveLeaseLocked(id) {
			continue
		}
		order := l.leases[id].present
		if l.admin == "" || order < oldest || (order == oldest && id < l.admin) {
			l.admin = id
			oldest = order
		}
	}
}

func (l *Lobby) liveLeaseLocked(id string) bool {
	lease := l.leases[id]
	return lease != nil && lease.client != nil && !lease.client.Closed()
}

// A transport loss is NOT the explicit disconnect command. Preserve membership,
// lives and results even for the last socket, with the round clock still running.
func (l *Lobby) clientLost(c *websocket.Client) {
	var e lobbyEffects
	l.mu.Lock()
	if l.ownsLocked(c) {
		l.cancelStartLocked(&e)
		l.reconcilePowerupWindowLocked(&e)
		l.hub.UnregisterClient(c)
		l.players[c.ID].Connected, l.players[c.ID].Ready = false, false
		lease := l.leases[c.ID]
		lease.expires = time.Now().Add(reconnectGrace)
		lease.grace = time.AfterFunc(reconnectGrace, func() { l.expireLease(c.ID, lease, lease.generation) })
		l.selectAdminLocked()
		l.reconcilePowerupWindowLocked(&e)
		l.broadcastLocked(&e, models.ResponseBase{Status: "OK", Type: "LEFT_LOBBY", Payload: models.ResponsePayload{User: c.ID, Lobby: l.snapshotLocked()}})
	}
	l.mu.Unlock()
	l.applyEffects(e)
}

func (l *Lobby) leaveLocked(e *lobbyEffects, c *websocket.Client) {
	l.cancelStartLocked(e)
	l.reconcilePowerupWindowLocked(e)
	l.removePowerupMemberLocked(e, c.ID)
	l.hub.UnregisterClient(c)
	delete(l.players, c.ID)
	delete(l.leases, c.ID)
	l.selectAdminLocked()
	e.closeWith(c, ws.CloseNormalClosure, "Left lobby")
	l.afterRemovalLocked(e, c.ID)
}

func (l *Lobby) expireLease(id string, lease *playerLease, generation uint64) {
	var e lobbyEffects
	l.mu.Lock()
	if p := l.players[id]; !l.closed && l.leases[id] == lease && lease.generation == generation && p != nil && !p.Connected && !time.Now().Before(lease.expires) {
		l.cancelStartLocked(&e)
		l.reconcilePowerupWindowLocked(&e)
		l.removePowerupMemberLocked(&e, id)
		if a := l.reservations[id]; a != nil && a.previous == lease {
			a.timer.Stop()
			delete(l.reservations, id)
		}
		delete(l.players, id)
		delete(l.leases, id)
		l.selectAdminLocked()
		l.afterRemovalLocked(&e, id)
	}
	l.mu.Unlock()
	l.applyEffects(e)
}

func (l *Lobby) afterRemovalLocked(e *lobbyEffects, id string) {
	l.resetReadyLocked()
	l.reconcilePowerupWindowLocked(e)
	l.broadcastPowerupLocked(e)
	l.broadcastLocked(e, models.ResponseBase{Status: "OK", Type: "LEFT_LOBBY", Payload: models.ResponsePayload{User: id, Lobby: l.snapshotLocked()}})
	if len(l.players) == 0 && len(l.reservations) == 0 {
		l.closeLocked(e)
	} else if l.active && l.allFinishedLocked() {
		l.finalizeLocked(e, l.roundGen)
	}
}

func (l *Lobby) expireEmpty() {
	var e lobbyEffects
	l.mu.Lock()
	if l.emptyTimer != nil {
		// The initial wait is spent. Pending admissions now own the empty
		// lobby's lifetime; the last Cancel must be able to close it.
		l.emptyTimer = nil
		if len(l.players) == 0 && len(l.reservations) == 0 {
			l.closeLocked(&e)
		}
	}
	l.mu.Unlock()
	l.applyEffects(e)
}

func (l *Lobby) closeLocked(e *lobbyEffects) {
	if l.closed {
		return
	}
	l.closed = true
	l.stopPowerupWindowLocked()
	l.cancelStartLocked(e)
	if l.roundTimer != nil {
		l.roundTimer.Stop()
	}
	if l.emptyTimer != nil {
		l.emptyTimer.Stop()
		l.emptyTimer = nil
	}
	for _, lease := range l.leases {
		if lease.grace != nil {
			lease.grace.Stop()
		}
	}
	for _, a := range l.reservations {
		a.timer.Stop()
	}
	clear(l.reservations)
	e.closeLobby = true
}
