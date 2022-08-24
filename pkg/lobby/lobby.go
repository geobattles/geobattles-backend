package lobby

import (
	"example/web-service-gin/pkg/defaults"
	"example/web-service-gin/pkg/logic"
	"fmt"
	"math"
)

type Lobby struct {
	Name            string                             `json:"name"`
	Admin           string                             `json:"admin"`
	MaxPlayers      int                                `json:"maxPlayers"`
	NumPlayers      int                                `json:"numPlayers"`
	PlayerList      map[string]string                  `json:"playerList"`
	NumAttempt      int                                `json:"numAttempt"`
	RoundTime       int                                `json:"roundTime"`
	CurrentLocation logic.Coordinates                  `json:"-"`
	ScoreFactor     int                                `json:"scoreFactor"`
	CurrentRound    int                                `json:"currentRound"`
	Results         map[int]map[string][]logic.Results `json:"results"`
}

// initial lobby list for debugging
var LobbyMap = map[string]*Lobby{
	"U4YPR6": {Name: "prvi lobby", MaxPlayers: 8, NumPlayers: 0, PlayerList: make(map[string]string), ScoreFactor: 100, NumAttempt: 3, RoundTime: 60, Results: make(map[int]map[string][]logic.Results)},
	"8CKXRG": {Name: "LOBBY #2", MaxPlayers: 6, NumPlayers: 0, PlayerList: make(map[string]string), ScoreFactor: 60, NumAttempt: 2, RoundTime: 40, Results: make(map[int]map[string][]logic.Results)},
}

// validates values and creates new lobby
func CreateLobby(name string, maxPlayers int, numAttempt int, scoreFactor int, roundTime int) *Lobby {
	var newLobby Lobby
	newLobby.PlayerList = make(map[string]string)
	newLobby.Results = make(map[int]map[string][]logic.Results)
	lobbyID := logic.GenerateRndID(6)
	// validate values and set defaults otherwise
	if name == "" {
		newLobby.Name = lobbyID
	} else {
		newLobby.Name = name
	}

	if maxPlayers <= 0 {
		newLobby.MaxPlayers = defaults.MaxPlayers
	} else {
		newLobby.MaxPlayers = maxPlayers
	}

	if numAttempt <= 0 {
		newLobby.NumAttempt = defaults.NumOfTries
	} else {
		newLobby.NumAttempt = numAttempt
	}

	if scoreFactor == 0 || scoreFactor < defaults.ScoreFactorLow || scoreFactor > defaults.ScoreFactorHigh {
		newLobby.ScoreFactor = defaults.ScoreFactor
	} else {
		newLobby.ScoreFactor = scoreFactor
	}

	if roundTime <= 0 {
		newLobby.RoundTime = defaults.RoundTime
	} else {
		newLobby.RoundTime = roundTime
	}

	LobbyMap[lobbyID] = &newLobby
	return LobbyMap[lobbyID]
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

// calculate score based on distance and scorefactor
// scorefactor determines how fast score drops off
// for example a low scorefactor means if your guess is 100km off you only get 100 points
// it also means you get full points for distance closer that scorefactor
func scoreDistance(x float64, a float64) int {
	//return int(5000 * math.Pow(0.999, (x/1000-a/1000)*(0.25+150*(math.Pow(0.62, math.Sqrt(a))))))
	score := int(5000 * math.Pow(0.999, (x/1000-a/1000)*(0.2+30*(math.Pow(0.98, 10*math.Pow(a, 0.6))))))

	if score > 5000 {
		return 5000
	}
	return score
}

// adds result to map of all results in lobby
func AddToResults(lobbyID string, clientID string, location logic.Coordinates, distance float64) {
	// if user currently doesnt have a result in this round create new map
	if LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound] == nil {
		LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound] = make(map[string][]logic.Results)
	}
	score := scoreDistance(distance, float64(LobbyMap[lobbyID].ScoreFactor))
	fmt.Println("SCORE IS: ", score)
	// TODO: split this monstrosity, maybe use variables
	LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound][clientID] = append(LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound][clientID], logic.Results{Location: location, Distance: distance, Score: score})
}
