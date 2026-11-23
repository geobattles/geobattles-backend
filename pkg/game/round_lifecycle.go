package game

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/models"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
	"github.com/google/uuid"
)

type roundStart struct {
	generation uint64
	newGame    bool
	initiator  *websocket.Client
	conf       LobbyConf
	countries  []string
	ccSize     float64
	ctx        context.Context
}

func (l *Lobby) beginStartLocked(c *websocket.Client) (*roundStart, error) {
	if c.ID != l.admin {
		return nil, errors.New("NOT_ADMIN")
	}
	if l.active {
		return nil, errors.New("ALREADY_ACTIVE")
	}
	if l.starting {
		return nil, errors.New("ROUND_STARTING")
	}
	if len(l.reservations) != 0 {
		return nil, errors.New("LOBBY_BUSY")
	}
	if l.phase != PhaseFinished && (l.round < 0 || l.round >= l.conf.NumRounds) {
		return nil, errors.New("INVALID_GAME_STATE")
	}
	newGame := l.phase == PhaseWaiting || l.phase == PhaseFinished
	if newGame && !l.allReadyLocked() {
		return nil, errors.New("PLAYERS_NOT_READY")
	}
	// Re-derive the captured configuration for initial starts and replays. It
	// commits only with successful selection, preserving failed replay state.
	conf, size, err := normalizeConf(*l.conf)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	l.startCancel = cancel
	l.startInitiator = c
	l.starting = true
	l.generation++
	return &roundStart{generation: l.generation, newGame: newGame, initiator: c, conf: conf, countries: append([]string{}, conf.CCList...), ccSize: size, ctx: ctx}, nil
}

// The single reserved selection runs outside all locks. The reader remains free
// to process explicit leave, and transport loss can cancel the provider context.
func (l *Lobby) completeStart(work *roundStart) {
	loc, cc, err := logic.RndLocationContext(work.ctx, work.countries, work.ccSize)
	var e lobbyEffects
	l.mu.Lock()
	live := !l.closed && l.starting && l.generation == work.generation &&
		l.ownsLocked(work.initiator) && !work.initiator.Closed() && l.admin == work.initiator.ID
	if !live {
		if l.starting && l.generation == work.generation {
			l.cancelStartLocked(&e)
		}
	} else {
		l.startCancel()
		l.startCancel = nil
		l.startInitiator = nil
		l.starting = false
		if err == nil && work.newGame && !l.allReadyLocked() {
			err = errors.New("PLAYERS_NOT_READY")
		}
		var gameID string
		if err == nil && work.newGame {
			id, idErr := uuid.NewRandom()
			if idErr != nil {
				err = errors.New("REQUEST_FAILED")
			} else {
				gameID = id.String()
			}
		}
		if err != nil {
			l.powerStart = nil
			l.broadcastPowerupLocked(&e)
			l.errorLocked(&e, work.initiator, "start", err.Error())
		} else {
			conf := cloneConf(work.conf)
			l.conf, l.ccSize = &conf, work.ccSize
			// Retain the last completed summary through failed/cancelled replay
			// selection; reset only at this successful commit point.
			if l.phase == PhaseFinished {
				for id, player := range l.players {
					if !player.Connected {
						l.removeMemberLocked(&e, id)
					}
				}
				l.resetResultsLocked()
			}
			if work.newGame {
				l.gameID = gameID
			}
			l.roundGen = work.generation
			l.setupRoundLocked(loc, cc)
			l.deadline = time.Now().Add(time.Duration(l.conf.RoundTime+3) * time.Second)
			generation := l.roundGen
			l.roundTimer = time.AfterFunc(time.Until(l.deadline), func() { l.roundExpired(generation) })
			l.broadcastLocked(&e, models.ResponseBase{Status: "OK", Type: "START_ROUND", Payload: models.ResponsePayload{
				GameID: l.gameID, Loc: loc, Players: l.playersLocked(), PowerLog: append([]Powerup{}, l.powerLogs[l.round]...), Round: l.round, Phase: l.phase,
				TimeRemaining: l.remainingLocked(),
				PowerupState:  l.powerupStateLocked(),
			}})
		}
	}
	l.mu.Unlock()
	l.applyEffects(e)
}

func (l *Lobby) roundExpired(generation uint64) {
	var e lobbyEffects
	l.mu.Lock()
	l.finalizeLocked(&e, generation)
	l.mu.Unlock()
	l.applyEffects(e)
}

// Timer, final guess, intentional leave and grace expiry all use this once-only
// generation guard under the same mutex. Scoring and persistence snapshot once.
func (l *Lobby) finalizeLocked(e *lobbyEffects, generation uint64) {
	if l.closed || !l.active || l.roundGen != generation || l.finalizedGen == generation {
		return
	}
	l.finalizedGen = generation
	l.active = false
	if l.roundTimer != nil {
		l.roundTimer.Stop()
		l.roundTimer = nil
	}
	_ = l.processBonus()
	_ = l.processPowerups()
	_ = l.processTotal()
	l.phase = PhaseMidRound
	if l.round >= l.conf.NumRounds {
		l.phase = PhaseFinished
		for id, player := range l.players {
			if !player.Connected {
				l.scheduleSeatExpiryLocked(id, l.leases[id])
			}
		}
		l.resetReadyLocked()
		if !l.persisted {
			l.persisted = true
			e.save = l.persistenceSnapshotLocked()
		}
	}
	if l.phase == PhaseMidRound {
		l.openPowerupWindowLocked()
	}
	l.broadcastLocked(e, models.ResponseBase{Status: "WRN", Type: "ROUND_FINISHED"})
	message := models.ResponsePayload{
		GameID: l.gameID, Round: l.round, Phase: l.phase, Players: l.playersLocked(),
		FullRoundRes: copyRaw(l.rawResults[l.round]), RoundRes: copyResults(l.endResults[l.round]),
		TotalResults: copyResults(l.totalResults), PowerLog: append([]Powerup{}, l.powerLogs[l.round]...),
		Loc:          l.locations[l.round-1],
		PowerupState: l.powerupStateLocked(),
	}
	if l.conf.Mode == 2 {
		message.Polygon = append(json.RawMessage(nil), logic.PolyDB[l.currentCC]...)
	}
	l.broadcastLocked(e, models.ResponseBase{Status: "OK", Type: "ROUND_RESULT", Payload: message})
	if l.phase == PhaseFinished {
		snapshot := l.snapshotLocked()
		l.broadcastLocked(e, models.ResponseBase{Status: "OK", Type: "GAME_END", Payload: models.ResponsePayload{
			GameID: l.gameID, AllRes: snapshot.Results, EndResults: snapshot.EndResults,
			RevealedAnswers: snapshot.RevealedAnswers, TotalResults: snapshot.TotalResults, Round: l.round, Phase: l.phase,
			PowerupState: l.powerupStateLocked(),
		}})
	}
}
