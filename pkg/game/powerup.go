package game

import (
	"errors"
	"log/slog"
	"sort"

	"github.com/geobattles/geobattles-backend/pkg/models"
)

// uses given powerup. adds succesfully used powerup to powerlog
// Requires l.mu; targets the exact current owning connection.
func (l *Lobby) usePowerupLocked(e *lobbyEffects, sourceID string, powerup Powerup) error {
	player := l.players[sourceID]
	if player == nil || !player.Connected {
		return errors.New("PLAYER_NOT_IN_LOBBY")
	}
	if len(player.Powerups) != 2 || l.powerLogs == nil {
		return errors.New("INVALID_GAME_STATE")
	}
	if l.round == 0 || l.phase == PhaseFinished {
		return errors.New("GAME_NOT_ACTIVE")
	}
	if l.round == l.conf.NumRounds {
		return errors.New("CANT_USE_LAST_ROUND")
	}

	powerup.Source = sourceID

	switch powerup.Type {
	// double points
	case 0:
		if !l.players[sourceID].Powerups[0] {
			return errors.New("NOT_AVAILABLE")
		}
		l.players[sourceID].Powerups[0] = false
		l.powerLogs[l.round+1] = append(l.powerLogs[l.round+1], powerup)
		return nil

	// duel
	case 1:
		if !l.players[sourceID].Powerups[1] {
			return errors.New("NOT_AVAILABLE")
		}
		if target := l.players[powerup.Target]; target == nil || !target.Connected || sourceID == powerup.Target {
			return errors.New("WRONG_TARGET")
		}
		l.players[sourceID].Powerups[1] = false
		l.powerLogs[l.round+1] = append(l.powerLogs[l.round+1], powerup)

		if lease := l.leases[powerup.Target]; lease != nil {
			l.sendLocked(e, lease.client, models.ResponseBase{
				Status: "WRN", Type: "DUEL_FROM", Payload: models.ResponsePayload{User: sourceID},
			})
		}

		return nil
	default:
		return errors.New("WRONG_TYPE")
	}
}

// processes powerups from powerlog. double score is given priority in processing order
func (l *Lobby) processPowerups() error {
	slog.Info("Processing power", "PowerLogs", l.powerLogs[l.round])
	// sort powerlogs so that type 0 is processed before others
	sort.Slice(l.powerLogs[l.round], func(p, q int) bool {
		return l.powerLogs[l.round][p].Type < l.powerLogs[l.round][q].Type
	})
	for _, power := range l.powerLogs[l.round] {
		slog.Debug("Processing powerup", "Power", power)
		switch power.Type {
		case 0:
			// TODO: dont stack with placement bonus
			// double points
			if result, ok := l.endResults[l.round][power.Source]; ok {
				result.DoubleScore = result.BaseScore
				l.endResults[l.round][power.Source] = result
			}
		case 1:
			// duel
			resultSource := l.endResults[l.round][power.Source]
			resultTarget := l.endResults[l.round][power.Target]
			// if source player left dont process anything
			if _, okS := l.players[power.Source]; !okS {
				slog.Debug("Player left, dont do duel")
				break
			}
			// if target left refund source and dont process anything
			if _, okT := l.players[power.Target]; !okT {
				slog.Debug("Player left, dont do duel, refund player")
				l.players[power.Source].Powerups[1] = true
				break
			}

			if resultSource == nil || resultTarget == nil {
				break
			}

			switch l.conf.Mode {
			case 2:
				// if neither user guessed a country dont apply points
				if (resultSource.Attempt == 0 || resultSource.CC != "XX") && (resultTarget.Attempt == 0 || resultTarget.CC != "XX") {
					break
				}
				if (resultSource.Attempt == 0 || resultSource.CC != "XX") || (resultTarget.Time <= resultSource.Time && !(resultTarget.Attempt == 0 || resultTarget.CC != "XX")) {
					resultSource.DuelScore -= 1000
					resultTarget.DuelScore += 1000
				} else {
					resultSource.DuelScore += 1000
					resultTarget.DuelScore -= 1000
				}
			default:
				if resultSource.Attempt == 0 || (resultTarget.Dist <= resultSource.Dist && resultTarget.Attempt != 0) {
					resultSource.DuelScore -= 1000
					resultTarget.DuelScore += 1000
				} else {
					resultSource.DuelScore += 1000
					resultTarget.DuelScore -= 1000
				}
			}
		}
	}
	return nil
}

// add bonus points based on player placement. first player gets 30% bonus, second 20, third 10
// depends on the number of players; with 2 players only first gets 10%
func (l *Lobby) processBonus() error {
	if !*l.conf.PlaceBonus {
		slog.Debug("Bonus disabled")
		return errors.New("BONUS_DISABLED")
	}
	var playerOrder []string
	for name := range l.endResults[l.round] {
		playerOrder = append(playerOrder, name)
	}
	sort.SliceStable(playerOrder, func(i, j int) bool {
		iEmpty := l.endResults[l.round][playerOrder[i]].Attempt == 0
		jEmpty := l.endResults[l.round][playerOrder[j]].Attempt == 0
		if iEmpty || jEmpty {
			return !iEmpty && jEmpty // both placeholders compare false
		}
		switch l.conf.Mode {
		case 2:
			return l.endResults[l.round][playerOrder[i]].Time < l.endResults[l.round][playerOrder[j]].Time
		default:
			return l.endResults[l.round][playerOrder[i]].Dist < l.endResults[l.round][playerOrder[j]].Dist
		}
	})
	slog.Debug("Player order", "playerOrder", playerOrder)
	switch num := len(playerOrder); {
	case num == 2:
		l.endResults[l.round][playerOrder[0]].PlaceScore = int(float64(l.endResults[l.round][playerOrder[0]].BaseScore) * 0.1)
	case num == 3:
		l.endResults[l.round][playerOrder[0]].PlaceScore = int(float64(l.endResults[l.round][playerOrder[0]].BaseScore) * 0.2)
		l.endResults[l.round][playerOrder[1]].PlaceScore = int(float64(l.endResults[l.round][playerOrder[1]].BaseScore) * 0.1)
	case num >= 4:
		l.endResults[l.round][playerOrder[0]].PlaceScore = int(float64(l.endResults[l.round][playerOrder[0]].BaseScore) * 0.3)
		l.endResults[l.round][playerOrder[1]].PlaceScore = int(float64(l.endResults[l.round][playerOrder[1]].BaseScore) * 0.2)
		l.endResults[l.round][playerOrder[2]].PlaceScore = int(float64(l.endResults[l.round][playerOrder[2]].BaseScore) * 0.1)
	}
	return nil
}

// processes total score for each player (total score = base + place + double + duel)
func (l *Lobby) processTotal() error {
	for player, result := range l.endResults[l.round] {
		roundTotal := result.BaseScore + result.PlaceScore + result.DoubleScore + result.DuelScore
		l.endResults[l.round][player].Total = roundTotal
		l.totalResults[player].Total += roundTotal
	}
	return nil
}
