package game

import (
	"errors"
	"log/slog"
	"sort"

	"github.com/slinarji/go-geo-server/pkg/models"
)

// uses given powerup. adds succesfully used powerup to powerlog
func (l *Lobby) usePowerup(sourceID string, powerup Powerup) error {
	if l.CurrentRound == 0 {
		return errors.New("GAME_NOT_ACTIVE")
	}
	if l.CurrentRound == l.Conf.NumRounds {
		return errors.New("CANT_USE_LAST_ROUND")
	}

	powerup.Source = sourceID

	switch powerup.Type {
	// double points
	case 0:
		if !l.PlayerMap[sourceID].Powerups[0] {
			return errors.New("NOT_AVAILABLE")
		}
		l.PlayerMap[sourceID].Powerups[0] = false
		l.PowerLogs[l.CurrentRound+1] = append(l.PowerLogs[l.CurrentRound+1], powerup)
		return nil

	// duel
	case 1:
		if !l.PlayerMap[sourceID].Powerups[1] {
			return errors.New("NOT_AVAILABLE")
		}
		if _, ok := l.PlayerMap[powerup.Target]; !ok || sourceID == powerup.Target {
			return errors.New("WRONG_TARGET")
		}
		l.PlayerMap[sourceID].Powerups[1] = false
		l.PowerLogs[l.CurrentRound+1] = append(l.PowerLogs[l.CurrentRound+1], powerup)

		// notify target player
		for client := range l.Hub.Clients {
			if client.ID == powerup.Target {
				client.Send <- models.ResponseBase{Status: "WRN", Type: "DUEL_FROM", Payload: models.ResponsePayload{User: sourceID}}
			}
		}

		return nil
	default:
		return errors.New("WRONG_TYPE")
	}
}

// processes powerups from powerlog. double score is given priority in processing order
func (l *Lobby) processPowerups() error {
	slog.Info("Processing power", "PowerLogs", l.PowerLogs[l.CurrentRound])
	// sort powerlogs so that type 0 is processed before others
	sort.Slice(l.PowerLogs[l.CurrentRound], func(p, q int) bool {
		return l.PowerLogs[l.CurrentRound][p].Type < l.PowerLogs[l.CurrentRound][q].Type
	})
	for _, power := range l.PowerLogs[l.CurrentRound] {
		slog.Info("Processing powerup", "Power", power)
		switch power.Type {
		case 0:
			// TODO: dont stack with placement bonus
			// double points
			if result, ok := l.EndResults[l.CurrentRound][power.Source]; ok {
				result.DoubleScore = result.BaseScore
				l.EndResults[l.CurrentRound][power.Source] = result
			}
		case 1:
			// duel
			resultSource := l.EndResults[l.CurrentRound][power.Source]
			resultTarget := l.EndResults[l.CurrentRound][power.Target]
			// if source player left dont process anything
			if _, okS := l.PlayerMap[power.Source]; !okS {
				slog.Info("Player left, dont do duel")
				break
			}
			// if target left refund source and dont process anything
			if _, okT := l.PlayerMap[power.Source]; !okT {
				slog.Info("Player left, dont do duel, refund player")
				l.PlayerMap[power.Source].Powerups[1] = true
				break
			}

			switch l.Conf.Mode {
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
	if !*l.Conf.PlaceBonus {
		slog.Info("Bonus disabled")
		return errors.New("BONUS_DISABLED")
	}
	var playerOrder []string
	for name := range l.EndResults[l.CurrentRound] {
		playerOrder = append(playerOrder, name)
	}
	sort.SliceStable(playerOrder, func(i, j int) bool {
		if l.EndResults[l.CurrentRound][playerOrder[j]].Attempt == 0 {
			return true
		}
		if l.EndResults[l.CurrentRound][playerOrder[i]].Attempt == 0 {
			return false
		}
		switch l.Conf.Mode {
		case 2:
			return l.EndResults[l.CurrentRound][playerOrder[i]].Time < l.EndResults[l.CurrentRound][playerOrder[j]].Time
		default:
			return l.EndResults[l.CurrentRound][playerOrder[i]].Dist < l.EndResults[l.CurrentRound][playerOrder[j]].Dist
		}
	})
	slog.Info("Player order", "playerOrder", playerOrder)
	switch num := len(playerOrder); {
	case num == 2:
		l.EndResults[l.CurrentRound][playerOrder[0]].PlaceScore = int(float64(l.EndResults[l.CurrentRound][playerOrder[0]].BaseScore) * 0.1)
	case num == 3:
		l.EndResults[l.CurrentRound][playerOrder[0]].PlaceScore = int(float64(l.EndResults[l.CurrentRound][playerOrder[0]].BaseScore) * 0.2)
		l.EndResults[l.CurrentRound][playerOrder[1]].PlaceScore = int(float64(l.EndResults[l.CurrentRound][playerOrder[1]].BaseScore) * 0.1)
	case num >= 4:
		l.EndResults[l.CurrentRound][playerOrder[0]].PlaceScore = int(float64(l.EndResults[l.CurrentRound][playerOrder[0]].BaseScore) * 0.3)
		l.EndResults[l.CurrentRound][playerOrder[1]].PlaceScore = int(float64(l.EndResults[l.CurrentRound][playerOrder[1]].BaseScore) * 0.2)
		l.EndResults[l.CurrentRound][playerOrder[2]].PlaceScore = int(float64(l.EndResults[l.CurrentRound][playerOrder[2]].BaseScore) * 0.1)
	}
	return nil
}

// processes total score for each player (total score = base + place + double + duel)
func (l *Lobby) processTotal() error {
	for player, result := range l.EndResults[l.CurrentRound] {
		l.TotalResults[player].Total += result.BaseScore + result.PlaceScore + result.DoubleScore + result.DuelScore
	}
	return nil
}
