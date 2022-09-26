package lobby

import (
	"errors"
	"example/web-service-gin/pkg/logic"
	"fmt"
	"sort"
)

func UsePowerup(powerup logic.Powerup, lobbyID string) error {
	switch powerup.Type {
	case 0:
		// double points
		if !LobbyMap[lobbyID].PlayerMap[powerup.Source].Powerups[0] {
			return errors.New("NOT_AVAILABLE")
		}
		LobbyMap[lobbyID].PlayerMap[powerup.Source].Powerups[0] = false
		LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound+1] = append(LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound+1], powerup)
		fmt.Println("PWLOG", LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound+1])
	case 1:
		// duel
		if !LobbyMap[lobbyID].PlayerMap[powerup.Source].Powerups[1] {
			return errors.New("NOT_AVAILABLE")
		}
		if _, ok := LobbyMap[lobbyID].PlayerMap[powerup.Target]; !ok || powerup.Source == powerup.Target {
			return errors.New("WRONG_TARGET")
		}
		LobbyMap[lobbyID].PlayerMap[powerup.Source].Powerups[1] = false
		LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound+1] = append(LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound+1], powerup)
		fmt.Println("PWLOG", LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound+1])
	default:
		return errors.New("WRONG_TYPE")
	}
	return nil
}

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
			// double points
			if result, ok := LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][power.Source]; ok {
				result.Score *= 2
				LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][power.Source] = result
				fmt.Println("DOUBLE SCORE")
			}
			fmt.Println(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound])
		case 1:
			// duel
			resultSource := LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][power.Source]
			resultTarget := LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][power.Target]
			// if source player left dont process anything
			if _, okS := LobbyMap[lobbyID].PlayerMap[power.Source]; okS {
				fmt.Println("player left, dont do duel")
				break
			}
			// if target left refund source and dont process anything
			if _, okT := LobbyMap[lobbyID].PlayerMap[power.Source]; okT {
				fmt.Println("player left, dont do duel, refund player")
				LobbyMap[lobbyID].PlayerMap[power.Source].Powerups[1] = true
				break
			}

			if resultSource.Dist < resultTarget.Dist {
				resultSource.Score += 1000
				resultTarget.Score -= 1000
			} else {
				resultSource.Score -= 1000
				resultTarget.Score += 1000
			}
			fmt.Println("DUEL SCORE")
			fmt.Println(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound])

		}
	}
	return nil
}

func ProcessBonus(lobbyID string) error {
	if !*LobbyMap[lobbyID].Conf.PlaceBonus {
		return errors.New("BONUS_DISABLED")
	}
	var playerOrder []string
	for name := range LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound] {
		playerOrder = append(playerOrder, name)
	}
	sort.SliceStable(playerOrder, func(i, j int) bool {
		// fmt.Println(i, j, LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[i]].Dist < LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[j]].Dist)
		if LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[j]].Attempt == 0 {
			return true
		}
		if LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[i]].Attempt == 0 {
			return false
		}
		return LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[i]].Dist < LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[j]].Dist
	})
	fmt.Println("PLAYER ORDER", playerOrder)
	switch num := len(playerOrder); {
	case num == 2:
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[0]].Score = int(float64(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[0]].Score) * 1.1)
	case num == 3:
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[0]].Score = int(float64(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[0]].Score) * 1.25)
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[1]].Score = int(float64(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[1]].Score) * 1.1)
	case num >= 4:
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[0]].Score = int(float64(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[0]].Score) * 1.5)
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[1]].Score = int(float64(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[1]].Score) * 1.25)
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[2]].Score = int(float64(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][playerOrder[2]].Score) * 1.1)
	}
	return nil
}
