package lobby

import (
	"errors"
	"example/web-service-gin/pkg/logic"
	"fmt"
	"sort"
)

// uses given powerup. adds succesfully used powerup to powerlog
func UsePowerup(powerup logic.Powerup, lobbyID string) (string, error) {
	switch powerup.Type {
	case 0:
		// double points
		if !LobbyMap[lobbyID].PlayerMap[powerup.Source].Powerups[0] {
			return "", errors.New("NOT_AVAILABLE")
		}
		LobbyMap[lobbyID].PlayerMap[powerup.Source].Powerups[0] = false
		LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound+1] = append(LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound+1], powerup)
		// fmt.Println("PWLOG", LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound+1])
	case 1:
		// duel
		if !LobbyMap[lobbyID].PlayerMap[powerup.Source].Powerups[1] {
			return "", errors.New("NOT_AVAILABLE")
		}
		if _, ok := LobbyMap[lobbyID].PlayerMap[powerup.Target]; !ok || powerup.Source == powerup.Target {
			return "", errors.New("WRONG_TARGET")
		}
		LobbyMap[lobbyID].PlayerMap[powerup.Source].Powerups[1] = false
		LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound+1] = append(LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound+1], powerup)
		// fmt.Println("PWLOG", LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound+1])
		return powerup.Target, nil
	default:
		return "", errors.New("WRONG_TYPE")
	}
	return "", nil
}

// processes powerups from powerlog. double score is given priority in processing order
func ProcessPowerups(lobbyID string) error {
	fmt.Println("PROCESS POWER", LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound])
	// sort powerlogs so that type 0 is processed before others
	sort.Slice(LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound], func(p, q int) bool {
		return LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound][p].Type < LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound][q].Type
	})
	for _, power := range LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound] {
		fmt.Println("powerup", power)
		switch power.Type {
		case 0:
			// TODO: dont stack with placement bonus
			// double points
			if result, ok := LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][power.Source]; ok {
				result.DoubleScore = result.BaseScore
				LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][power.Source] = result
			}
		case 1:
			// duel
			resultSource := LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][power.Source]
			resultTarget := LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][power.Target]
			// if source player left dont process anything
			if _, okS := LobbyMap[lobbyID].PlayerMap[power.Source]; !okS {
				fmt.Println("player left, dont do duel")
				break
			}
			// if target left refund source and dont process anything
			if _, okT := LobbyMap[lobbyID].PlayerMap[power.Source]; !okT {
				fmt.Println("player left, dont do duel, refund player")
				LobbyMap[lobbyID].PlayerMap[power.Source].Powerups[1] = true
				break
			}

			switch LobbyMap[lobbyID].Conf.Mode {
			case 2:
				if resultSource.Attempt == 0 || (resultTarget.Time <= resultSource.Time && resultTarget.Attempt != 0) {
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
func ProcessBonus(lobbyID string) error {
	if !*LobbyMap[lobbyID].Conf.PlaceBonus {
		fmt.Println("BONUS_DISABLED")
		return errors.New("BONUS_DISABLED")
	}
	var playerOrder []string
	for name := range LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound] {
		playerOrder = append(playerOrder, name)
	}
	sort.SliceStable(playerOrder, func(i, j int) bool {
		if LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[j]].Attempt == 0 {
			return true
		}
		if LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[i]].Attempt == 0 {
			return false
		}
		switch LobbyMap[lobbyID].Conf.Mode {
		case 2:
			return LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[i]].Time < LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[j]].Time
		default:
			return LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[i]].Dist < LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[j]].Dist
		}
	})
	fmt.Println("PLAYER ORDER", playerOrder)
	switch num := len(playerOrder); {
	case num == 2:
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[0]].PlaceScore = int(float64(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[0]].BaseScore) * 0.1)
	case num == 3:
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[0]].PlaceScore = int(float64(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[0]].BaseScore) * 0.2)
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[1]].PlaceScore = int(float64(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[1]].BaseScore) * 0.1)
	case num >= 4:
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[0]].PlaceScore = int(float64(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[0]].BaseScore) * 0.3)
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[1]].PlaceScore = int(float64(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[1]].BaseScore) * 0.2)
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[2]].PlaceScore = int(float64(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[2]].BaseScore) * 0.1)
	}
	return nil
}

// add bonus points based on player placement. first player gets 30% bonus, second 20, third 10
// depends on the number of players; with 2 players only first gets 10%
func ProcessTotal(lobbyID string) error {
	for player, result := range LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound] {
		LobbyMap[lobbyID].TotalResults[player].Total += result.BaseScore + result.PlaceScore + result.DoubleScore + result.DuelScore
	}
	return nil
}
