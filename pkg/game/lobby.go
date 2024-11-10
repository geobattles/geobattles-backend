package game

import (
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/slinarji/go-geo-server/pkg/reverse"

	"github.com/slinarji/go-geo-server/pkg/defaults"
	"github.com/slinarji/go-geo-server/pkg/logic"
	"github.com/slinarji/go-geo-server/pkg/websocket"
)

var LobbyMap = make(map[string]*Lobby)

func init() {
	slog.Info("Creating initial lobby")
	hub := websocket.NewHub()
	go hub.Start()
	LobbyMap["U4YPR6"] = &Lobby{ID: "U4YPR6", Hub: hub, Conf: &LobbyConf{Name: "prvi lobby", Mode: 1, MaxPlayers: 8, ScoreFactor: 100, NumAttempt: 3, NumRounds: 2, RoundTime: 30, CCList: []string{}, Powerups: defaults.Powerups(), PlaceBonus: defaults.PlaceBonus(), DynLives: defaults.DynLives()}, NumPlayers: 0, PlayerMap: make(map[string]*Player), RawResults: make(map[int]map[string][]Result), EndResults: make(map[int]map[string]*Result), TotalResults: make(map[string]*Result), PowerLogs: make(map[int][]Powerup)}
}

// initial lobby list for debugging
// var LobbyMap = map[string]*Lobby{
// 	"U4YPR6": {ID: "U4YPR6", Conf: &LobbyConf{Name: "prvi lobby", Mode: 1, MaxPlayers: 8, ScoreFactor: 100, NumAttempt: 3, NumRounds: 2, RoundTime: 30, CCList: []string{}, Powerups: defaults.Powerups(), PlaceBonus: defaults.PlaceBonus(), DynLives: defaults.DynLives()}, NumPlayers: 0, PlayerMap: make(map[string]*Player), RawResults: make(map[int]map[string][]Result), EndResults: make(map[int]map[string]*Result), TotalResults: make(map[string]*Result), PowerLogs: make(map[int][]Powerup)},
// }

var ColorList = [12]string{"#e6194B", "#3cb44b", "#4363d8", "#f58231", "#911eb4", "#42d4f4", "#f032e6", "#000075", "#469990", "#9A6324", "#dcbeff", "#800000"}

