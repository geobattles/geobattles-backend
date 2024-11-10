package logic

type Coords struct {
	Lat float64 `json:"lat,omitempty"`
	Lng float64 `json:"lng,omitempty"`
}

// type Results struct {
// 	Loc         Coords  `json:"location"`
// 	Dist        float64 `json:"distance"`
// 	BaseScore   int     `json:"baseScr"`
// 	PlaceScore  int     `json:"placeScr,omitempty"`
// 	DoubleScore int     `json:"dblScr,omitempty"`
// 	DuelScore   int     `json:"duelScr,omitempty"`
// 	Total       int     `json:"total,omitempty"`
// 	Attempt     int     `json:"attempt"`
// 	Lives       int     `json:"lives"`
// 	CC          string  `json:"cc,omitempty"`
// 	Time        int     `json:"time,omitempty"`
// }

// response from google maps metadata api
type ApiMetaResponse struct {
	Loc    Coords `json:"location"`
	Status string `json:"status"`
}

// TODO: moved to models
// either Conn or Room must be provided. if Conn is set Data will be sent to this connection
// type ClientResp struct {
// 	Status       string                       `json:"status"`
// 	Type         string                       `json:"type"`
// 	Loc          *Coords                      `json:"location,omitempty"`
// 	User         string                       `json:"user,omitempty"`
// 	AllRes       map[int]map[string][]Results `json:"results,omitempty"`
// 	FullRoundRes map[string][]Results         `json:"fullroundRes,omitempty"`
// 	RoundRes     map[string]*Results          `json:"roundRes,omitempty"`
// 	TotalResults map[string]*Results          `json:"totalResults,omitempty"`
// 	GuessRes     *Results                     `json:"playerRes,omitempty"`
// 	Round        int                          `json:"round,omitempty"`
// 	CC           string                       `json:"cc,omitempty"`
// 	Lobby        *Lobby                       `json:"lobby,omitempty"`
// 	PowerLog     []Powerup                    `json:"powerLog,omitempty"`
// 	Players      map[string]*Player           `json:"players,omitempty"`
// 	Polygon      json.RawMessage              `json:"polygon,omitempty"`
// }

// TODO: remove
// else it will be broadcast to the entire Room. Conn takes precedence over Room
// type RouteMsg struct {
// 	Conn *websocket.Conn
// 	Room string
// 	Data ClientResp
// }
