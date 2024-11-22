package game

import (
	"encoding/json"
	"log/slog"
	"time"

	ws "github.com/gorilla/websocket"
	"github.com/slinarji/go-geo-server/pkg/logic"
	"github.com/slinarji/go-geo-server/pkg/models"
	"github.com/slinarji/go-geo-server/pkg/reverse"
	"github.com/slinarji/go-geo-server/pkg/websocket"
)

// adds player as map[id]name to playerlist in lobby
func AddPlayerToLobby(clientID string, clientName string, lobbyID string, conn *ws.Conn) {
	lobby := LobbyMap[lobbyID]
	hub := LobbyMap[lobbyID].Hub

	client := &websocket.Client{
		ID:             clientID,
		Room:           lobbyID,
		Name:           clientName,
		Hub:            hub,
		Conn:           conn,
		Send:           make(chan interface{}, 1),
		MessageHandler: PlayerMessageHandler,
	}

	hub.Register <- client

	slog.Info("Added player to lobby", "lobby map", LobbyMap)

	go client.Read()
	go client.Write()

	// reconnect player
	if player, exists := lobby.PlayerMap[clientID]; exists {
		player.Connected = true
		player.Name = clientName
	} else {
		// if there is no lobby admin make this user one
		if lobby.Admin == "" {
			lobby.Admin = clientID
		}

		lobby.PlayerMap[clientID] = &Player{
			Name:      clientName,
			Connected: true,
			Color:     genPlayerColor(lobbyID),
			Powerups:  make([]bool, len(*lobby.Conf.Powerups))}

	}

	lobby.NumPlayers = len(lobby.PlayerMap)

	hub.Broadcast <- models.ResponseBase{Status: "OK", Type: "JOINED_LOBBY", Payload: models.ResponsePayload{User: client.ID, Lobby: lobby}}
}

// removes player map from playerlist in lobby
func RemovePlayerFromLobby(clientID string, lobbyID string) {
	lobby := LobbyMap[lobbyID]
	if player, ok := lobby.PlayerMap[clientID]; ok {
		player.Connected = false
	}

	// delete(LobbyMap[lobbyID].PlayerMap, clientID)
	// LobbyMap[lobbyID].NumPlayers = len(LobbyMap[lobbyID].PlayerMap)
	// delete from end results
	// for _, results := range LobbyMap[lobbyID].EndResults {
	// 	delete(results, clientID)
	// }

	// if removed player was admin & there are other players left
	// select one of them as new admin, otherwise make admin empty
	// if there are no players left delete lobby
	if lobby.Admin == clientID {
		for id := range lobby.PlayerMap {
			lobby.Admin = id
			break
		}

	} else if lobby.NumPlayers == 0 {
		slog.Info("Deleting lobby", "lobbyID", lobbyID)
		if lobby.Timer != nil {
			lobby.Timer.Stop()
		}
		delete(LobbyMap, lobbyID)
	}
}

