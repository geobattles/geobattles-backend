package game

import (
	"errors"
	"log/slog"
	"math"
	"time"

	"github.com/slinarji/go-geo-server/pkg/reverse"

	"github.com/slinarji/go-geo-server/pkg/db"
	"github.com/slinarji/go-geo-server/pkg/defaults"
	"github.com/slinarji/go-geo-server/pkg/logic"
	"github.com/slinarji/go-geo-server/pkg/models"

	"github.com/slinarji/go-geo-server/pkg/websocket"
)

var LobbyMap = make(map[string]*Lobby)

var colorList = [12]string{"#e6194B", "#3cb44b", "#4363d8", "#f58231", "#911eb4", "#42d4f4", "#f032e6", "#000075", "#469990", "#9A6324", "#dcbeff", "#800000"}

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

	lobbyID := logic.GenerateRndID(4)
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
		if conf.ScoreFactor < defaults.ScoreFactorLow || conf.ScoreFactor > defaults.ScoreFactorHigh {
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

// update existing lobby config
func (l *Lobby) updateConf(clientID string, conf LobbyConf) error {
	if clientID != l.Admin {
		return errors.New("NOT_ADMIN")
	}

	if l.CurrentRound != 0 {
		return errors.New("GAME_IN_PROGRESS")
	}
	slog.Debug("Updating lobby settings", "new settings", conf)

	if conf.Name != "" {
		l.Conf.Name = conf.Name
	}
	// if mode is changing apply defaults for new fields if not explicitly set
	if conf.Mode != 0 && conf.Mode != l.Conf.Mode {
		l.Conf.Mode = conf.Mode
		switch conf.Mode {
		case 2:
			l.Conf.ScoreFactor = 0

		case 1:
			if conf.ScoreFactor >= defaults.ScoreFactorLow && conf.ScoreFactor <= defaults.ScoreFactorHigh {
				l.Conf.ScoreFactor = conf.ScoreFactor
			} else {
				l.Conf.ScoreFactor = defaults.ScoreFactor
			}
		}
	}
	if conf.ScoreFactor >= defaults.ScoreFactorLow && conf.ScoreFactor <= defaults.ScoreFactorHigh {
		l.Conf.ScoreFactor = conf.ScoreFactor
	}
	if conf.Powerups != nil && len(*conf.Powerups) == 2 {
		l.Conf.Powerups = conf.Powerups
	}
	if conf.PlaceBonus != nil {
		l.Conf.PlaceBonus = conf.PlaceBonus
	}

	if conf.MaxPlayers > 0 {
		l.Conf.MaxPlayers = conf.MaxPlayers
	}
	if conf.NumAttempt > 0 {
		l.Conf.NumAttempt = conf.NumAttempt
	}
	if conf.NumRounds > 0 {
		l.Conf.NumRounds = conf.NumRounds
	}
	if conf.RoundTime > 0 {
		l.Conf.RoundTime = conf.RoundTime
	}

	if conf.CCList != nil {
		l.Conf.CCList = conf.CCList
		l.CCSize = logic.SumCCListSize(conf.CCList)
	}

	if conf.DynLives != nil {
		l.Conf.DynLives = conf.DynLives
	}
	return nil
}

func (l *Lobby) getActivePlayers() int {
	active := 0
	for _, player := range l.PlayerMap {
		if player.Connected {
			active++
		}
	}
	return active
}

// removes player from lobby, assigns new admin if necessary
func (l *Lobby) removePlayer(clientID string) {
	if player, ok := l.PlayerMap[clientID]; ok {
		player.Connected = false
	}

	// if removed player was admin & there are other players left
	// select one of them as new admin, otherwise make admin empty
	// if there are no players left delete lobby
	if l.getActivePlayers() == 0 {
		slog.Info("Deleting lobby", "lobbyID", l.ID)
		if l.RountTimer.Timer != nil {
			l.RountTimer.Timer.Stop()
		}
		delete(LobbyMap, l.ID)
	} else if l.Admin == clientID {
		for id, player := range l.PlayerMap {
			if player.Connected {
				l.Admin = id
				break
			}
		}
	}
}

func (l *Lobby) startGame(clientID string) (*models.ResponsePayload, error) {
	if clientID != l.Admin {
		return nil, errors.New("NOT_ADMIN")
	}
	if l.Active {
		return nil, errors.New("ALREADY_ACTIVE")
	}

	location, ccode := logic.RndLocation(l.Conf.CCList, l.CCSize)
	l.setupNewRound(location, ccode)
	slog.Debug("Start game timer")

	message := models.ResponsePayload{
		Loc:      &location,
		Players:  l.PlayerMap,
		PowerLog: l.PowerLogs[l.CurrentRound],
	}

	l.setupRoundTimer()
	return &message, nil
}

func (l *Lobby) getPlayerColor() string {
	var color string
cycle:
	for i := 0; i < len(colorList); i++ {
		color = colorList[i]
		for _, player := range l.PlayerMap {
			if player.Color == color {
				continue cycle
			}
		}
		break cycle
	}
	return color
}

// resets lobby state for new game
func (l *Lobby) resetLobby() {
	l.saveResults()

	l.RawResults = make(map[int]map[string][]Result)
	l.EndResults = make(map[int]map[string]*Result)
	l.TotalResults = make(map[string]*Result)

	l.PowerLogs = make(map[int][]Powerup)
	l.CurrentLoc = make([]*logic.Coords, 0, l.Conf.NumRounds)
	l.UsersFinished = make(map[string]bool)
	l.CurrentRound = 0
	// for _, player := range l.PlayerMap {
	// 	player.Powerups = *l.Conf.Powerups
	// }
	slog.Info("Lobby reset", "lobby", l)
}

// save results to db
func (l *Lobby) saveResults() {
	rounds := make([]models.Round, l.Conf.NumRounds)

	for roundIdx, roundRes := range l.EndResults {
		rounds[roundIdx-1].Loc = *l.CurrentLoc[roundIdx-1]
		rounds[roundIdx-1].Results = make([]models.Result, 0, len(roundRes))

		for userID, res := range roundRes {
			rounds[roundIdx-1].Results = append(rounds[roundIdx-1].Results, models.Result{
				UserID: userID,
				Loc:    res.Loc,
				Dist:   res.Dist,
				Total:  res.Total,
			})
		}
	}

	game := models.Game{
		Timestamp: time.Now(),
		Rounds:    rounds,
	}

	result := db.DB.Create(&game)

	if result.Error != nil {
		slog.Error("Error saving game results", "error", result.Error.Error())
	} else {
		slog.Info("Saved game results", "gameID", game.ID)
	}
}

// sets up lobby for new round and results
func (l *Lobby) setupNewRound(location logic.Coords, ccode string) {
	slog.Info("Updating lobby location", "lobbyID", l.ID, "location", location)
	l.UsersFinished = make(map[string]bool)
	l.CurrentLoc = append(l.CurrentLoc, &location)
	if l.Conf.Mode == 2 {
		l.CurrentCC = ccode
		l.StartTime = time.Now()
	}
	l.Active = true

	l.CurrentRound++
	l.RawResults[l.CurrentRound] = make(map[string][]Result)
	l.EndResults[l.CurrentRound] = make(map[string]*Result)

	for name, player := range l.PlayerMap {
		l.EndResults[l.CurrentRound][name] = &Result{}

		if l.CurrentRound == 1 {
			l.TotalResults[name] = &Result{}
			copy(player.Powerups, *l.Conf.Powerups)
			player.Lives = l.Conf.NumAttempt
		} else if *l.Conf.DynLives {
			player.Lives += (l.Conf.NumAttempt + 1) / 2
			if player.Lives > l.Conf.NumAttempt+1 {
				player.Lives = l.Conf.NumAttempt + 1
			}
		} else {
			player.Lives = l.Conf.NumAttempt
		}
	}
}

// sets up events that fire after round timer expires
func (l *Lobby) setupRoundTimer() {
	// 3 sec added to timer for frontend countdown
	l.RountTimer.Timer = time.AfterFunc(time.Second*time.Duration(l.Conf.RoundTime+3), func() {
		slog.Debug("Times up")
		// TODO: possible race condition if a user submits final guess at the same time?
		l.Active = false

		l.processRoundEnd()
	})
	l.RountTimer.End = time.Now().Add(time.Second * time.Duration(l.Conf.RoundTime+3))
}

// processes bonus points at round and / or game end
func (l *Lobby) processRoundEnd() {
	l.Hub.Broadcast <- models.ResponseBase{Status: "WRN", Type: "ROUND_FINISHED"}
	l.processBonus()
	l.processPowerups()
	l.processTotal()

	var message models.ResponsePayload
	if l.Conf.Mode == 2 {
		message = models.ResponsePayload{
			FullRoundRes: l.RawResults[l.CurrentRound],
			Round:        l.CurrentRound,
			PowerLog:     l.PowerLogs[l.CurrentRound],
			Polygon:      logic.PolyDB[l.CurrentCC],
			RoundRes:     l.EndResults[l.CurrentRound],
			TotalResults: l.TotalResults,
		}
	} else {
		message = models.ResponsePayload{
			RoundRes:     l.EndResults[l.CurrentRound],
			Round:        l.CurrentRound,
			PowerLog:     l.PowerLogs[l.CurrentRound],
			TotalResults: l.TotalResults,
		}
	}
	l.Hub.Broadcast <- models.ResponseBase{
		Status:  "OK",
		Type:    "ROUND_RESULT",
		Payload: message,
	}

	// send end of game msg and cleanup lobby
	if l.CurrentRound >= l.Conf.NumRounds {
		message := models.ResponsePayload{
			AllRes:       l.RawResults,
			TotalResults: l.TotalResults,
		}
		l.Hub.Broadcast <- models.ResponseBase{
			Status:  "OK",
			Type:    "GAME_END",
			Payload: message,
		}
		l.resetLobby()
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
func (l *Lobby) submitResult(clientID string, location logic.Coords) (float64, int, error) {
	// immediately return error if game hasnt started yet, time's up or player has no more attempts left
	if len(l.CurrentLoc) == 0 {
		return -1, -1, errors.New("GAME_NOT_ACTIVE")
	}
	if !l.Active {
		return -1, -1, errors.New("TIMES_UP")
	}
	if l.PlayerMap[clientID].Lives <= 0 {
		return -1, -1, errors.New("NO_MORE_ATTEMPTS")
	}

	// lock lobby mutex while processing, unlock after
	// can probably switch to rwmutex & move it inside processing functions to only guard actually critical sections
	l.mu.Lock()
	defer l.mu.Unlock()

	// if time has expired throw an error?
	switch l.Conf.Mode {
	case 2:
		_, err := l.processCountryGuess(clientID, location)
		return 0, 0, err
	default:
		distance := logic.CalcDistance(*l.CurrentLoc[l.CurrentRound-1], location)
		score, err := l.addToResults(clientID, location, distance)
		return distance, score, err

	}
}

// processes submitted guess in mode 1 (location guessing)
func (l *Lobby) addToResults(clientID string, location logic.Coords, distance float64) (int, error) {

	score := scoreDistance(distance, float64(l.Conf.ScoreFactor))
	// TODO: split this monstrosity, maybe use variables
	l.PlayerMap[clientID].Lives -= 1
	l.RawResults[l.CurrentRound][clientID] = append(l.RawResults[l.CurrentRound][clientID], Result{Loc: location, Dist: distance, BaseScore: score, Lives: l.PlayerMap[clientID].Lives, Attempt: len(l.RawResults[l.CurrentRound][clientID]) + 1})
	if l.EndResults[l.CurrentRound][clientID].Dist > distance || l.EndResults[l.CurrentRound][clientID].Attempt == 0 {
		slog.Debug("New best result")
		l.EndResults[l.CurrentRound][clientID] = &Result{Loc: location, Dist: distance, BaseScore: score, Attempt: len(l.RawResults[l.CurrentRound][clientID])}
	}

	// if this is last attempt indicate finished and check if everyone has finished
	if l.PlayerMap[clientID].Lives <= 0 {
		l.UsersFinished[clientID] = true

		if l.checkAllFinished() {
			return score, errors.New("ROUND_FINISHED")
		}
	}
	return score, nil
}

func (l *Lobby) checkAllFinished() bool {
	for id, player := range l.PlayerMap {
		if player.Connected && !l.UsersFinished[id] {
			return false
		}
	}
	return true
}

// processes submitted guess in mode 2 (country guessing)
func (l *Lobby) processCountryGuess(clientID string, location logic.Coords) (string, error) {
	// if user has already submitted correct guess
	if l.UsersFinished[clientID] {
		return "", errors.New("ALREADY_FINISHED")
	}
	cc, err := reverse.ReverseGeocode(location.Lng, location.Lat)
	if err != nil {
		return "", err
	}

	l.PlayerMap[clientID].Lives -= 1
	timeUsed := int(time.Since(l.StartTime).Microseconds() - 3*1000000)
	// guesses submitted within first 4s get full 5000 points
	score := int(float64(l.Conf.RoundTime*1000000-timeUsed) / float64((l.Conf.RoundTime-4)*1000000) * 5000)
	if score > 5000 {
		score = 5000
	}
	// correct result is indicated by ccode = XX and score, false result gets score = 0
	if cc == l.CurrentCC {
		l.RawResults[l.CurrentRound][clientID] = append(l.RawResults[l.CurrentRound][clientID], Result{Loc: location, BaseScore: score, Time: timeUsed, Lives: l.PlayerMap[clientID].Lives, Attempt: len(l.RawResults[l.CurrentRound][clientID]) + 1, CC: "XX"})
		l.EndResults[l.CurrentRound][clientID] = &Result{Loc: location, BaseScore: score, Time: timeUsed, Lives: l.PlayerMap[clientID].Lives, Attempt: len(l.RawResults[l.CurrentRound][clientID]), CC: "XX"}

	} else {
		l.RawResults[l.CurrentRound][clientID] = append(l.RawResults[l.CurrentRound][clientID], Result{Loc: location, BaseScore: 0, Time: timeUsed, Lives: l.PlayerMap[clientID].Lives, Attempt: len(l.RawResults[l.CurrentRound][clientID]) + 1, CC: cc})
		l.EndResults[l.CurrentRound][clientID] = &Result{Loc: location, BaseScore: 0, Time: timeUsed, Lives: l.PlayerMap[clientID].Lives, Attempt: len(l.RawResults[l.CurrentRound][clientID]), CC: cc}

	}

	// if last/correct guess mark user as finished. end round if all users have finished
	if l.PlayerMap[clientID].Lives <= 0 || l.CurrentCC == cc {
		l.UsersFinished[clientID] = true

		if l.checkAllFinished() {
			return cc, errors.New("ROUND_FINISHED")
		}
	}
	return cc, nil
}
