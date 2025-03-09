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
		ID: clientID,
		// Room:           lobbyID,
		Name:           clientName,
		Hub:            hub,
		Conn:           conn,
		Send:           make(chan interface{}, 1),
		MessageHandler: PlayerMessageHandler,
		Lobby:          lobby,
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
			Color:     lobby.getPlayerColor(),
			Powerups:  make([]bool, len(*lobby.Conf.Powerups))}

	}

	lobby.NumPlayers = len(lobby.PlayerMap)

	hub.Broadcast <- models.ResponseBase{
		Status: "OK", Type: "JOINED_LOBBY",
		Payload: models.ResponsePayload{
			User:  client.ID,
			Lobby: lobby,
		},
	}

	// send data to resume mid round
	if lobby.Active && lobby.RountTimer.Timer != nil {
		message := models.ResponsePayload{
			Loc:           &lobby.CurrentLoc[lobby.CurrentRound-1],
			Players:       lobby.PlayerMap,
			PowerLog:      lobby.PowerLogs[lobby.CurrentRound],
			FullRoundRes:  lobby.RawResults[lobby.CurrentRound],
			TimeRemaining: time.Duration(time.Until(lobby.RountTimer.End).Milliseconds()),
		}
		client.Send <- models.ResponseBase{
			Status:  "OK",
			Type:    "REJOIN_ROUND",
			Payload: message,
		}
	}
}

func PlayerMessageHandler(c *websocket.Client, message []byte) {
	lobby, ok := c.Lobby.(*Lobby)
	if !ok {
		slog.Error("Failed to assert lobby from client struct")
		return
	}

	var clientReq ClientReq
	err := json.Unmarshal(message, &clientReq)
	if err != nil {
		slog.Error("Error unmarshalling client request: ", "error", err)
		c.Send <- models.ResponseBase{
			Status: "ERR",
			Type:   "INVALID_REQUEST",
		}
		return
	}

	slog.Debug("Parsing client msg", "msg", clientReq)

	switch clientReq.Cmd {
	case "disconnect":
		lobby.removePlayer(c.ID)
		c.Hub.Broadcast <- models.ResponseBase{
			Status: "OK",
			Type:   "LEFT_LOBBY",
			Payload: models.ResponsePayload{
				User:  c.ID,
				Lobby: lobby,
			},
		}

		// end round if remaining players have submitted all guesses
		if lobby.Active && lobby.checkAllFinished() {
			lobby.Active = false
			lobby.RountTimer.Timer.Stop()

			lobby.processRoundEnd()
		}

	case "update_lobby_settings":
		slog.Info("update lobby settings", "conf", clientReq.Conf)

		err := lobby.updateConf(c.ID, clientReq.Conf)
		if err != nil {
			c.Send <- models.ResponseBase{
				Status: "ERR",
				Type:   err.Error(),
			}
		} else {
			c.Hub.Broadcast <- models.ResponseBase{
				Status:  "OK",
				Type:    "UPDATED_LOBBY",
				Payload: models.ResponsePayload{Lobby: lobby}}
		}

	case "start":
		slog.Info("start round")
		msg, err := lobby.startGame(c.ID)

		if err != nil {
			c.Send <- models.ResponseBase{
				Status: "ERR",
				Type:   err.Error(),
			}
		} else {
			c.Hub.Broadcast <- models.ResponseBase{
				Status:  "OK",
				Type:    "START_ROUND",
				Payload: msg,
			}
		}

	case "use_powerup":
		slog.Debug("use powerup", "clientID", c.ID, "powerup", clientReq.Powerup)
		err := lobby.usePowerup(c.ID, clientReq.Powerup)

		if err != nil {
			c.Send <- models.ResponseBase{
				Status: "ERR",
				Type:   err.Error(),
			}
		} else {
			c.Send <- models.ResponseBase{
				Status: "OK",
				Type:   "POWERUP_USED",
			}
		}

	case "submit_location":
		slog.Debug("submit location", "location", *clientReq.Loc)
		_, _, err := lobby.submitResult(c.ID, *clientReq.Loc)

		if err != nil && err.Error() != "ROUND_FINISHED" {
			c.Send <- models.ResponseBase{
				Status: "ERR",
				Type:   err.Error(),
			}
			break
		}
		msg := models.ResponsePayload{
			User:     c.ID,
			GuessRes: &lobby.RawResults[lobby.CurrentRound][c.ID][len(lobby.RawResults[lobby.CurrentRound][c.ID])-1],
		}
		c.Hub.Broadcast <- models.ResponseBase{
			Status:  "OK",
			Type:    "NEW_RESULT",
			Payload: msg,
		}

		// if round is finished notify lobby
		// what does this do??
		if err != nil && err.Error() == "ROUND_FINISHED" {
			lobby.Active = false
			lobby.RountTimer.Timer.Stop()
			slog.Debug("Stopped timer")

			lobby.processRoundEnd()
		}

	case "loc_to_cc":
		cc, err := reverse.ReverseGeocode(clientReq.Loc.Lng, clientReq.Loc.Lat)
		if err != nil {
			c.Send <- models.ResponseBase{
				Status: "ERR",
				Type:   err.Error(),
			}
			break
		}
		c.Send <- models.ResponseBase{
			Status: "OK",
			Type:   "CC",
			Payload: models.ResponsePayload{
				CC:      cc,
				Polygon: logic.PolyDB[cc],
			},
		}

	case "ping":
		c.Send <- models.ResponseBase{
			Status: "OK",
			Type:   "PONG",
		}

	case "pong":
		c.Conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		slog.Debug("Received pong message")

	default:
		slog.Debug("echo message", "message", clientReq)
		c.Send <- clientReq
	}
}
