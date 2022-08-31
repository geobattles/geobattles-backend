package lobby

import (
	"errors"
	"example/web-service-gin/pkg/defaults"
	"example/web-service-gin/pkg/logic"
	"fmt"
	"math"
)

// initial lobby list for debugging
var LobbyMap = map[string]*logic.Lobby{
	"U4YPR6": {ID: "U4YPR6", Conf: &logic.LobbyConf{Name: "prvi lobby", MaxPlayers: 8, ScoreFactor: 100, NumAttempt: 3, NumRounds: 2, RoundTime: 30}, NumPlayers: 0, PlayerList: make(map[string]string), Results: make(map[int]map[string][]logic.Results)},
	"8CKXRG": {ID: "8CKXRG", Conf: &logic.LobbyConf{Name: "LOBBY #2", MaxPlayers: 8, ScoreFactor: 100, NumAttempt: 3, NumRounds: 2, RoundTime: 60}, NumPlayers: 0, PlayerList: make(map[string]string), Results: make(map[int]map[string][]logic.Results)},
}

// validates values and creates new lobby
func CreateLobby(conf *logic.LobbyConf) *logic.Lobby {
	var newLobby logic.Lobby
	newLobby.Conf = &logic.LobbyConf{}
	newLobby.PlayerList = make(map[string]string)
	newLobby.Results = make(map[int]map[string][]logic.Results)
	lobbyID := logic.GenerateRndID(6)
	newLobby.ID = lobbyID
	// validate values and set defaults otherwise
	if conf.Name == "" {
		newLobby.Conf.Name = lobbyID
	} else {
		newLobby.Conf.Name = conf.Name
	}

	if conf.MaxPlayers <= 0 {
		newLobby.Conf.MaxPlayers = defaults.MaxPlayers
	} else {
		newLobby.Conf.MaxPlayers = conf.MaxPlayers
	}

	if conf.NumAttempt <= 0 {
		newLobby.Conf.NumAttempt = defaults.NumOfTries
	} else {
		newLobby.Conf.NumAttempt = conf.NumAttempt
	}

	if conf.NumRounds <= 0 {
		newLobby.Conf.NumRounds = defaults.NumOfRounds
	} else {
		newLobby.Conf.NumRounds = conf.NumRounds
	}

	if conf.ScoreFactor == 0 || conf.ScoreFactor < defaults.ScoreFactorLow || conf.ScoreFactor > defaults.ScoreFactorHigh {
		newLobby.Conf.ScoreFactor = defaults.ScoreFactor
	} else {
		newLobby.Conf.ScoreFactor = conf.ScoreFactor
	}

	if conf.RoundTime <= 0 {
		newLobby.Conf.RoundTime = defaults.RoundTime
	} else {
		newLobby.Conf.RoundTime = conf.RoundTime
	}

	LobbyMap[lobbyID] = &newLobby
	return LobbyMap[lobbyID]
}

// update existing lobby settings
func UpdateLobby(clientID string, ID string, conf *logic.LobbyConf) (*logic.Lobby, error) {
	if clientID != LobbyMap[ID].Admin {
		return nil, errors.New("NOT_ADMIN")
	}

	if conf.Name != "" {
		LobbyMap[ID].Conf.Name = conf.Name
	}
	if conf.MaxPlayers > 0 {
		LobbyMap[ID].Conf.MaxPlayers = conf.MaxPlayers
	}
	if conf.NumAttempt > 0 {
		LobbyMap[ID].Conf.NumAttempt = conf.NumAttempt
	}
	if conf.NumRounds > 0 {
		LobbyMap[ID].Conf.NumRounds = conf.NumRounds
	}
	if conf.ScoreFactor > defaults.ScoreFactorLow && conf.ScoreFactor < defaults.ScoreFactorHigh {
		LobbyMap[ID].Conf.ScoreFactor = conf.ScoreFactor
	}
	if conf.RoundTime > 0 {
		LobbyMap[ID].Conf.RoundTime = conf.RoundTime
	}
	return LobbyMap[ID], nil
}

// adds player as map[id]name to playerlist in lobby
func AddPlayerToLobby(clientID string, clientName string, lobbyID string) {
	// if there is no lobby admin make this user one
	if LobbyMap[lobbyID].Admin == "" {
		LobbyMap[lobbyID].Admin = clientID
	}
	LobbyMap[lobbyID].PlayerList[clientID] = clientName
	LobbyMap[lobbyID].NumPlayers = len(LobbyMap[lobbyID].PlayerList)
}

// removes player map from playerlist in lobby
func RemovePlayerFromLobby(clientID string, lobbyID string) {
	delete(LobbyMap[lobbyID].PlayerList, clientID)
	// if removed player was admin & there are other players left
	// select one of them as new admin, otherwise make admin empty
	// if there are no players left delete lobby
	if LobbyMap[lobbyID].Admin == clientID && len(LobbyMap[lobbyID].PlayerList) != 0 {
		for id := range LobbyMap[lobbyID].PlayerList {
			LobbyMap[lobbyID].Admin = id
			break
		}

	} else if len(LobbyMap[lobbyID].PlayerList) == 0 {
		fmt.Println("deleting lobby")
		delete(LobbyMap, lobbyID)
	}
}

// keeps track of the location of the currently active game in lobby
// increments round counter every call
func UpdateCurrentLocation(lobbyID string, location logic.Coordinates) {
	fmt.Println("updating lobby loaction: ", lobbyID, location)
	LobbyMap[lobbyID].UsersFinished = 0
	LobbyMap[lobbyID].CurrentLocation = &location
	LobbyMap[lobbyID].CurrentRound++
	LobbyMap[lobbyID].Timer = true
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

func SubmitResult(lobbyID string, clientID string, location logic.Coordinates) (float64, int, error) {
	distance := logic.CalcDistance(*LobbyMap[lobbyID].CurrentLocation, location)
	score, error := addToResults(lobbyID, clientID, location, distance)
	return distance, score, error
}

// adds result to map of all results in lobby
func addToResults(lobbyID string, clientID string, location logic.Coordinates, distance float64) (int, error) {
	// if user currently doesnt have a result in this round create new map
	if LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound] == nil {
		LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound] = make(map[string][]logic.Results)
	}
	// if all attempts have been used up throw error
	if len(LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound][clientID]) >= LobbyMap[lobbyID].Conf.NumAttempt {
		return -1, errors.New("NO_MORE_ATTEMPTS")
	}
	// if time has expired throw an error
	if !LobbyMap[lobbyID].Timer {
		return -1, errors.New("TIMES_UP")
	}
	score := scoreDistance(distance, float64(LobbyMap[lobbyID].Conf.ScoreFactor))
	// TODO: split this monstrosity, maybe use variables
	LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound][clientID] = append(LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound][clientID], logic.Results{Location: location, Distance: distance, Score: score})

	// if this is last attempt indicate finished and check if everyone has finished
	if len(LobbyMap[lobbyID].Results[LobbyMap[lobbyID].CurrentRound][clientID]) == LobbyMap[lobbyID].Conf.NumAttempt {
		LobbyMap[lobbyID].UsersFinished++
		if LobbyMap[lobbyID].UsersFinished >= LobbyMap[lobbyID].NumPlayers {
			return score, errors.New("ROUND_FINISHED")
		}
	}
	return score, nil
}
