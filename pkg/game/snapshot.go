package game

import (
	"maps"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/logic"
)

// DTOs contain only detached values. Never put *Lobby, live players, config,
// result maps, polygon byte slices or slice backing arrays into a payload.
type LobbySnapshot struct {
	ID               string                      `json:"ID"`
	GameID           string                      `json:"gameId,omitempty"`
	Spectating       bool                        `json:"spectating,omitempty"`
	SpectatorCount   int                         `json:"spectatorCount,omitempty"`
	OpenToSpectators bool                        `json:"openToSpectators,omitempty"`
	EndedEarly       bool                        `json:"endedEarly,omitempty"`
	Admin            string                      `json:"admin"`
	Conf             LobbyConf                   `json:"conf"`
	NumPlayers       int                         `json:"numPlayers"`
	Players          map[string]Player           `json:"playerList"`
	MatchRoster      map[string]playerIdentity   `json:"matchRoster,omitempty"`
	CurrentRound     int                         `json:"currentRound"`
	Results          map[int]map[string][]Result `json:"results"`
	EndResults       map[int]map[string]Result   `json:"endResults"`
	RevealedAnswers  map[int]logic.Coords        `json:"revealedAnswers,omitempty"`
	TotalResults     map[string]Result           `json:"totalResults"`
	Phase            string                      `json:"phase"`
	Round            int                         `json:"round"`
	FullRoundRes     map[string][]Result         `json:"fullroundRes"`
	RoundRes         map[string]Result           `json:"roundRes"`
	Location         *logic.Coords               `json:"location,omitempty"`
	TimeRemaining    int64                       `json:"timeRemaining"`
	PowerLog         []Powerup                   `json:"powerLog"`
	PowerupState     *PowerupState               `json:"powerupState,omitempty"`
}

type rejoinPayload struct {
	GameID        string              `json:"gameId,omitempty"`
	Phase         string              `json:"phase"`
	Round         int                 `json:"round"`
	Lobby         LobbySnapshot       `json:"lobby"`
	Players       map[string]Player   `json:"players"`
	TimeRemaining int64               `json:"timeRemaining"`
	FullRoundRes  map[string][]Result `json:"fullroundRes"`
	RoundRes      map[string]Result   `json:"roundRes"`
	TotalResults  map[string]Result   `json:"totalResults"`
	Location      *logic.Coords       `json:"location,omitempty"`
	PowerLog      []Powerup           `json:"powerLog"`
}

func cloneConf(c LobbyConf) LobbyConf {
	if c.CCList != nil {
		c.CCList = append([]string{}, c.CCList...)
	}
	if c.Powerups != nil {
		v := append([]bool{}, (*c.Powerups)...)
		c.Powerups = &v
	}
	if c.PlaceBonus != nil {
		v := *c.PlaceBonus
		c.PlaceBonus = &v
	}
	if c.DynLives != nil {
		v := *c.DynLives
		c.DynLives = &v
	}
	return c
}

func copyResults(in map[string]*Result) map[string]Result {
	out := make(map[string]Result, len(in))
	for id, r := range in {
		if r != nil {
			out[id] = *r
		}
	}
	return out
}

func copyRaw(in map[string][]Result) map[string][]Result {
	out := make(map[string][]Result, len(in))
	for id, r := range in {
		out[id] = append([]Result{}, r...)
	}
	return out
}

func (l *Lobby) playersLocked() map[string]Player {
	out := make(map[string]Player, len(l.players))
	for id, p := range l.players {
		v := *p
		v.Powerups = append([]bool{}, p.Powerups...)
		out[id] = v
	}
	return out
}

func (l *Lobby) remainingLocked() int64 {
	if !l.active {
		return 0
	}
	return max(0, time.Until(l.deadline).Milliseconds())
}

// publicSnapshotLocked preserves the legacy discovery fields and JSON shape.
// Match results are available only through the appropriate socket audience.
func (l *Lobby) publicSnapshotLocked() LobbySnapshot {
	snapshot := LobbySnapshot{
		ID: l.id, GameID: l.gameID, Admin: l.admin, Conf: cloneConf(*l.conf), NumPlayers: len(l.players),
		Players: l.playersLocked(), CurrentRound: l.round, Round: l.round, Phase: l.phase,
		TimeRemaining: l.remainingLocked(), EndedEarly: l.endedEarly,
		OpenToSpectators: l.canOfferSpectatingLocked() && l.occupiedSpectatorsLocked() < maxSpectators,
		Results:          map[int]map[string][]Result{}, EndResults: map[int]map[string]Result{}, TotalResults: map[string]Result{},
		FullRoundRes: map[string][]Result{}, RoundRes: map[string]Result{},
	}
	for id, player := range snapshot.Players {
		player.Powerups, player.Lives = nil, 0
		player.RematchSeat = false
		snapshot.Players[id] = player
	}
	return snapshot
}

func (l *Lobby) snapshotLocked() LobbySnapshot {
	power := l.powerupStateLocked()
	out := LobbySnapshot{
		PowerupState: &power,
		ID:           l.id, GameID: l.gameID, Admin: l.admin, Conf: cloneConf(*l.conf), NumPlayers: len(l.players), EndedEarly: l.endedEarly,
		Players: l.playersLocked(), CurrentRound: l.round, Round: l.round, Phase: l.phase,
		MatchRoster:     maps.Clone(l.matchRoster),
		Results:         make(map[int]map[string][]Result, len(l.rawResults)),
		EndResults:      make(map[int]map[string]Result, len(l.endResults)),
		RevealedAnswers: make(map[int]logic.Coords, len(l.endResults)),
		TotalResults:    copyResults(l.totalResults), FullRoundRes: copyRaw(l.rawResults[l.round]),
		RoundRes: copyResults(l.endResults[l.round]), TimeRemaining: l.remainingLocked(),
		PowerLog: append([]Powerup{}, l.powerLogs[l.round]...),
	}
	for round, r := range l.rawResults {
		out.Results[round] = copyRaw(r)
	}
	for round, r := range l.endResults {
		out.EndResults[round] = copyResults(r)
		// The current endResults entry exists during PLAYING, before finalization.
		// Earlier entries are final because advancement requires the round to end.
		finalized := round < l.round || (round == l.round &&
			(l.phase == PhaseMidRound || l.phase == PhaseFinished) && l.finalizedGen == l.roundGen)
		if l.phase != PhaseWaiting && finalized && round > 0 && round <= len(l.locations) {
			out.RevealedAnswers[round] = l.locations[round-1]
		}
	}
	if l.phase != PhaseWaiting && l.round > 0 && l.round <= len(l.locations) {
		loc := l.locations[l.round-1]
		out.Location = &loc
	}
	return out
}

func (l *Lobby) resyncLocked() rejoinPayload {
	s := l.snapshotLocked()
	return rejoinPayload{
		GameID: s.GameID, Phase: s.Phase, Round: s.Round, Lobby: s, Players: s.Players,
		TimeRemaining: s.TimeRemaining, FullRoundRes: s.FullRoundRes, RoundRes: s.RoundRes,
		TotalResults: s.TotalResults, Location: s.Location, PowerLog: s.PowerLog,
	}
}
