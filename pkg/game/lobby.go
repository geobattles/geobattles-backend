package game

import (
	"errors"
	"math"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/defaults"
	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/reverse"
)

var colorList = [12]string{"#e6194B", "#3cb44b", "#4363d8", "#f58231", "#911eb4", "#42d4f4", "#f032e6", "#000075", "#469990", "#9A6324", "#dcbeff", "#800000"}

func defaultConf(c LobbyConf) LobbyConf {
	c = cloneConf(c)
	if c.Mode == 0 {
		c.Mode = defaults.Mode
	}
	if c.Mode == 2 {
		c.ScoreFactor = 0
	}
	if c.CCList == nil {
		c.CCList = []string{}
	}
	if c.Mode != 2 && c.ScoreFactor == 0 {
		c.ScoreFactor = defaults.ScoreFactor
	}
	if c.Powerups == nil {
		c.Powerups = defaults.Powerups()
	}
	if c.PlaceBonus == nil {
		c.PlaceBonus = defaults.PlaceBonus()
	}
	if c.DynLives == nil {
		c.DynLives = defaults.DynLives()
	}
	if c.MaxPlayers == 0 {
		c.MaxPlayers = defaults.MaxPlayers
	}
	if c.NumAttempt == 0 {
		c.NumAttempt = defaults.NumOfTries
	}
	if c.NumRounds == 0 {
		c.NumRounds = defaults.NumOfRounds
	}
	if c.RoundTime == 0 {
		c.RoundTime = defaults.RoundTime
	}
	return c
}

// All helpers in this file require l.mu. Updates are atomic and copy inputs.
func (l *Lobby) updateConfLocked(clientID string, c LobbyConf) error {
	if clientID != l.admin {
		return errors.New("NOT_ADMIN")
	}
	if l.starting || (l.phase != PhaseWaiting && l.phase != PhaseFinished) {
		return errors.New("GAME_IN_PROGRESS")
	}
	if len(l.reservations) != 0 {
		return errors.New("LOBBY_BUSY")
	}
	if err := ValidateLobbyConf(c); err != nil {
		return err
	}
	codes, err := logic.NormalizeCountryCodes(c.CCList)
	if err != nil {
		return err
	}
	if c.MaxPlayers > 0 && c.MaxPlayers < len(l.players) {
		return errors.New("LOBBY_CAPACITY_TOO_SMALL")
	}
	c = cloneConf(c)
	next := cloneConf(*l.conf)
	if c.Name != "" {
		next.Name = c.Name
	}
	if c.Mode != 0 && c.Mode != next.Mode {
		next.Mode = c.Mode
		if c.Mode == 2 {
			next.ScoreFactor = 0
		} else {
			next.ScoreFactor = defaults.ScoreFactor
		}
	}
	if c.ScoreFactor != 0 {
		next.ScoreFactor = c.ScoreFactor
	}
	if c.Powerups != nil {
		next.Powerups = c.Powerups
	}
	if c.PlaceBonus != nil {
		next.PlaceBonus = c.PlaceBonus
	}
	if c.DynLives != nil {
		next.DynLives = c.DynLives
	}
	if c.MaxPlayers > 0 {
		next.MaxPlayers = c.MaxPlayers
	}
	if c.NumAttempt > 0 {
		next.NumAttempt = c.NumAttempt
	}
	if c.NumRounds > 0 {
		next.NumRounds = c.NumRounds
	}
	if c.RoundTime > 0 {
		next.RoundTime = c.RoundTime
	}
	if c.CCList != nil {
		if !sameRegion(next.CCList, codes) && c.ScoringMode == "" {
			next.ScoringMode = ScoringAuto
		}
		next.CCList = codes
	}
	if c.ScoringMode != "" {
		next.ScoringMode = c.ScoringMode
	}
	// Validate/derive the whole candidate before the single commit. In
	// particular, failures must not partially change fields or readiness.
	next, size, err := normalizeConf(next)
	if err != nil {
		return err
	}
	l.conf, l.ccSize = &next, size
	return nil
}

func (l *Lobby) playerColorLocked() string {
cycle:
	for _, color := range colorList {
		for _, p := range l.players {
			if p.Color == color {
				continue cycle
			}
		}
		return color
	}
	return colorList[0]
}

func (l *Lobby) resetResultsLocked() {
	l.stopPowerupWindowLocked()
	l.powerWindow = nil
	l.powerEffects = nil
	l.powerReceipts = make(map[powerupReceiptKey]powerupReceipt)
	l.rawResults = make(map[int]map[string][]Result)
	l.endResults = make(map[int]map[string]*Result)
	l.totalResults = make(map[string]*Result)
	l.powerLogs = make(map[int][]Powerup)
	l.locations = make([]logic.Coords, 0, l.conf.NumRounds)
	l.finished = make(map[string]bool)
	l.round = 0
	l.persisted = false
}

