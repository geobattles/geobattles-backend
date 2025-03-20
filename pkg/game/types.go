package game

import (
	"sync"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
)

type Lobby struct {
	ID            string                      `json:"ID"`
	Hub           *websocket.Hub              `json:"-"`
	Admin         string                      `json:"admin"`
	Conf          *LobbyConf                  `json:"conf"`
	NumPlayers    int                         `json:"numPlayers"`
	PlayerMap     map[string]*Player          `json:"playerList"`
	CurrentLoc    []*logic.Coords             `json:"-"`
	CurrentCC     string                      `json:"-"`
	CurrentRound  int                         `json:"currentRound"`
	RawResults    map[int]map[string][]Result `json:"results"`
	EndResults    map[int]map[string]*Result  `json:"endResults"`
	TotalResults  map[string]*Result          `json:"totalResults"`
	Active        bool                        `json:"-"`
	UsersFinished map[string]bool             `json:"-"`
	CCSize        float64                     `json:"-"`
	PowerLogs     map[int][]Powerup           `json:"-"`
	StartTime     time.Time                   `json:"-"`
	RountTimer    RoundTimer                  `json:"-"`
	mu            sync.Mutex                  `json:"-"`
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
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	Color     string `json:"color"`
	Powerups  []bool `json:"powerups,omitempty"`
	Lives     int    `json:"lives,omitempty"`
}

type Result struct {
	Loc         logic.Coords `json:"location"`
	Dist        float64      `json:"distance"`
	BaseScore   int          `json:"baseScr"`
	PlaceScore  int          `json:"placeScr,omitempty"`
	DoubleScore int          `json:"dblScr,omitempty"`
	DuelScore   int          `json:"duelScr,omitempty"`
	Total       int          `json:"total,omitempty"`
	Attempt     int          `json:"attempt"`
	Lives       int          `json:"lives"`
	CC          string       `json:"cc,omitempty"`
	Time        int          `json:"time,omitempty"`
}

type Powerup struct {
	Type   int    `json:"type"`
	Source string `json:"source"`
	Target string `json:"target,omitempty"`
}

type ClientReq struct {
	Cmd     string        `json:"command"`
	Loc     *logic.Coords `json:"location,omitempty"`
	Conf    LobbyConf     `json:"conf,omitempty"`
	Powerup Powerup       `json:"powerup,omitempty"`
}

type RoundTimer struct {
	Timer *time.Timer
	End   time.Time
}
