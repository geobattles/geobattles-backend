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
	returnWindow   = 90 * time.Second
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

// AdmissionStatus is advisory and carries no game or other player data.
type AdmissionStatus struct {
	Code         string `json:"code"`
	Retained     bool   `json:"retained,omitempty"`
	RetryAfterMs int64  `json:"retryAfterMs,omitempty"`
}

func CheckAdmission(lobbyID, id string) AdmissionStatus {
	l := lookupLobby(lobbyID)
	if l == nil {
		return AdmissionStatus{Code: "LOBBY_NOT_FOUND"}
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.admissionStatusLocked(id, time.Now())
}

func (l *Lobby) idleExpiredLocked(now time.Time) bool {
	return !l.idleDeadline.IsZero() && !now.Before(l.idleDeadline)
}

func (l *Lobby) seatExpiredLocked(lease *playerLease, now time.Time) bool {
	return l.phase != PhasePlaying && l.phase != PhaseMidRound && !lease.expires.IsZero() && !now.Before(lease.expires)
}

func (l *Lobby) occupiedSeatsLocked() int {
	occupied := len(l.players)
	for _, reservation := range l.reservations {
		if reservation.previous == nil {
			occupied++
		}
	}
	return occupied
}

func (l *Lobby) newPlayerStatusLocked(occupied int) string {
	if l.phase != PhaseWaiting && l.phase != PhaseFinished {
		return "GAME_IN_PROGRESS"
	}
	if occupied >= l.conf.MaxPlayers {
		return "LOBBY_FULL"
	}
	return "OK"
}

func (l *Lobby) admissionStatusLocked(id string, now time.Time) AdmissionStatus {
	if l.closed || l.idleExpiredLocked(now) {
		return AdmissionStatus{Code: "LOBBY_NOT_FOUND"}
	}
	previous := l.leases[id]
	retained := previous != nil && !l.seatExpiredLocked(previous, now)
	if l.starting {
		return AdmissionStatus{Code: "ROUND_STARTING", Retained: retained}
	}
	if pending := l.reservations[id]; pending != nil {
		return AdmissionStatus{Code: "ADMISSION_PENDING", Retained: retained, RetryAfterMs: max(0, pending.deadline.Sub(now).Milliseconds())}
	}
	if p := l.players[id]; p != nil && p.Connected {
		return AdmissionStatus{Code: "ALREADY_CONNECTED", Retained: true}
	}
	if previous != nil && !retained {
		return AdmissionStatus{Code: "REJOIN_EXPIRED"}
	}
	if !retained {
		return AdmissionStatus{Code: l.newPlayerStatusLocked(l.occupiedSeatsLocked())}
	}
	return AdmissionStatus{Code: "OK", Retained: retained}
}

func ReserveAdmission(lobbyID, id, name string) (*Admission, error) {
	l := lookupLobby(lobbyID)
	if l == nil {
		return nil, errors.New("LOBBY_NOT_FOUND")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	if status := l.admissionStatusLocked(id, now); status.Code != "OK" {
		if status.Code == "ADMISSION_PENDING" {
			return nil, errors.New("ALREADY_CONNECTED") // preserve the upgrade contract
		}
		return nil, errors.New(status.Code)
	}
	l.nextLease++
	a := &Admission{lobby: l, id: id, name: name, generation: l.nextLease, previous: l.leases[id], deadline: now.Add(admissionWait)}
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
	if l.closed || l.idleExpiredLocked(now) || l.starting || l.reservations[a.id] != a || l.leases[a.id] != a.previous ||
		!now.Before(a.deadline) || (a.previous != nil && l.seatExpiredLocked(a.previous, now)) {
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
	l.clearIdleLocked()
	returning := a.previous != nil
	var present uint64
	if returning {
		a.previous.stopTimers()
		present = a.previous.present
		p := l.players[a.id]
		p.Connected, p.Name, p.Ready, p.Away = true, a.name, false, false
	} else {
		l.nextMember++
		present = l.nextMember
		l.players[a.id] = &Player{Name: a.name, Connected: true, Color: l.playerColorLocked(), Powerups: make([]bool, len(*l.conf.Powerups))}
		l.resetReadyLocked()
	}
	l.leases[a.id] = &playerLease{client: c, generation: a.generation, present: present}
	if l.active && now.Before(l.deadline) {
		l.roundEntrants[a.id] = true
	}
	previousAdmin := l.admin
	l.selectAdminLocked()
	if previousAdmin != l.admin && l.powerStart != nil {
		l.cancelStartLocked(&e)
	}
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
	if p := l.players[l.creator]; p != nil && p.Connected && l.liveLeaseLocked(l.creator) {
		l.admin = l.creator
		return
	}
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

func (lease *playerLease) stopTimers() {
	if lease.grace != nil {
		lease.grace.Stop()
	}
	if lease.seatTimer != nil {
		lease.seatTimer.Stop()
	}
}

func (l *Lobby) clientLost(c *websocket.Client) {
	var e lobbyEffects
	l.mu.Lock()
	if l.ownsLocked(c) {
		l.retainSeatLocked(&e, c, false)
	}
	l.mu.Unlock()
	l.applyEffects(e)
}

// Both exits preserve the exact match membership. A deliberate exit stops
// holding up round completion immediately; a network loss gets a short grace.
func (l *Lobby) retainSeatLocked(e *lobbyEffects, c *websocket.Client, away bool) {
	l.cancelStartLocked(e)
	l.reconcilePowerupWindowLocked(e)
	l.hub.UnregisterClient(c)
	p := l.players[c.ID]
	p.Connected, p.Ready, p.Away = false, false, away
	id := c.ID
	lease := l.leases[id]
	lease.client = nil
	now := time.Now()
	lease.expires = now.Add(returnWindow)
	if away {
		e.closeWith(c, ws.CloseNormalClosure, "Left lobby with a returnable seat")
	} else if l.phase == PhasePlaying {
		lease.blocksUntil = now.Add(reconnectGrace)
		lease.grace = time.AfterFunc(time.Until(lease.blocksUntil), func() {
			var effects lobbyEffects
			l.mu.Lock()
			if l.leases[id] == lease {
				l.maybeFinishLocked(&effects)
			}
			l.mu.Unlock()
			l.applyEffects(effects)
		})
	}
	l.scheduleSeatExpiryLocked(id, lease)
	l.selectAdminLocked()
	l.updateIdleLocked()
	l.reconcilePowerupWindowLocked(e)
	l.broadcastLocked(e, models.ResponseBase{Status: "OK", Type: "LEFT_LOBBY", Payload: models.ResponsePayload{User: c.ID, Lobby: l.snapshotLocked()}})
	l.maybeFinishLocked(e)
}

func (l *Lobby) scheduleSeatExpiryLocked(id string, lease *playerLease) {
	if lease.seatTimer != nil {
		lease.seatTimer.Stop()
	}
	lease.seatTimer = time.AfterFunc(time.Until(lease.expires), func() {
		var e lobbyEffects
		l.mu.Lock()
		if p := l.players[id]; !l.closed && l.leases[id] == lease && p != nil && !p.Connected && l.seatExpiredLocked(lease, time.Now()) {
			l.removeMemberLocked(&e, id)
			l.afterRemovalLocked(&e, id)
		}
		l.mu.Unlock()
		l.applyEffects(e)
	})
}

func (l *Lobby) removeMemberLocked(e *lobbyEffects, id string) {
	l.removePowerupMemberLocked(e, id)
	if lease := l.leases[id]; lease != nil {
		lease.stopTimers()
	}
	if pending := l.reservations[id]; pending != nil {
		pending.timer.Stop()
		delete(l.reservations, id)
	}
	delete(l.players, id)
	delete(l.leases, id)
}

func (l *Lobby) leaveLocked(e *lobbyEffects, c *websocket.Client) {
	l.cancelStartLocked(e)
	l.reconcilePowerupWindowLocked(e)
	l.hub.UnregisterClient(c)
	l.removeMemberLocked(e, c.ID)
	l.resetReadyLocked()
	e.closeWith(c, ws.CloseNormalClosure, "Left lobby")
	l.afterRemovalLocked(e, c.ID)
}

func (l *Lobby) maybeFinishLocked(e *lobbyEffects) {
	if !l.closed && l.active && l.allFinishedLocked() {
		l.finalizeLocked(e, l.roundGen)
	}
}

func (l *Lobby) afterRemovalLocked(e *lobbyEffects, id string) {
	l.selectAdminLocked()
	l.reconcilePowerupWindowLocked(e)
	l.broadcastPowerupLocked(e)
	l.broadcastLocked(e, models.ResponseBase{Status: "OK", Type: "LEFT_LOBBY", Payload: models.ResponsePayload{User: id, Lobby: l.snapshotLocked()}})
	if len(l.players) == 0 && len(l.reservations) == 0 {
		l.closeLocked(e)
		return
	}
	l.updateIdleLocked()
	l.maybeFinishLocked(e)
}

func (l *Lobby) clearIdleLocked() {
	if l.idleTimer != nil {
		l.idleTimer.Stop()
		l.idleTimer = nil
	}
	l.idleDeadline = time.Time{}
}

func (l *Lobby) updateIdleLocked() {
	if l.admin != "" || l.idleTimer != nil {
		return
	}
	deadline := time.Now().Add(returnWindow)
	l.idleDeadline = deadline
	l.idleTimer = time.AfterFunc(time.Until(deadline), func() {
		var e lobbyEffects
		l.mu.Lock()
		if l.idleDeadline == deadline {
			l.closeLocked(&e)
		}
		l.mu.Unlock()
		l.applyEffects(e)
	})
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
	l.clearIdleLocked()
	for _, lease := range l.leases {
		lease.stopTimers()
	}
	for _, a := range l.reservations {
		a.timer.Stop()
	}
	clear(l.reservations)
	e.closeLobby = true
}
