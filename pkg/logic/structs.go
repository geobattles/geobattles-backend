package logic

import (
	"github.com/gorilla/websocket"
)

type Coords struct {
	Lat float64 `json:"lat,omitempty"`
	Lng float64 `json:"lng,omitempty"`
}

type Results struct {
	Loc   Coords  `json:"location"`
	Dist  float64 `json:"distance"`
	Score int     `json:"score"`
}

// response from google maps metadata api
type ApiMetaResponse struct {
	Loc    Coords `json:"location"`
	Status string `json:"status"`
}

type ClientReq struct {
	Cmd  string    `json:"command"`
	Loc  *Coords   `json:"location"`
	Conf LobbyConf `json:"conf"`
}

// either Conn or Room must be provided. if Conn is set Data will be sent to this connection
// else it will be broadcast to the entire Room. Conn takes precedence over Room
type RouteMsg struct {
	Conn *websocket.Conn
	Room string
	Data ClientResp
}

type ClientResp struct {
	Status   string                       `json:"status"`
	Type     string                       `json:"type"`
	Loc      *Coords                      `json:"location,omitempty"`
	User     string                       `json:"user,omitempty"`
	AllRes   map[int]map[string][]Results `json:"results,omitempty"`
	RoundRes map[string][]Results         `json:"roundRes,omitempty"`
	GuessRes *Results                     `json:"playerRes,omitempty"`
	Round    int                          `json:"round,omitempty"`
	Lobby    *Lobby                       `json:"lobby,omitempty"`
	//Distance float64                      `json:"distance,omitempty"`
	//Score    int                          `json:"score,omitempty"`
}

type LobbyConf struct {
	Name        string   `json:"name"`
	MaxPlayers  int      `json:"maxPlayers"`
	NumAttempt  int      `json:"numAttempt"`
	NumRounds   int      `json:"numRounds"`
	RoundTime   int      `json:"roundTime"`
	ScoreFactor int      `json:"scoreFactor"`
	CCList      []string `json:"ccList"`
}

type Lobby struct {
	ID            string                       `json:"ID"`
	Admin         string                       `json:"admin"`
	Conf          *LobbyConf                   `json:"conf"`
	NumPlayers    int                          `json:"numPlayers"`
	PlayerMap     map[string]string            `json:"playerList"`
	CurrentLoc    *Coords                      `json:"-"`
	CurrentRound  int                          `json:"currentRound"`
	Results       map[int]map[string][]Results `json:"results"`
	Timer         bool                         `json:"-"`
	UsersFinished int                          `json:"-"`
	CCSize        float64                      `json:"-"`
}
