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
	"U4YPR6": {ID: "U4YPR6", Conf: &logic.LobbyConf{Name: "prvi lobby", MaxPlayers: 8, ScoreFactor: 100, NumAttempt: 3, NumRounds: 2, RoundTime: 30, CCList: []string{}, Powerups: defaults.Powerups(), PlaceBonus: defaults.PlaceBonus()}, NumPlayers: 0, PlayerMap: make(map[string]*logic.Player), RawResults: make(map[int]map[string][]logic.Results), EndResults: make(map[int]map[string]*logic.Results), PowerLogs: make(map[int][]logic.Powerup)},
}

var ColorList = [12]string{"#e6194B", "#3cb44b", "#4363d8", "#f58231", "#911eb4", "#42d4f4", "#f032e6", "#000075", "#469990", "#9A6324", "#dcbeff", "#800000"}

// validates values and creates new lobby
func CreateLobby(conf logic.LobbyConf) *logic.Lobby {
	var newLobby logic.Lobby
	newLobby.Conf = &logic.LobbyConf{}
	newLobby.PlayerMap = make(map[string]*logic.Player)
	newLobby.RawResults = make(map[int]map[string][]logic.Results)
	newLobby.EndResults = make(map[int]map[string]*logic.Results)

	newLobby.PowerLogs = make(map[int][]logic.Powerup)
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

	if len(conf.CCList) == 0 {
		newLobby.Conf.CCList = []string{}
	} else {
		newLobby.Conf.CCList = conf.CCList
		newLobby.CCSize = logic.SumCCListSize(conf.CCList)
	}

	if conf.Powerups != nil && len(*conf.Powerups) == 2 {
		newLobby.Conf.Powerups = conf.Powerups
	} else {
		newLobby.Conf.Powerups = defaults.Powerups()
	}

	if conf.PlaceBonus != nil {
		newLobby.Conf.PlaceBonus = conf.PlaceBonus
	} else {
		newLobby.Conf.PlaceBonus = defaults.PlaceBonus()
	}

	LobbyMap[lobbyID] = &newLobby
	return LobbyMap[lobbyID]
}

// update existing lobby settings
func UpdateLobby(clientID string, ID string, conf logic.LobbyConf) (*logic.Lobby, error) {
	if clientID != LobbyMap[ID].Admin {
		return nil, errors.New("NOT_ADMIN")
	}
	if LobbyMap[ID].CurrentRound != 0 {
		return nil, errors.New("GAME_IN_PROGRESS")
	}
	fmt.Println("update", conf)

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

	if conf.CCList != nil {
		LobbyMap[ID].Conf.CCList = conf.CCList
		LobbyMap[ID].CCSize = logic.SumCCListSize(conf.CCList)
	}

	if conf.Powerups != nil && len(*conf.Powerups) == 2 {
		LobbyMap[ID].Conf.Powerups = conf.Powerups
	}

	if conf.PlaceBonus != nil {
		LobbyMap[ID].Conf.PlaceBonus = conf.PlaceBonus
	}
	return LobbyMap[ID], nil
}

func genPlayerColor(lobbyID string) string {
	var color string
cycle:
	for i := 0; i < len(ColorList); i++ {
		color = ColorList[i]
		for _, player := range LobbyMap[lobbyID].PlayerMap {
			if player.Color == color {
				continue cycle
			}
		}
		break cycle
	}
	return color
}

// adds player as map[id]name to playerlist in lobby
func AddPlayerToLobby(clientID string, clientName string, lobbyID string) {
	// if there is no lobby admin make this user one
	if LobbyMap[lobbyID].Admin == "" {
		LobbyMap[lobbyID].Admin = clientID
	}
	// LobbyMap[lobbyID].PlayerMap[clientID] = &logic.Player{Name: clientName, Color: genPlayerColor(lobbyID), Powerups: *LobbyMap[lobbyID].Conf.Powerups}
	LobbyMap[lobbyID].PlayerMap[clientID] = &logic.Player{Name: clientName, Color: genPlayerColor(lobbyID)}
	LobbyMap[lobbyID].NumPlayers = len(LobbyMap[lobbyID].PlayerMap)
}

// removes player map from playerlist in lobby
func RemovePlayerFromLobby(clientID string, lobbyID string) {
	delete(LobbyMap[lobbyID].PlayerMap, clientID)
	LobbyMap[lobbyID].NumPlayers = len(LobbyMap[lobbyID].PlayerMap)
	// if removed player was admin & there are other players left
	// select one of them as new admin, otherwise make admin empty
	// if there are no players left delete lobby
	if LobbyMap[lobbyID].Admin == clientID && LobbyMap[lobbyID].NumPlayers != 0 {
		for id := range LobbyMap[lobbyID].PlayerMap {
			LobbyMap[lobbyID].Admin = id
			break
		}

	} else if LobbyMap[lobbyID].NumPlayers == 0 {
		fmt.Println("deleting lobby")
		delete(LobbyMap, lobbyID)
	}
}