func PlayerMessageHandler(c *websocket.Client, message []byte) {
	var clientReq ClientReq
	err := json.Unmarshal(message, &clientReq)
	if err != nil {
		slog.Error("Error unmarshalling client request: ", "error", err)
		c.Send <- models.ResponseBase{Status: "ERR", Type: "INVALID_REQUEST"}
		return
	}

	slog.Info("Received message", "message", clientReq)

	switch clientReq.Cmd {
	case "update_lobby_settings":
		slog.Info("update lobby settings", "conf", clientReq.Conf)
		lobby, err := UpdateLobby(c.ID, c.Room, clientReq.Conf)
		if err != nil {
			c.Send <- models.ResponseBase{Status: "ERR", Type: err.Error()}
		} else {
			c.Hub.Broadcast <- models.ResponseBase{Status: "OK", Type: "UPDATED_LOBBY", Payload: models.ResponsePayload{Lobby: lobby}}
		}

	case "start":
		// if user is lobby admin send coordinates, otherwise return error
		if c.ID != LobbyMap[c.Room].Admin {
			c.Send <- models.ResponseBase{Status: "ERR", Type: "NOT_ADMIN"}
			break
		}
		if LobbyMap[c.Room].Active {
			c.Send <- models.ResponseBase{Status: "ERR", Type: "ALREADY_ACTIVE"}
			break
		}

		location, ccode := logic.RndLocation(LobbyMap[c.Room].Conf.CCList, LobbyMap[c.Room].CCSize)
		UpdateCurrentLocation(c.Room, location, ccode)
		slog.Info("Start game timer")
		message := models.ResponsePayload{Loc: &location, Players: LobbyMap[c.Room].PlayerMap, PowerLog: LobbyMap[c.Room].PowerLogs[LobbyMap[c.Room].CurrentRound]}
		c.Hub.Broadcast <- models.ResponseBase{Status: "OK", Type: "START_ROUND", Payload: message}

		// 3 sec added to timer for frontend countdown
		LobbyMap[c.Room].Timer = time.AfterFunc(time.Second*time.Duration(LobbyMap[c.Room].Conf.RoundTime+3), func() {
			slog.Info("Times up")
			LobbyMap[c.Room].Active = false

			c.Hub.Broadcast <- models.ResponseBase{Status: "WRN", Type: "TIMES_UP"}
			ProcessBonus(c.Room)
			ProcessPowerups(c.Room)
			ProcessTotal(c.Room)

			var message models.ResponsePayload
			if LobbyMap[c.Room].Conf.Mode == 2 {
				message = models.ResponsePayload{FullRoundRes: LobbyMap[c.Room].RawResults[LobbyMap[c.Room].CurrentRound], Round: LobbyMap[c.Room].CurrentRound, PowerLog: LobbyMap[c.Room].PowerLogs[LobbyMap[c.Room].CurrentRound], Polygon: logic.PolyDB[LobbyMap[c.Room].CurrentCC], RoundRes: LobbyMap[c.Room].EndResults[LobbyMap[c.Room].CurrentRound], TotalResults: LobbyMap[c.Room].TotalResults}
			} else {
				message = models.ResponsePayload{RoundRes: LobbyMap[c.Room].EndResults[LobbyMap[c.Room].CurrentRound], Round: LobbyMap[c.Room].CurrentRound, PowerLog: LobbyMap[c.Room].PowerLogs[LobbyMap[c.Room].CurrentRound], TotalResults: LobbyMap[c.Room].TotalResults}
			}
			c.Hub.Broadcast <- models.ResponseBase{Status: "OK", Type: "ROUND_RESULT", Payload: message}
			// send end of game msg and cleanup lobby
			if LobbyMap[c.Room].CurrentRound >= LobbyMap[c.Room].Conf.NumRounds {
				message := models.ResponsePayload{AllRes: LobbyMap[c.Room].RawResults, TotalResults: LobbyMap[c.Room].TotalResults}
				c.Hub.Broadcast <- models.ResponseBase{Status: "OK", Type: "GAME_END", Payload: message}
				ResetLobby(c.Room)
			}
		})

	case "use_powerup":
		if LobbyMap[c.Room].CurrentRound == 0 {
			c.Send <- models.ResponseBase{Status: "ERR", Type: "GAME_NOT_ACTIVE"}
			break
		}
		if LobbyMap[c.Room].CurrentRound == LobbyMap[c.Room].Conf.NumRounds {
			c.Send <- models.ResponseBase{Status: "ERR", Type: "CANT_USE_LAST_ROUND"}
			break
		}
		clientReq.Powerup.Source = c.ID
		target, err := UsePowerup(clientReq.Powerup, c.Room)
		if err != nil {
			c.Send <- models.ResponseBase{Status: "ERR", Type: err.Error()}
			break
		}
		c.Send <- models.ResponseBase{Status: "OK", Type: "POWERUP_USED"}
		if target != "" {
			c.Send <- models.ResponseBase{Status: "WRN", Type: "TODO: nofify duel target"}
			// c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Hub.Clients[c.Room][target], Data: logic.ClientResp{Status: "WRN", Type: "DUEL_FROM", User: c.ID}}
		}

	case "submit_location":
		slog.Info("submit location", "location", *clientReq.Loc)
		_, _, err := SubmitResult(c.Room, c.ID, *clientReq.Loc)
		//err := lobby.AddToResults(c.Room, c.ID, clientReq.Location, distance)

		if err != nil && err.Error() != "ROUND_FINISHED" {
			c.Send <- models.ResponseBase{Status: "ERR", Type: err.Error()}
			break
		}
		c.Hub.Broadcast <- models.ResponseBase{Status: "OK", Type: "NEW_RESULT", Payload: models.ResponsePayload{User: c.ID, GuessRes: &LobbyMap[c.Room].RawResults[LobbyMap[c.Room].CurrentRound][c.ID][len(LobbyMap[c.Room].RawResults[LobbyMap[c.Room].CurrentRound][c.ID])-1]}}

		// if round is finished notify lobby
		// what does this do??
		if err != nil && err.Error() == "ROUND_FINISHED" {
			LobbyMap[c.Room].Active = false
			LobbyMap[c.Room].Timer.Stop()
			slog.Info("Stopped timer")
			c.Hub.Broadcast <- models.ResponseBase{Status: "WRN", Type: err.Error()}
			ProcessBonus(c.Room)
			ProcessPowerups(c.Room)
			ProcessTotal(c.Room)

			var message models.ResponsePayload
			if LobbyMap[c.Room].Conf.Mode == 2 {
				message = models.ResponsePayload{FullRoundRes: LobbyMap[c.Room].RawResults[LobbyMap[c.Room].CurrentRound], Round: LobbyMap[c.Room].CurrentRound, PowerLog: LobbyMap[c.Room].PowerLogs[LobbyMap[c.Room].CurrentRound], Polygon: logic.PolyDB[LobbyMap[c.Room].CurrentCC], RoundRes: LobbyMap[c.Room].EndResults[LobbyMap[c.Room].CurrentRound], TotalResults: LobbyMap[c.Room].TotalResults}
			} else {
				message = models.ResponsePayload{RoundRes: LobbyMap[c.Room].EndResults[LobbyMap[c.Room].CurrentRound], Round: LobbyMap[c.Room].CurrentRound, PowerLog: LobbyMap[c.Room].PowerLogs[LobbyMap[c.Room].CurrentRound], TotalResults: LobbyMap[c.Room].TotalResults}
			}
			c.Hub.Broadcast <- models.ResponseBase{Status: "OK", Type: "ROUND_RESULT", Payload: message}
			// send end of game msg and cleanup lobby
			if LobbyMap[c.Room].CurrentRound >= LobbyMap[c.Room].Conf.NumRounds {
				message := models.ResponsePayload{AllRes: LobbyMap[c.Room].RawResults, TotalResults: LobbyMap[c.Room].TotalResults}
				c.Hub.Broadcast <- models.ResponseBase{Status: "OK", Type: "GAME_END", Payload: message}
				ResetLobby(c.Room)
			}
		}
	case "loc_to_cc":
		cc, err := reverse.ReverseGeocode(clientReq.Loc.Lng, clientReq.Loc.Lat)
		if err != nil {
			c.Send <- models.ResponseBase{Status: "ERR", Type: err.Error()}
			break
		}
		c.Send <- models.ResponseBase{Status: "OK", Type: "CC", Payload: models.ResponsePayload{CC: cc, Polygon: logic.PolyDB[cc]}}

	default:
		slog.Info("echo message", "message", clientReq)
		c.Send <- clientReq
	}
}
