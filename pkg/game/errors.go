package game

import (
	"encoding/json"

	"github.com/geobattles/geobattles-backend/pkg/models"
	"github.com/geobattles/geobattles-backend/pkg/websocket"
)

var requestErrorMessages = map[string]string{
	"LOBBY_BUSY":                         "A player is joining. Retry after the connection finishes.",
	"START_CANCELLED":                    "Round start was cancelled by a membership change. Retry when the lobby is ready.",
	"PLAYERS_NOT_READY":                  "All lobby members must be connected and ready before starting a new game.",
	"INVALID_READY":                      "Send setReady with an explicit boolean ready value.",
	"INVALID_REQUEST":                    "Send a valid JSON command object.",
	"INVALID_COMMAND":                    "This command is not supported.",
	"INVALID_LOCATION":                   "Select a location with latitude -90 to 90 and longitude -180 to 180.",
	"INVALID_COUNTRY_CODE":               "Choose countries from the available country list.",
	"INVALID_LOBBY_SETTINGS":             "Check lobby settings and supported limits before retrying.",
	"LOBBY_CAPACITY_TOO_SMALL":           "Capacity cannot be smaller than the number of lobby members, including reconnecting players.",
	"INVALID_GAME_STATE":                 "Reconnect to the lobby before retrying.",
	"PLAYER_NOT_IN_LOBBY":                "Reconnect to the lobby before sending this command.",
	"NOT_ADMIN":                          "Only the lobby administrator can perform this action.",
	"GAME_IN_PROGRESS":                   "Wait for the current game to finish before changing settings.",
	"ALREADY_ACTIVE":                     "A round is already active. Resume the current round.",
	"ROUND_STARTING":                     "Location selection is already in progress. Wait for its result.",
	"GAME_NOT_ACTIVE":                    "Wait for the administrator to start a round.",
	"TIMES_UP":                           "The round has ended. Wait for the next round.",
	"NO_MORE_ATTEMPTS":                   "You have no guesses left in this round.",
	"ALREADY_FINISHED":                   "You have already finished this round.",
	"NO_COUNTRY":                         "Select a location inside a country.",
	"CANT_USE_LAST_ROUND":                "Powerups cannot be used in the last round.",
	"NOT_AVAILABLE":                      "This powerup is no longer available.",
	"WRONG_TARGET":                       "Choose another connected player as the powerup target.",
	"WRONG_TYPE":                         "Choose a supported powerup.",
	"LOCATION_NOT_FOUND":                 "No Street View location was found within the search limit. Retry or change countries.",
	"LOCATION_GENERATION_TIMEOUT":        "Location selection timed out. Retry shortly.",
	"LOCATION_PROVIDER_UNAVAILABLE":      "Street View is temporarily unavailable. Retry shortly.",
	"LOCATION_PROVIDER_DENIED":           "Street View access was denied. Ask the server operator to check API access and billing.",
	"LOCATION_PROVIDER_QUOTA":            "Street View quota is exhausted. Ask the server operator to check quota and billing before retrying.",
	"LOCATION_PROVIDER_UNCONFIGURED":     "Street View is not configured on the server. Contact the server operator.",
	"LOCATION_PROVIDER_INVALID_RESPONSE": "Street View returned an invalid response. Retry later or contact the server operator.",
	"LOCATION_DATA_INVALID":              "Location selection data is unavailable. Change countries or contact the server operator.",
	"REQUEST_FAILED":                     "The request failed. Reconnect and retry, or contact the server operator.",
}

// Only curated codes/text cross the socket boundary; no provider/DB errors,
// URLs, tokens or arbitrary parser details are exposed in this envelope.
func requestError(command, code string) models.ResponseBase {
	message, ok := requestErrorMessages[code]
	if !ok {
		code = "REQUEST_FAILED"
		message = requestErrorMessages[code]
	}
	return models.ResponseBase{
		Status: "ERR", Type: "REQUEST_ERROR",
		Payload: models.RequestErrorPayload{Command: command, Code: code, Message: message},
	}
}

func sendRequestError(c *websocket.Client, command, code string) {
	encoded, err := json.Marshal(requestError(command, code))
	if err != nil {
		c.Close()
		return
	}
	c.Enqueue(encoded)
}