// validates values and creates new lobby
func CreateLobby(conf LobbyConf) *Lobby {
	var newLobby Lobby

	hub := websocket.NewHub()
	go hub.Start()
	newLobby.Hub = hub

	newLobby.Conf = &LobbyConf{}
	newLobby.PlayerMap = make(map[string]*Player)
	newLobby.RawResults = make(map[int]map[string][]Result)
	newLobby.EndResults = make(map[int]map[string]*Result)
	newLobby.TotalResults = make(map[string]*Result)
	newLobby.PowerLogs = make(map[int][]Powerup)

	lobbyID := logic.GenerateRndID(6)
	newLobby.ID = lobbyID
	// validate values and set defaults otherwise
	if conf.Name == "" {
		newLobby.Conf.Name = lobbyID
	} else {
		newLobby.Conf.Name = conf.Name
	}

	switch conf.Mode {
	case 2:
		newLobby.Conf.Mode = conf.Mode
	default:
		newLobby.Conf.Mode = defaults.Mode
		if conf.ScoreFactor == 0 || conf.ScoreFactor < defaults.ScoreFactorLow || conf.ScoreFactor > defaults.ScoreFactorHigh {
			newLobby.Conf.ScoreFactor = defaults.ScoreFactor
		} else {
			newLobby.Conf.ScoreFactor = conf.ScoreFactor
		}
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

	if conf.DynLives != nil {
		newLobby.Conf.DynLives = conf.DynLives
	} else {
		newLobby.Conf.DynLives = defaults.DynLives()
	}

	LobbyMap[lobbyID] = &newLobby
	return LobbyMap[lobbyID]
}

// update existing lobby settings
func UpdateLobby(clientID string, ID string, conf LobbyConf) (*Lobby, error) {
	if clientID != LobbyMap[ID].Admin {
		return nil, errors.New("NOT_ADMIN")
	}
	if LobbyMap[ID].CurrentRound != 0 {
		return nil, errors.New("GAME_IN_PROGRESS")
	}
	fmt.Println("update", conf)
	slog.Info("Updating lobby settings", "new settings", conf)

	if conf.Name != "" {
		LobbyMap[ID].Conf.Name = conf.Name
	}
	// if mode is changing apply defaults for new fields if not explicitly set
	if conf.Mode != 0 && conf.Mode != LobbyMap[ID].Conf.Mode {
		LobbyMap[ID].Conf.Mode = conf.Mode
		switch conf.Mode {
		case 2:
			LobbyMap[ID].Conf.ScoreFactor = 0

		case 1:
			if conf.ScoreFactor > defaults.ScoreFactorLow && conf.ScoreFactor < defaults.ScoreFactorHigh {
				LobbyMap[ID].Conf.ScoreFactor = conf.ScoreFactor
			} else {
				LobbyMap[ID].Conf.ScoreFactor = defaults.ScoreFactor
			}
		}
	}
	if conf.ScoreFactor > defaults.ScoreFactorLow && conf.ScoreFactor < defaults.ScoreFactorHigh {
		LobbyMap[ID].Conf.ScoreFactor = conf.ScoreFactor
	}
	if conf.Powerups != nil && len(*conf.Powerups) == 2 {
		LobbyMap[ID].Conf.Powerups = conf.Powerups
	}
	if conf.PlaceBonus != nil {
		LobbyMap[ID].Conf.PlaceBonus = conf.PlaceBonus
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
	if conf.RoundTime > 0 {
		LobbyMap[ID].Conf.RoundTime = conf.RoundTime
	}

	if conf.CCList != nil {
		LobbyMap[ID].Conf.CCList = conf.CCList
		LobbyMap[ID].CCSize = logic.SumCCListSize(conf.CCList)
	}

	if conf.DynLives != nil {
		LobbyMap[ID].Conf.DynLives = conf.DynLives
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

func ResetLobby(lobbyID string) {
	LobbyMap[lobbyID].RawResults = make(map[int]map[string][]Result)
	LobbyMap[lobbyID].EndResults = make(map[int]map[string]*Result)
	LobbyMap[lobbyID].TotalResults = make(map[string]*Result)

	LobbyMap[lobbyID].PowerLogs = make(map[int][]Powerup)
	LobbyMap[lobbyID].CurrentLoc = nil
	LobbyMap[lobbyID].UsersFinished = make(map[string]bool)
	LobbyMap[lobbyID].CurrentRound = 0
	// for _, player := range LobbyMap[lobbyID].PlayerMap {
	// 	player.Powerups = *LobbyMap[lobbyID].Conf.Powerups
	// }
	fmt.Println("Lobby after reset ", LobbyMap[lobbyID])
	slog.Info("Lobby reset", "lobbyMap", LobbyMap[lobbyID])
}

// keeps track of the location of the currently active game in lobby
// increments round counter every call
func UpdateCurrentLocation(lobbyID string, location logic.Coords, ccode string) {
	fmt.Println("updating lobby loaction: ", lobbyID, location)
	slog.Info("Updating lobby location", "lobbyID", lobbyID, "location", location)
	LobbyMap[lobbyID].UsersFinished = make(map[string]bool)
	LobbyMap[lobbyID].CurrentLoc = &location
	if LobbyMap[lobbyID].Conf.Mode == 2 {
		LobbyMap[lobbyID].CurrentCC = ccode
		LobbyMap[lobbyID].StartTime = time.Now()
	}
	LobbyMap[lobbyID].Active = true

	LobbyMap[lobbyID].CurrentRound++
	LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound] = make(map[string][]Result)
	LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound] = make(map[string]*Result)

	for name, player := range LobbyMap[lobbyID].PlayerMap {
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][name] = &Result{}

		if LobbyMap[lobbyID].CurrentRound == 1 {
			LobbyMap[lobbyID].TotalResults[name] = &Result{}
			copy(player.Powerups, *LobbyMap[lobbyID].Conf.Powerups)
			player.Lives = LobbyMap[lobbyID].Conf.NumAttempt
		} else if *LobbyMap[lobbyID].Conf.DynLives {
			player.Lives += (LobbyMap[lobbyID].Conf.NumAttempt + 1) / 2
			if player.Lives > LobbyMap[lobbyID].Conf.NumAttempt+1 {
				player.Lives = LobbyMap[lobbyID].Conf.NumAttempt + 1
			}
		} else {
			player.Lives = LobbyMap[lobbyID].Conf.NumAttempt
		}
	}
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
	if LobbyMap[lobbyID].PlayerMap[clientID].Lives <= 0 {
		return -1, -1, errors.New("NO_MORE_ATTEMPTS")
	}
	// if time has expired throw an error
	switch LobbyMap[lobbyID].Conf.Mode {
	case 2:
		_, err := processCountryGuess(lobbyID, clientID, location)
		return 0, 0, err
	default:
		distance := logic.CalcDistance(*LobbyMap[lobbyID].CurrentLoc, location)
		score, error := addToResults(lobbyID, clientID, location, distance)
		return distance, score, error

	}
}

// processes submitted guess in mode 1 (location guessing)
func addToResults(lobbyID string, clientID string, location logic.Coords, distance float64) (int, error) {

	score := scoreDistance(distance, float64(LobbyMap[lobbyID].Conf.ScoreFactor))
	// TODO: split this monstrosity, maybe use variables
	LobbyMap[lobbyID].PlayerMap[clientID].Lives -= 1
	LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID] = append(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID], Result{Loc: location, Dist: distance, BaseScore: score, Lives: LobbyMap[lobbyID].PlayerMap[clientID].Lives, Attempt: len(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID]) + 1})
	if LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][clientID].Dist > distance || LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][clientID].Attempt == 0 {
		fmt.Println("NEW BEST RESULT")
		slog.Info("New best result")
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][clientID] = &Result{Loc: location, Dist: distance, BaseScore: score, Attempt: len(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID])}
	}

	// if this is last attempt indicate finished and check if everyone has finished
	if LobbyMap[lobbyID].PlayerMap[clientID].Lives <= 0 {
		LobbyMap[lobbyID].UsersFinished[clientID] = true
		if len(LobbyMap[lobbyID].UsersFinished) >= LobbyMap[lobbyID].NumPlayers {
			return score, errors.New("ROUND_FINISHED")
		}
	}
	return score, nil
}