func (l *Lobby) setupRoundLocked(location logic.Coords, cc string) {
	l.finished = make(map[string]bool)
	l.roundEntrants = make(map[string]bool)
	l.locations = append(l.locations, location)
	l.currentCC = cc
	l.startTime = time.Now()
	l.active, l.phase = true, PhasePlaying
	l.round++
	l.stopPowerupWindowLocked()
	for i := range l.powerEffects {
		if r := &l.powerEffects[i]; r.Round == l.round && r.Status == "QUEUED" {
			r.Status = "ACTIVE"
		}
	}
	l.rawResults[l.round] = make(map[string][]Result)
	l.endResults[l.round] = make(map[string]*Result)
	for id, p := range l.players {
		if p.Connected && l.liveLeaseLocked(id) {
			l.roundEntrants[id] = true
		}
		l.endResults[l.round][id] = &Result{}
		if l.round == 1 {
			l.totalResults[id] = &Result{}
			p.Powerups = append([]bool{}, (*l.conf.Powerups)...)
			p.Lives = l.conf.NumAttempt
		} else if *l.conf.DynLives {
			p.Lives = min(p.Lives+(l.conf.NumAttempt+1)/2, l.conf.NumAttempt+1)
		} else {
			p.Lives = l.conf.NumAttempt
		}
	}
}

// Away seats stay eligible to return, but only a recent network loss can
// delay early completion. An empty room lets the normal round clock run.
func (l *Lobby) allFinishedLocked() bool {
	hasLivePlayer := false
	now := time.Now()
	for id, player := range l.players {
		if player.Connected && l.liveLeaseLocked(id) {
			hasLivePlayer = true
		} else if lease := l.leases[id]; lease == nil || !now.Before(lease.blocksUntil) {
			continue
		}
		if !l.finished[id] {
			return false
		}
	}
	return hasLivePlayer
}

func scoreDistance(x float64, a float64) int {
	score := int(5000 * math.Pow(0.999, (x/1000-a/1000)*(0.2+30*(math.Pow(0.98, 10*math.Pow(a, 0.6))))))
	return min(score, 5000)
}

func (l *Lobby) submitResultLocked(id string, loc logic.Coords) error {
	if !loc.Valid() {
		return logic.ErrInvalidLocation
	}
	if l.round <= 0 || l.round > len(l.locations) {
		return errors.New("GAME_NOT_ACTIVE")
	}
	if !l.active {
		return errors.New("TIMES_UP")
	}
	p := l.players[id]
	if p == nil || !p.Connected {
		return errors.New("PLAYER_NOT_IN_LOBBY")
	}
	if l.rawResults[l.round] == nil || l.endResults[l.round][id] == nil || l.totalResults[id] == nil {
		return errors.New("INVALID_GAME_STATE")
	}
	if p.Lives <= 0 {
		return errors.New("NO_MORE_ATTEMPTS")
	}
	if l.conf.Mode == 2 {
		return l.countryGuessLocked(id, loc)
	}
	distance := logic.CalcDistance(l.locations[l.round-1], loc)
	if math.IsNaN(distance) || math.IsInf(distance, 0) {
		return logic.ErrInvalidLocation
	}
	p.Lives--
	result := Result{Loc: loc, Dist: distance, BaseScore: l.conf.scoreDistance(distance), Lives: p.Lives, Attempt: len(l.rawResults[l.round][id]) + 1}
	l.rawResults[l.round][id] = append(l.rawResults[l.round][id], result)
	best := l.endResults[l.round][id]
	if best.Attempt == 0 || best.Dist > distance {
		v := result
		l.endResults[l.round][id] = &v
	}
	if p.Lives <= 0 {
		l.finished[id] = true
	}
	return nil
}

// Retain the existing unfinished Country Battle behavior; no feature completion.
func (l *Lobby) countryGuessLocked(id string, loc logic.Coords) error {
	if l.finished[id] {
		return errors.New("ALREADY_FINISHED")
	}
	cc, err := reverse.ReverseGeocode(loc.Lng, loc.Lat)
	if err != nil {
		return err
	}
	p := l.players[id]
	p.Lives--
	timeUsed := int(time.Since(l.startTime).Microseconds() - 3*1000000)
	score := min(5000, int(float64(l.conf.RoundTime*1000000-timeUsed)/float64((l.conf.RoundTime-4)*1000000)*5000))
	if cc == l.currentCC {
		cc = "XX"
	} else {
		score = 0
	}
	r := Result{Loc: loc, BaseScore: score, Time: timeUsed, Lives: p.Lives, Attempt: len(l.rawResults[l.round][id]) + 1, CC: cc}
	l.rawResults[l.round][id] = append(l.rawResults[l.round][id], r)
	l.endResults[l.round][id] = &r
	if p.Lives <= 0 || cc == "XX" {
		l.finished[id] = true
	}
	return nil
}
