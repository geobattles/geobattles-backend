package logic

import (
	"encoding/json"

	"github.com/gorilla/websocket"
)

type Coords struct {
	Lat float64 `json:"lat,omitempty"`
	Lng float64 `json:"lng,omitempty"`
}

type Results struct {
	Loc     Coords  `json:"location"`
	Dist    float64 `json:"distance"`
	Score   int     `json:"score"`
	Attempt int     `json:"attempt"`
	Lives   int     `json:"lives"`
	CC      string  `json:"cc,omitempty"`
}

// response from google maps metadata api
type ApiMetaResponse struct {
	Loc    Coords `json:"location"`
	Status string `json:"status"`
}

type ClientReq struct {
	Cmd     string    `json:"command"`
	Loc     *Coords   `json:"location"`
	CC      string    `json:"cc,omitempty"`
	Conf    LobbyConf `json:"conf"`
	Powerup Powerup   `json:"powerup"`
}

// either Conn or Room must be provided. if Conn is set Data will be sent to this connection
type ClientResp struct {
	Status string                       `json:"status"`
	Type   string                       `json:"type"`
	Loc    *Coords                      `json:"location,omitempty"`
	User   string                       `json:"user,omitempty"`
	AllRes map[int]map[string][]Results `json:"results,omitempty"`
	// RoundRes map[string][]Results         `json:"roundRes,omitempty"`
	RoundRes map[string]*Results `json:"roundRes,omitempty"`
	GuessRes *Results            `json:"playerRes,omitempty"`
	Round    int                 `json:"round,omitempty"`
	CC       string              `json:"cc,omitempty"`
	Lobby    *Lobby              `json:"lobby,omitempty"`
	PowerLog []Powerup           `json:"powerLog,omitempty"`
	Players  map[string]*Player  `json:"players,omitempty"`
	Polygon  json.RawMessage     `json:"polygon,omitempty"`
}

// else it will be broadcast to the entire Room. Conn takes precedence over Room
type RouteMsg struct {
	Conn *websocket.Conn
	Room string
	Data ClientResp
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
	Active        bool                         `json:"-"`
	UsersFinished int                          `json:"-"`
	CCSize        float64                      `json:"-"`
	PowerLogs     map[int][]Powerup            `json:"-"`
}
