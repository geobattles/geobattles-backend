package game

import (
	"context"
	"sync"
	"time"

	"github.com/geobattles/geobattles-backend/pkg/logic"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
)

// Lobby is never serialized. mu owns ALL state below except immutable id/hub.
// Registry operations must be performed with mu released. Effects are encoded and
// nonblocking-enqueued in the transaction; socket closes/DB writes run afterward.
type Lobby struct {
	id             string
	hub            *websocket.Hub
	mu             sync.Mutex
	admin          string
	conf           *LobbyConf
	players        map[string]*Player
	locations      []logic.Coords
	currentCC      string
	round          int
	rawResults     map[int]map[string][]Result
	endResults     map[int]map[string]*Result
	totalResults   map[string]*Result
	active         bool
	finished       map[string]bool
	ccSize         float64
	powerLogs      map[int][]Powerup
	startTime      time.Time
	roundTimer     *time.Timer
	deadline       time.Time
	phase          string
	gameID         string
	seq            uint64
	closed         bool
	starting       bool
	generation     uint64
	roundGen       uint64
	finalizedGen   uint64
	startCancel    context.CancelFunc
	startInitiator *websocket.Client
	leases         map[string]*playerLease
	reservations   map[string]*Admission
	nextLease      uint64
	nextMember     uint64
	emptyTimer     *time.Timer
	persisted      bool
}

const (
	PhaseWaiting  = "WAITING"
	PhasePlaying  = "PLAYING"
	PhaseMidRound = "MID-ROUND"
	PhaseFinished = "FINISHED"
)

type playerLease struct {
	client     *websocket.Client
	generation uint64
	present    uint64 // successful membership order; preserved through recovery
	grace      *time.Timer
	expires    time.Time
}

type LobbyConf struct {
	Name         string   `json:"name"`
	Mode         int      `json:"mode"`
	MaxPlayers   int      `json:"maxPlayers"`
	NumAttempt   int      `json:"numAttempt"`
	NumRounds    int      `json:"numRounds"`
	RoundTime    int      `json:"roundTime"`
	ScoreFactor  int      `json:"scoreFactor,omitempty"`
	ScoringMode  string   `json:"scoringMode"`
	ScoreScaleKm float64  `json:"scoreScaleKm"` // read-only effective half-score decay distance
	CCList       []string `json:"ccList"`
	Powerups     *[]bool  `json:"powerups,omitempty"`
	PlaceBonus   *bool    `json:"placeBonus,omitempty"`
	DynLives     *bool    `json:"dynLives"`
}

type Player struct {
	Name      string `json:"name"`
	Connected bool   `json:"connected"`
	Ready     bool   `json:"ready"`
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
	Ready   *bool         `json:"ready,omitempty"`
	Loc     *logic.Coords `json:"location,omitempty"`
	Conf    LobbyConf     `json:"conf,omitempty"`
	Powerup Powerup       `json:"powerup,omitempty"`
}
