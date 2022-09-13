package lobby

import (
	"errors"
	"example/web-service-gin/pkg/logic"
	"fmt"
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
	default:
		return errors.New("WRONG_TYPE")
	}
	return nil
}

func ProcessPowerups(lobbyID string) error {
	fmt.Println("PROCESS POWER", LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound])
	for _, power := range LobbyMap[lobbyID].PowerLogs[LobbyMap[lobbyID].CurrentRound] {
		fmt.Println("powerup", power)
		switch power.Type {
		case 0:
			fmt.Println("CASE 0")
			// double points
			if result, ok := LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][power.Source]; ok {
				result.Score *= 2
				LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][power.Source] = result
				fmt.Println("DOUBLE SCORE")
			}
			fmt.Println(LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound])

		}
	}
	return nil
}
