package models

import (
	"encoding/json"
	"time"
)

type Coords struct {
	Lat float64 `json:"lat,omitempty"`
	Lng float64 `json:"lng,omitempty"`
}

type Results struct {
	Loc         Coords  `json:"location"`
	Dist        float64 `json:"distance"`
	BaseScore   int     `json:"baseScr"`
	PlaceScore  int     `json:"placeScr,omitempty"`
	DoubleScore int     `json:"dblScr,omitempty"`
	DuelScore   int     `json:"duelScr,omitempty"`
	Total       int     `json:"total,omitempty"`
	Attempt     int     `json:"attempt"`
	Lives       int     `json:"lives"`
	CC          string  `json:"cc,omitempty"`
	Time        int     `json:"time,omitempty"`
}

type Powerup struct {
	Type   int    `json:"type"`
	Source string `json:"source"`
	Target string `json:"target,omitempty"`
}

type LobbyConf struct {
	Name        string   `json:"name"`
	Mode        int      `json:"mode"`
	MaxPlayers  int      `json:"maxPlayers"`
	NumAttempt  int      `json:"numAttempt"`
	NumRounds   int      `json:"numRounds"`
	RoundTime   int      `json:"roundTime"`
	ScoreFactor int      `json:"scoreFactor,omitempty"`
	CCList      []string `json:"ccList"`
	Powerups    *[]bool  `json:"powerups,omitempty"`
	PlaceBonus  *bool    `json:"placeBonus,omitempty"`
	DynLives    *bool    `json:"dynLives"`
}

type Player struct {
	Name     string `json:"name"`
	Color    string `json:"color"`
	Powerups []bool `json:"powerups,omitempty"`
	Lives    int    `json:"lives,omitempty"`
}

type Lobby struct {
	ID            string                       `json:"ID"`
	Admin         string                       `json:"admin"`
	Conf          *LobbyConf                   `json:"conf"`
	NumPlayers    int                          `json:"numPlayers"`
	PlayerMap     map[string]*Player           `json:"playerList"`
	CurrentLoc    *Coords                      `json:"-"`
	CurrentCC     string                       `json:"-"`
	CurrentRound  int                          `json:"currentRound"`
	RawResults    map[int]map[string][]Results `json:"results"`
	EndResults    map[int]map[string]*Results  `json:"endResults"`
	TotalResults  map[string]*Results          `json:"totalResults"`
	Active        bool                         `json:"-"`
	UsersFinished map[string]bool              `json:"-"`
	CCSize        float64                      `json:"-"`
	PowerLogs     map[int][]Powerup            `json:"-"`
	StartTime     time.Time                    `json:"-"`
	Timer         *time.Timer                  `json:"-"`
}

type ClientResp struct {
	Status       string                       `json:"status"`
	Type         string                       `json:"type"`
	Loc          *Coords                      `json:"location,omitempty"`
	User         string                       `json:"user,omitempty"`
	AllRes       map[int]map[string][]Results `json:"results,omitempty"`
	FullRoundRes map[string][]Results         `json:"fullroundRes,omitempty"`
	RoundRes     map[string]*Results          `json:"roundRes,omitempty"`
	TotalResults map[string]*Results          `json:"totalResults,omitempty"`
	GuessRes     *Results                     `json:"playerRes,omitempty"`
	Round        int                          `json:"round,omitempty"`
	CC           string                       `json:"cc,omitempty"`
	Lobby        *Lobby                       `json:"lobby,omitempty"`
	PowerLog     []Powerup                    `json:"powerLog,omitempty"`
	Players      map[string]*Player           `json:"players,omitempty"`
	Polygon      json.RawMessage              `json:"polygon,omitempty"`
}

type ResponseBase struct {
	Status  string      `json:"status"`
	Type    string      `json:"type"`
	Payload interface{} `json:"payload,omitempty"`
}

type ResponsePayload struct {
	Lobby        interface{}     `json:"lobby,omitempty"`
	Loc          interface{}     `json:"location,omitempty"`
	User         string          `json:"user,omitempty"`
	AllRes       interface{}     `json:"results,omitempty"`
	FullRoundRes interface{}     `json:"fullroundRes,omitempty"`
	RoundRes     interface{}     `json:"roundRes,omitempty"`
	TotalResults interface{}     `json:"totalResults,omitempty"`
	GuessRes     interface{}     `json:"playerRes,omitempty"`
	Round        int             `json:"round,omitempty"`
	CC           string          `json:"cc,omitempty"`
	PowerLog     interface{}     `json:"powerLog,omitempty"`
	Players      interface{}     `json:"players,omitempty"`
	Polygon      json.RawMessage `json:"polygon,omitempty"`
}
