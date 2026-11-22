package game

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/models"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
)

const powerupWindowDuration = 10 * time.Second

type PowerupEffect struct {
	ID        string `json:"id"`
	RequestID string `json:"requestId"`
	GameID    string `json:"gameId"`
	Round     int    `json:"round"`
	Type      int    `json:"type"`
	Source    string `json:"source"`
	Target    string `json:"target,omitempty"`
	Status    string `json:"status"`
	Outcome   string `json:"outcome,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type PowerupState struct {
	GameID         string          `json:"gameId,omitempty"`
	Round          int             `json:"round"`
	TargetRound    *int            `json:"targetRound"`
	Status         string          `json:"status"`
	ClosesAt       *int64          `json:"closesAt"`
	ServerNow      int64           `json:"serverNow"`
	RemainingMs    int64           `json:"remainingMs"`
	DecisionReady  map[string]bool `json:"decisionReady"`
	StartRequested bool            `json:"startRequested"`
	Starting       bool            `json:"starting"`
	Effects        []PowerupEffect `json:"effects"`
}

type powerupRecord struct {
	PowerupEffect
	sourceMember uint64
	targetMember uint64
}

type powerupWindow struct {
	round      int
	generation uint64
	deadline   time.Time
	timer      *time.Timer
	open       bool
	// Membership order survives reconnect, unlike the owning socket generation.
	done         map[uint64]bool
	participants map[string]bool
}

type powerupReceiptKey struct {
	member  uint64
	request string
}
type powerupReceipt struct {
	command string
	game    string
	round   int
	kind    int
	target  string
	effect  int
}

type powerupStatePayload struct {
	GameID       string            `json:"gameId,omitempty"`
	Round        int               `json:"round"`
	Phase        string            `json:"phase"`
	Players      map[string]Player `json:"players"`
	PowerupState PowerupState      `json:"powerupState"`
}

type powerupUsedPayload struct {
	GameID       string            `json:"gameId"`
	RequestID    string            `json:"requestId"`
	Effect       PowerupEffect     `json:"effect"`
	PowerupState PowerupState      `json:"powerupState"`
	Players      map[string]Player `json:"players"`
}

type powerupReadyPayload struct {
	GameID       string       `json:"gameId"`
	RequestID    string       `json:"requestId"`
	PowerupState PowerupState `json:"powerupState"`
}

type duelFromPayload struct {
	User     string `json:"user"`
	GameID   string `json:"gameId"`
	Round    int    `json:"round"`
	EffectID string `json:"effectId"`
}

func validPowerupRequestID(id string) bool {
	if len(id) < 1 || len(id) > 64 {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

// Decode separately from legacy Powerup: a missing/null type must never mean 0.
func normalizedPowerupRequest(req ClientReq) (powerupReceipt, error) {
	r := powerupReceipt{command: req.Cmd, effect: -1, kind: -1}
	if req.GameID == nil || *req.GameID == "" || req.Round == nil || *req.Round < 1 || req.RequestID == nil || !validPowerupRequestID(*req.RequestID) {
		return r, errors.New("INVALID_REQUEST")
	}
	r.game, r.round = *req.GameID, *req.Round
	if req.Cmd == "use_powerup" {
		var p struct {
			Type   *int            `json:"type"`
			Target json.RawMessage `json:"target"`
		}
		if len(req.Powerup) == 0 || json.Unmarshal(req.Powerup, &p) != nil || p.Type == nil {
			return r, errors.New("INVALID_REQUEST")
		}
		r.kind = *p.Type
		if p.Target != nil {
			if string(p.Target) == "null" || json.Unmarshal(p.Target, &r.target) != nil {
				return r, errors.New("INVALID_REQUEST")
			}
		}
	}
	return r, nil
}

func (l *Lobby) powerupStateLocked() PowerupState {
	now := time.Now()
	out := PowerupState{GameID: l.gameID, Round: l.round, Status: "CLOSED", ServerNow: now.UnixMilli(), DecisionReady: map[string]bool{}, Starting: l.starting, StartRequested: l.powerStart != nil, Effects: make([]PowerupEffect, len(l.powerEffects))}
	for i, r := range l.powerEffects {
		out.Effects[i] = r.PowerupEffect
	}
	if l.phase == PhaseMidRound {
		target := l.round + 1
		out.TargetRound = &target
	}
	if w := l.powerWindow; w != nil {
		at := w.deadline.UnixMilli()
		out.ClosesAt = &at
		if w.open && now.Before(w.deadline) && l.phase == PhaseMidRound {
			out.Status = "OPEN"
			out.RemainingMs = max(0, w.deadline.Sub(now).Milliseconds())
		}
		for id, done := range w.participants {
			out.DecisionReady[id] = done
		}
	}
	return out
}

func (l *Lobby) broadcastPowerupLocked(e *lobbyEffects) {
	l.broadcastLocked(e, models.ResponseBase{Status: "OK", Type: "POWERUP_STATE", Payload: powerupStatePayload{GameID: l.gameID, Round: l.round, Phase: l.phase, Players: l.playersLocked(), PowerupState: l.powerupStateLocked()}})
}

func (l *Lobby) usablePowerupLocked(id string) bool {
	p := l.players[id]
	if p == nil || !p.Connected || !l.liveLeaseLocked(id) || len(p.Powerups) != 2 || l.conf.Powerups == nil {
		return false
	}
	enabled := *l.conf.Powerups
	if len(enabled) != 2 {
		return false
	}
	if enabled[0] && p.Powerups[0] {
		return true
	}
	if enabled[1] && p.Powerups[1] {
		for target, other := range l.players {
			if target != id && other.Connected && l.liveLeaseLocked(target) {
				return true
			}
		}
	}
	return false
}

func (l *Lobby) openPowerupWindowLocked() {
	l.stopPowerupWindowLocked()
	l.powerGeneration++
	duration := powerupWindowDuration
	if enabled := l.conf.Powerups; enabled != nil && len(*enabled) == 2 && (*enabled)[0] && (*enabled)[1] {
		duration = 20 * time.Second
	}
	w := &powerupWindow{round: l.round, generation: l.powerGeneration, deadline: time.Now().Add(duration), open: true, done: map[uint64]bool{}, participants: map[string]bool{}}
	l.powerWindow = w
	l.refreshPowerupParticipantsLocked()
	if w.open {
		w.timer = time.AfterFunc(time.Until(w.deadline), func() { l.powerupExpired(w.generation) })
	}
}

func (l *Lobby) stopPowerupWindowLocked() {
	if w := l.powerWindow; w != nil {
		w.open = false
		if w.timer != nil {
			w.timer.Stop()
			w.timer = nil
		}
	}
	l.powerStart = nil
}

// Returns whether externally visible participants/closure changed. done is never
// removed on disconnect or restored inventory; a CLOSED window never reopens.
func (l *Lobby) refreshPowerupParticipantsLocked() bool {
	w := l.powerWindow
	if w == nil {
		return false
	}
	changed := false
	next := map[string]bool{}
	allDone := true
	for id, p := range l.players {
		lease := l.leases[id]
		if !p.Connected || lease == nil || !l.liveLeaseLocked(id) {
			continue
		}
		usable := l.usablePowerupLocked(id)
		_, seen := w.done[lease.present]
		if !w.open && !seen {
			continue
		}
		if !usable && !seen {
			continue
		}
		if !usable && w.open {
			w.done[lease.present] = true
		} else if !seen {
			w.done[lease.present] = false
		}
		done := w.done[lease.present]
		next[id] = done
		if !done {
			allDone = false
		}
	}
	if len(next) != len(w.participants) {
		changed = true
	}
	for id, done := range next {
		old, ok := w.participants[id]
		if !ok || old != done {
			changed = true
		}
	}
	w.participants = next
	if w.open && (!time.Now().Before(w.deadline) || allDone) {
		w.open = false
		if w.timer != nil {
			w.timer.Stop()
			w.timer = nil
		}
		changed = true
	}
	return changed
}

func (l *Lobby) reconcilePowerupWindowLocked(e *lobbyEffects) {
	if l.refreshPowerupParticipantsLocked() {
		l.broadcastPowerupLocked(e)
	}
}

func (l *Lobby) powerupExpired(generation uint64) {
	var e lobbyEffects
	l.mu.Lock()
	if !l.closed && l.powerWindow != nil && l.powerWindow.generation == generation && l.phase == PhaseMidRound {
		l.reconcilePowerupWindowLocked(&e)
		l.launchPowerupStartLocked(&e)
	}
	l.mu.Unlock()
	l.applyEffects(e)
}

// Reserve only after the closing state and action receipt have been enqueued.
func (l *Lobby) launchPowerupStartLocked(e *lobbyEffects) {
	if l.closed || l.phase != PhaseMidRound || l.powerWindow == nil || l.powerWindow.open || l.powerStart == nil || l.starting {
		return
	}
	c := l.powerStart
	if !l.ownsLocked(c) || c.Closed() || c.ID != l.admin {
		l.powerStart = nil
		l.broadcastPowerupLocked(e)
		return
	}
	work, err := l.beginStartLocked(c)
	if err != nil {
		l.powerStart = nil
		l.broadcastPowerupLocked(e)
		l.errorLocked(e, c, "start", err.Error())
		return
	}
	e.selection = work
	l.broadcastPowerupLocked(e)
}

func (l *Lobby) requestPowerupStartLocked(e *lobbyEffects, c *websocket.Client, req ClientReq) error {
	if c.ID != l.admin {
		return errors.New("NOT_ADMIN")
	}
	if req.GameID == nil || *req.GameID == "" || req.Round == nil || *req.Round < 1 {
		return errors.New("INVALID_REQUEST")
	}
	if *req.GameID != l.gameID || *req.Round != l.round {
		return errors.New("STALE_POWERUP_CONTEXT")
	}
	l.reconcilePowerupWindowLocked(e)
	if l.powerWindow == nil {
		return errors.New("POWERUP_WINDOW_CLOSED")
	}
	if l.starting || l.powerStart == c {
		return nil
	}
	if len(l.reservations) != 0 {
		return errors.New("LOBBY_BUSY")
	}
	l.powerStart = c
	if lease := l.leases[c.ID]; lease != nil && l.powerWindow.open {
		l.powerWindow.done[lease.present] = true
	}
	l.refreshPowerupParticipantsLocked()
	l.broadcastPowerupLocked(e)
	l.launchPowerupStartLocked(e)
	return nil
}

func (l *Lobby) powerupCommandLocked(e *lobbyEffects, c *websocket.Client, req ClientReq) error {
	if !l.ownsLocked(c) || c.Closed() {
		return errors.New("PLAYER_NOT_IN_LOBBY")
	}
	content, err := normalizedPowerupRequest(req)
	if err != nil {
		return err
	}
	// An old game cannot replay a receipt after successful replay reset.
	if content.game != l.gameID {
		return errors.New("STALE_POWERUP_CONTEXT")
	}
	key := powerupReceiptKey{member: l.leases[c.ID].present, request: *req.RequestID}
	if receipt, ok := l.powerReceipts[key]; ok {
		same := receipt
		same.effect = -1
		if same != content {
			return errors.New("REQUEST_ID_CONFLICT")
		}
		l.reconcilePowerupWindowLocked(e)
		l.powerupReceiptLocked(e, c, *req.RequestID, receipt)
		l.launchPowerupStartLocked(e)
		return nil
	}
	if content.round != l.round {
		return errors.New("STALE_POWERUP_CONTEXT")
	}
	l.reconcilePowerupWindowLocked(e)
	w := l.powerWindow
	if l.phase != PhaseMidRound || w == nil || !w.open || l.starting {
		return errors.New("POWERUP_WINDOW_CLOSED")
	}
	member := l.leases[c.ID].present
	if w.done[member] {
		return errors.New("POWERUP_DECISION_DONE")
	}
	cap := l.conf.MaxPlayers * l.conf.NumRounds
	if req.Cmd == "use_powerup" {
		if content.kind < 0 || content.kind > 1 {
			return errors.New("WRONG_TYPE")
		}
		p := l.players[c.ID]
		if len(p.Powerups) != 2 || l.conf.Powerups == nil || len(*l.conf.Powerups) != 2 || !(*l.conf.Powerups)[content.kind] || !p.Powerups[content.kind] {
			return errors.New("NOT_AVAILABLE")
		}
		if content.kind == 0 && content.target != "" {
			return errors.New("WRONG_TARGET")
		}
		if content.kind == 1 {
			target := l.players[content.target]
			if content.target == c.ID || target == nil || !target.Connected || !l.liveLeaseLocked(content.target) {
				return errors.New("WRONG_TARGET")
			}
		}
		for _, r := range l.powerEffects {
			if r.sourceMember == member && r.Round == l.round+1 && r.Type == content.kind && (r.Status == "QUEUED" || r.Status == "ACTIVE" || r.Status == "APPLIED") {
				return errors.New("NOT_AVAILABLE")
			}
		}
		if len(l.powerEffects) >= 2*cap || len(l.powerReceipts) >= 3*cap {
			return errors.New("POWERUP_LIMIT")
		}
		record := powerupRecord{PowerupEffect: PowerupEffect{ID: l.gameID + "-" + strconv.Itoa(len(l.powerEffects)+1), RequestID: *req.RequestID, GameID: l.gameID, Round: l.round + 1, Type: content.kind, Source: c.ID, Target: content.target, Status: "QUEUED"}, sourceMember: member}
		if content.kind == 1 {
			record.targetMember = l.leases[content.target].present
		}
		// Validation can scan the bounded match ledger; sample again at the
		// actual commit so validation work cannot carry a spend past cutoff.
		if !time.Now().Before(w.deadline) {
			l.reconcilePowerupWindowLocked(e)
			return errors.New("POWERUP_WINDOW_CLOSED")
		}
		p.Powerups[content.kind] = false
		content.effect = len(l.powerEffects)
		l.powerEffects = append(l.powerEffects, record)
		l.powerLogs[l.round+1] = append(l.powerLogs[l.round+1], Powerup{Type: content.kind, Source: c.ID, Target: content.target})
	} else {
		if len(l.powerReceipts) >= 3*cap {
			return errors.New("POWERUP_LIMIT")
		}
		if !time.Now().Before(w.deadline) {
			l.reconcilePowerupWindowLocked(e)
			return errors.New("POWERUP_WINDOW_CLOSED")
		}
		w.done[member] = true
	}
	if l.powerReceipts == nil {
		l.powerReceipts = map[powerupReceiptKey]powerupReceipt{}
	}
	l.powerReceipts[key] = content
	l.refreshPowerupParticipantsLocked()
	l.broadcastPowerupLocked(e)
	l.powerupReceiptLocked(e, c, *req.RequestID, content)
	if content.kind == 1 {
		r := l.powerEffects[content.effect]
		l.sendLocked(e, l.leases[content.target].client, models.ResponseBase{Status: "WRN", Type: "DUEL_FROM", Payload: duelFromPayload{User: c.ID, GameID: l.gameID, Round: r.Round, EffectID: r.ID}})
	}
	l.launchPowerupStartLocked(e)
	return nil
}

func (l *Lobby) powerupReceiptLocked(e *lobbyEffects, c *websocket.Client, id string, r powerupReceipt) {
	if r.effect >= 0 {
		l.sendLocked(e, c, models.ResponseBase{Status: "OK", Type: "POWERUP_USED", Payload: powerupUsedPayload{GameID: l.gameID, RequestID: id, Effect: l.powerEffects[r.effect].PowerupEffect, PowerupState: l.powerupStateLocked(), Players: l.playersLocked()}})
	} else {
		l.sendLocked(e, c, models.ResponseBase{Status: "OK", Type: "POWERUP_READY", Payload: powerupReadyPayload{GameID: l.gameID, RequestID: id, PowerupState: l.powerupStateLocked()}})
	}
}

// Call before deleting membership: internal membership identities prevent a
// later admission of the same public ID from inheriting a cancelled effect.
func (l *Lobby) removePowerupMemberLocked(e *lobbyEffects, id string) {
	lease := l.leases[id]
	if lease == nil {
		return
	}
	for i := range l.powerEffects {
		r := &l.powerEffects[i]
		if r.Status != "QUEUED" && r.Status != "ACTIVE" {
			continue
		}
		if r.Source == id && r.sourceMember == lease.present {
			r.Status = "CANCELLED"
			r.Reason = "SOURCE_LEFT"
		} else if r.Type == 1 && r.Target == id && r.targetMember == lease.present {
			r.Status = "REFUNDED"
			r.Reason = "TARGET_LEFT"
			if source := l.leases[r.Source]; source != nil && source.present == r.sourceMember {
				if p := l.players[r.Source]; p != nil && len(p.Powerups) == 2 {
					p.Powerups[1] = true
				}
			}
		}
	}
	l.rebuildPowerLogLocked()
}

func (l *Lobby) rebuildPowerLogLocked() {
	l.powerLogs = map[int][]Powerup{}
	for _, r := range l.powerEffects {
		if r.Status != "CANCELLED" && r.Status != "REFUNDED" {
			l.powerLogs[r.Round] = append(l.powerLogs[r.Round], Powerup{Type: r.Type, Source: r.Source, Target: r.Target})
		}
	}
}
