package logic

import (
	"github.com/gorilla/websocket"
)

type Coordinates struct {
	Latitude  float64 `json:"lat"`
	Longitude float64 `json:"lng"`
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
	Command  string      `json:"command"`
	Location Coordinates `json:"location"`
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
	Status   string      `json:"status"`
	Location Coordinates `json:"location,omitempty"`
	//Room     string                       `json:"-"`
	Distance float64                      `json:"distance,omitempty"`
	Results  map[int]map[string][]Results `json:"results,omitempty"`
}