// processes submitted guess in mode 2 (country guessing)
func processCountryGuess(lobbyID string, clientID string, location logic.Coords) (string, error) {
	// if user has already submitted correct guess
	if LobbyMap[lobbyID].UsersFinished[clientID] {
		return "", errors.New("ALREADY_FINISHED")
	}
	cc, err := reverse.ReverseGeocode(location.Lng, location.Lat)
	if err != nil {
		return "", err
	}

	LobbyMap[lobbyID].PlayerMap[clientID].Lives -= 1
	timeUsed := int(time.Since(LobbyMap[lobbyID].StartTime).Microseconds() - 3*1000000)
	// guesses submitted within first 4s get full 5000 points
	score := int(float64(LobbyMap[lobbyID].Conf.RoundTime*1000000-timeUsed) / float64((LobbyMap[lobbyID].Conf.RoundTime-4)*1000000) * 5000)
	if score > 5000 {
		score = 5000
	}
	// correct result is indicated by ccode = XX and score, false result gets score = 0
	if cc == LobbyMap[lobbyID].CurrentCC {
		LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID] = append(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID], Result{Loc: location, BaseScore: score, Time: timeUsed, Lives: LobbyMap[lobbyID].PlayerMap[clientID].Lives, Attempt: len(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID]) + 1, CC: "XX"})
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][clientID] = &Result{Loc: location, BaseScore: score, Time: timeUsed, Lives: LobbyMap[lobbyID].PlayerMap[clientID].Lives, Attempt: len(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID]), CC: "XX"}

	} else {
		LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID] = append(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID], Result{Loc: location, BaseScore: 0, Time: timeUsed, Lives: LobbyMap[lobbyID].PlayerMap[clientID].Lives, Attempt: len(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID]) + 1, CC: cc})
		LobbyMap[lobbyID].EndResults[LobbyMap[lobbyID].CurrentRound][clientID] = &Result{Loc: location, BaseScore: 0, Time: timeUsed, Lives: LobbyMap[lobbyID].PlayerMap[clientID].Lives, Attempt: len(LobbyMap[lobbyID].RawResults[LobbyMap[lobbyID].CurrentRound][clientID]), CC: cc}

	}

	// if last/correct guess mark user as finished. end round if all users have finished
	if LobbyMap[lobbyID].PlayerMap[clientID].Lives <= 0 || LobbyMap[lobbyID].CurrentCC == cc {
		LobbyMap[lobbyID].UsersFinished[clientID] = true
		if len(LobbyMap[lobbyID].UsersFinished) >= LobbyMap[lobbyID].NumPlayers {
			return cc, errors.New("ROUND_FINISHED")
		}
	}
	return cc, nil
}
