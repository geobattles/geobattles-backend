package logic

import (
	"github.com/gorilla/websocket"
)

type Coordinates struct {
	Latitude  float64 `json:"lat,omitempty"`
	Longitude float64 `json:"lng,omitempty"`
}

type Results struct {
	Location Coordinates `json:"location"`
	Distance float64     `json:"distance"`
	Score    int         `json:"score"`
}

// response from google maps metadata api
type MetadataResponse struct {
	Location Coordinates `json:"location"`
	Status   string
}

type ClientReq struct {
	Command  string       `json:"command"`
	Location *Coordinates `json:"location"`
	Conf     *LobbyConf   `json:"conf,omitempty"`
}

// either Conn or Room must be provided
// if Conn is set Data will be sent to this connection
// else it will be broadcast to the entire Room
// Conn takes precedence over Room
type Message struct {
	Conn *websocket.Conn
	Room string
	Data ResponseMsg
}

type ResponseMsg struct {
	Status   string                       `json:"status"`
	Type     string                       `json:"type"`
	Location *Coordinates                 `json:"location,omitempty"`
	User     string                       `json:"user,omitempty"`
	Distance float64                      `json:"distance,omitempty"`
	Score    int                          `json:"score,omitempty"`
	Results  map[int]map[string][]Results `json:"results,omitempty"`
	RoundRes map[string][]Results         `json:"roundRes,omitempty"`
	Lobby    *Lobby                       `json:"lobby,omitempty"`
}

type LobbyConf struct {
	Name        string `json:"name"`
	MaxPlayers  int    `json:"maxPlayers"`
	NumAttempt  int    `json:"numAttempt"`
	NumRounds   int    `json:"numRounds"`
	RoundTime   int    `json:"roundTime"`
	ScoreFactor int    `json:"scoreFactor"`
}

type Lobby struct {
	ID              string                       `json:"ID"`
	Admin           string                       `json:"admin"`
	Conf            *LobbyConf                   `json:"conf"`
	NumPlayers      int                          `json:"numPlayers"`
	PlayerList      map[string]string            `json:"playerList"`
	CurrentLocation *Coordinates                 `json:"-"`
	CurrentRound    int                          `json:"currentRound"`
	Results         map[int]map[string][]Results `json:"results"`
	Timer           bool                         `json:"-"`
	UsersFinished   int                          `json:"-"`
}