func ResetLobby(lobbyID string) {
	LobbyMap[lobbyID].RawResults = make(map[int]map[string][]logic.Results)
	LobbyMap[lobbyID].EndResults = make(map[int]map[string]*logic.Results)

	LobbyMap[lobbyID].PowerLogs = make(map[int][]logic.Powerup)
	LobbyMap[lobbyID].CurrentLoc = nil
	LobbyMap[lobbyID].UsersFinished = 0
	LobbyMap[lobbyID].CurrentRound = 0
	for _, player := range LobbyMap[lobbyID].PlayerMap {
		player.Powerups = *LobbyMap[lobbyID].Conf.Powerups
	}
	fmt.Println("Lobby after reset ", LobbyMap[lobbyID])
}

// keeps track of the location of the currently active game in lobby
// increments round counter every call
func UpdateCurrentLocation(lobbyID string, location logic.Coords) {
	fmt.Println("updating lobby loaction: ", lobbyID, location)
	if LobbyMap[lobbyID].CurrentRound == 0 {
		for _, player := range LobbyMap[lobbyID].PlayerMap {
			player.Powerups = *LobbyMap[lobbyID].Conf.Powerups
		}
	}
	LobbyMap[lobbyID].UsersFinished = 0
	LobbyMap[lobbyID].CurrentLoc = &location
	LobbyMap[lobbyID].CurrentRound++
	LobbyMap[lobbyID].Active = true
}

// calculate score based on distance and scorefactor
// scorefactor determines how fast score drops off
// for example a low scorefactor means if your guess is 100km off you only get 100 points
// it also means you get full points for distance closer that scorefactor
func scoreDistance(x float64, a float64) int {
	score := int(5000 * math.Pow(0.999, (x/1000-a/1000)*(0.2+30*(math.Pow(0.98, 10*math.Pow(a, 0.6))))))

	if score > 5000 {
		return 5000
	}
	return score
}

// calculates distance and score and adds them to the results
func SubmitResult(lobbyID string, clientID string, location logic.Coords) (float64, int, error) {
	// immediately return error if game hasnt started yet, times up or player has no more attempts left
	if LobbyMap[lobbyID].CurrentLoc == nil {
		return -1, -1, errors.New("GAME_NOT_ACTIVE")
	}
	if !LobbyMap[lobbyID].Active {
		return -1, -1, errors.New("TIMES_UP")
	}
	if len(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID]) >= LobbyMap[lobbyID].Conf.NumAttempt {
		return -1, -1, errors.New("NO_MORE_ATTEMPTS")
	}
	// if time has expired throw an error
	distance := logic.CalcDistance(*LobbyMap[lobbyID].CurrentLoc, location)
	score, error := addToResults(lobbyID, clientID, location, distance)
	return distance, score, error
}

// adds result to map of all results in lobby
func addToResults(lobbyID string, clientID string, location logic.Coords, distance float64) (int, error) {
	// if user currently doesnt have a result in this round create new map
	if LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound] == nil {
		LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound] = make(map[string][]logic.Results)
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound] = make(map[string]*logic.Results)

	}
	score := scoreDistance(distance, float64(LobbyMap[lobbyID].Conf.ScoreFactor))
	// TODO: split this monstrosity, maybe use variables
	LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID] = append(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID], logic.Results{Loc: location, Dist: distance, Score: score, Attempt: len(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID])})
	if LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][clientID] == nil {
		fmt.Println("FIRST RESULT")
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][clientID] = &logic.Results{Loc: location, Dist: distance, Score: score, Attempt: len(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID])}
	} else if LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][clientID].Dist > distance {
		fmt.Println("NEW BEST RESULT")
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][clientID] = &logic.Results{Loc: location, Dist: distance, Score: score, Attempt: len(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID])}
	} else {
		fmt.Println("Current DISTANCE", LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][clientID].Dist)
		fmt.Println("Current2 DISTANCE", LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][clientID])
	}

	// if this is last attempt indicate finished and check if everyone has finished
	if len(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID]) == LobbyMap[lobbyID].Conf.NumAttempt {
		LobbyMap[lobbyID].UsersFinished++
		if LobbyMap[lobbyID].UsersFinished >= LobbyMap[lobbyID].NumPlayers {
			return score, errors.New("ROUND_FINISHED")
		}
	}
	return score, nil
}
