package models

import "encoding/json"

type ResponseBase struct {
	Status  string      `json:"status"`
	Type    string      `json:"type"`
	Seq     uint64      `json:"seq,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

type RequestErrorPayload struct {
	Command   string `json:"command"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"requestId,omitempty"`
	GameID    string `json:"gameId,omitempty"`
	Round     *int   `json:"round,omitempty"`
}

type ResponsePayload struct {
	GameID          string          `json:"gameId,omitempty"`
	EndedEarly      bool            `json:"endedEarly,omitempty"`
	Lobby           interface{}     `json:"lobby,omitempty"`
	Loc             interface{}     `json:"location,omitempty"`
	User            string          `json:"user,omitempty"`
	AllRes          interface{}     `json:"results,omitempty"`
	EndResults      interface{}     `json:"endResults,omitempty"`
	RevealedAnswers interface{}     `json:"revealedAnswers,omitempty"`
	FullRoundRes    interface{}     `json:"fullroundRes,omitempty"`
	RoundRes        interface{}     `json:"roundRes,omitempty"`
	TotalResults    interface{}     `json:"totalResults,omitempty"`
	GuessRes        interface{}     `json:"playerRes,omitempty"`
	Round           int             `json:"round,omitempty"`
	Phase           string          `json:"phase,omitempty"`
	CC              string          `json:"cc,omitempty"`
	PowerLog        interface{}     `json:"powerLog,omitempty"`
	PowerupState    interface{}     `json:"powerupState,omitempty"`
	Players         interface{}     `json:"players,omitempty"`
	Polygon         json.RawMessage `json:"polygon,omitempty"`
	TimeRemaining   int64           `json:"timeRemaining"`
}
