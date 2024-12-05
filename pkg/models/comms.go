package models

import (
	"encoding/json"
)

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
