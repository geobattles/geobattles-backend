package lobby

import (
	"example/web-service-gin/pkg/logic"
	"fmt"
)

type Lobby struct {
	Name       string            `json:"name"`
	Admin      string            `json:"admin"`
	MaxPlayers int               `json:"maxPlayers"`
	NumPlayers int               `json:"numPlayers"`
	PlayerList map[string]string `json:"playerList"`
	//GameActive      bool                         `json:"gameActive"`
	CurrentLocation logic.Coordinates            `json:"-"`
	CurrentRound    int                          `json:"currentRound"`
	Results         map[int]map[string][]float64 `json:"results"`
}

// initial lobbz list for debugging
var LobbyMap = map[string]*Lobby{
	"U4YPR6": {Name: "prvi lobby", MaxPlayers: 8, NumPlayers: 0, PlayerList: make(map[string]string), Results: make(map[int]map[string][]float64)},
	"8CKXRG": {Name: "LOBBY #2", MaxPlayers: 6, NumPlayers: 0, PlayerList: make(map[string]string), Results: make(map[int]map[string][]float64)},
}

// adds player as map[id]name to playerlist in lobby
func AddPlayerToLobby(clientID string, clientName string, lobbyID string) {
	// if there is no lobby admin make this user one
	if LobbyMap[lobbyID].Admin == "" {
		LobbyMap[lobbyID].Admin = clientID
	}
	LobbyMap[lobbyID].PlayerList[clientID] = clientName
}

// removes player map from playerlist in lobby
func RemovePlayerFromLobby(clientID string, lobbyID string) {
	delete(LobbyMap[lobbyID].PlayerList, clientID)
	// if removed player was admin & there are other players left
	// select one of them as new admin, otherwise make admin empty
	if LobbyMap[lobbyID].Admin == clientID && len(LobbyMap[lobbyID].PlayerList) != 0 {
		for id := range LobbyMap[lobbyID].PlayerList {
			LobbyMap[lobbyID].Admin = id
			break
		}
	} else {
		LobbyMap[lobbyID].Admin = ""
	}
}

// TODO: could use round number instead, use 0 or -1 as inactive
// func MarkGameActive(lobbyID string) {
// 	fmt.Println("Lobby starting: ", lobbyID)
// 	LobbyMap[lobbyID].GameActive = true
// }

// keeps track of the location of the currently active game in lobby
// increments round counter every call
func UpdateCurrentLocation(lobbyID string, location logic.Coordinates) {
	fmt.Println("updating lobby loaction: ", lobbyID, location)
	LobbyMap[lobbyID].CurrentLocation = location
	LobbyMap[lobbyID].CurrentRound++
	//fmt.Println("current round: ", LobbyMap[lobbyID].CurrentRound)
}

// calculates distance/score between correct and user submited coordinates
func CalculateDistance(lobbyID string, userLocation logic.Coordinates) float64 {
	fmt.Println("req calculate distance")
	return logic.CalcDistance(LobbyMap[lobbyID].CurrentLocation, userLocation)
}

// adds result to map of all results in lobby
func AddToResults(lobbyID string, clientID string, result float64) {
	// if user currently doesnt have a result in this round create new map
	if LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound] == nil {
		LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound] = make(map[string][]float64)
	}
	// TODO: split this monstrosity, maybe use variables
	LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound][clientID] = append(LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound][clientID], result)
}
