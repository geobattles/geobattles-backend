package game

import (
	"encoding/json"
	"fmt"
	"log/slog"

	ws "github.com/gorilla/websocket"
	"github.com/slinarji/go-geo-server/pkg/models"
	"github.com/slinarji/go-geo-server/pkg/websocket"
)

// adds player as map[id]name to playerlist in lobby
func AddPlayerToLobby(clientID string, clientName string, lobbyID string, conn *ws.Conn) {
	lobby := LobbyMap[lobbyID]
	hub := LobbyMap[lobbyID].Hub

	client := &websocket.Client{
		ID:   clientID,
		Room: lobbyID,
		Name: clientName,
		Hub:  hub,
		Conn: conn,
		Send: make(chan interface{}, 1),
	}

	hub.Register <- client

	slog.Info("Added player to lobby", "lobby map", LobbyMap)

	go client.Read()
	go client.Write()

	// if there is no lobby admin make this user one
	if lobby.Admin == "" {
		lobby.Admin = clientID
	}
	// LobbyMap[lobbyID].PlayerMap[clientID] = &logic.Player{Name: clientName, Color: genPlayerColor(lobbyID), Powerups: *LobbyMap[lobbyID].Conf.Powerups}
	lobby.PlayerMap[clientID] = &Player{Name: clientName, Color: genPlayerColor(lobbyID), Powerups: make([]bool, len(*lobby.Conf.Powerups))}
	lobby.NumPlayers = len(lobby.PlayerMap)
}

// removes player map from playerlist in lobby
func RemovePlayerFromLobby(clientID string, lobbyID string) {
	delete(LobbyMap[lobbyID].PlayerMap, clientID)
	LobbyMap[lobbyID].NumPlayers = len(LobbyMap[lobbyID].PlayerMap)
	// delete from end results
	for _, results := range LobbyMap[lobbyID].EndResults {
		delete(results, clientID)
	}
	fmt.Println("after deleting results", LobbyMap[lobbyID].EndResults)
	// if removed player was admin & there are other players left
	// select one of them as new admin, otherwise make admin empty
	// if there are no players left delete lobby
	if LobbyMap[lobbyID].Admin == clientID && LobbyMap[lobbyID].NumPlayers != 0 {
		for id := range LobbyMap[lobbyID].PlayerMap {
			LobbyMap[lobbyID].Admin = id
			break
		}

	} else if LobbyMap[lobbyID].NumPlayers == 0 {
		slog.Info("Deleting lobby", "lobbyID", lobbyID)
		if LobbyMap[lobbyID].Timer != nil {
			LobbyMap[lobbyID].Timer.Stop()
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

	// var clientReq logic.ClientReq
	// err := c.Conn.ReadJSON(&clientReq)

	// if err != nil {
	// 	fmt.Println("error reading client json: ", err)
	// 	// if connection was closed unregister client, on other error (egwrong json fields) just break current loop
	// 	if err.Error() == "websocket: close 1001 (going away)" {
	// 		fmt.Println("ws closed")
	// 		return
	// 	}

	// 	c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: err.Error()}}
	// 	break
	// }
	// fmt.Println("Client msg: ", clientReq)

	// switch clientReq.Cmd {
	// case "update_lobby_settings":
	// 	fmt.Println(clientReq.Conf)
	// 	lobby, err := lobby.UpdateLobby(c.ID, c.Room, clientReq.Conf)
	// 	if err != nil {
	// 		c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: err.Error()}}
	// 	} else {
	// 		c.Hub.Broadcast <- logic.RouteMsg{Room: c.Room, Data: logic.ClientResp{Status: "OK", Type: "UPDATED_LOBBY", Lobby: lobby}}
	// 	}
	// case "start":
	// 	// if user is lobby admin send coordinates, otherwise return error
	// 	if c.ID != lobby.LobbyMap[c.Room].Admin {
	// 		c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: "NOT_ADMIN"}}
	// 		break
	// 	}
	// 	if lobby.LobbyMap[c.Room].Active {
	// 		c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: "ALREADY_ACTIVE"}}
	// 		break
	// 	}

	// 	fmt.Println("USER IS ADMIN")
	// 	location, ccode := logic.RndLocation(lobby.LobbyMap[c.Room].Conf.CCList, lobby.LobbyMap[c.Room].CCSize)
	// 	lobby.UpdateCurrentLocation(c.Room, location, ccode)
	// 	fmt.Println("start timer")
	// 	message := logic.ClientResp{Status: "OK", Type: "START_ROUND", Loc: &location, Players: lobby.LobbyMap[c.Room].PlayerMap, PowerLog: lobby.LobbyMap[c.Room].PowerLogs[lobby.LobbyMap[c.Room].CurrentRound]}
	// 	c.Hub.Broadcast <- logic.RouteMsg{Room: c.Room, Data: message}

	// 	// 3 sec added to timer for frontend countdown
	// 	lobby.LobbyMap[c.Room].Timer = time.AfterFunc(time.Second*time.Duration(lobby.LobbyMap[c.Room].Conf.RoundTime+3), func() {
	// 		fmt.Println("times up")
	// 		lobby.LobbyMap[c.Room].Active = false

	// 		c.Hub.Broadcast <- logic.RouteMsg{Room: c.Room, Data: logic.ClientResp{Status: "WRN", Type: "TIMES_UP"}}
	// 		lobby.ProcessBonus(c.Room)
	// 		lobby.ProcessPowerups(c.Room)
	// 		lobby.ProcessTotal(c.Room)

	// 		var message logic.ClientResp
	// 		if lobby.LobbyMap[c.Room].Conf.Mode == 2 {
	// 			message = logic.ClientResp{Status: "OK", Type: "ROUND_RESULT", FullRoundRes: lobby.LobbyMap[c.Room].RawResults[lobby.LobbyMap[c.Room].CurrentRound], Round: lobby.LobbyMap[c.Room].CurrentRound, PowerLog: lobby.LobbyMap[c.Room].PowerLogs[lobby.LobbyMap[c.Room].CurrentRound], Polygon: logic.PolyDB[lobby.LobbyMap[c.Room].CurrentCC], RoundRes: lobby.LobbyMap[c.Room].EndResults[lobby.LobbyMap[c.Room].CurrentRound], TotalResults: lobby.LobbyMap[c.Room].TotalResults}
	// 		} else {
	// 			message = logic.ClientResp{Status: "OK", Type: "ROUND_RESULT", RoundRes: lobby.LobbyMap[c.Room].EndResults[lobby.LobbyMap[c.Room].CurrentRound], Round: lobby.LobbyMap[c.Room].CurrentRound, PowerLog: lobby.LobbyMap[c.Room].PowerLogs[lobby.LobbyMap[c.Room].CurrentRound], TotalResults: lobby.LobbyMap[c.Room].TotalResults}
	// 		}
	// 		c.Hub.Broadcast <- logic.RouteMsg{Room: c.Room, Data: message}
	// 		// send end of game msg and cleanup lobby
	// 		if lobby.LobbyMap[c.Room].CurrentRound >= lobby.LobbyMap[c.Room].Conf.NumRounds {
	// 			message := logic.ClientResp{Status: "OK", Type: "GAME_END", AllRes: lobby.LobbyMap[c.Room].RawResults, TotalResults: lobby.LobbyMap[c.Room].TotalResults}
	// 			c.Hub.Broadcast <- logic.RouteMsg{Room: c.Room, Data: message}
	// 			lobby.ResetLobby(c.Room)
	// 		}
	// 	})

	// case "use_powerup":
	// 	if lobby.LobbyMap[c.Room].CurrentRound == 0 {
	// 		c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: "GAME_NOT_ACTIVE"}}
	// 		break
	// 	}
	// 	if lobby.LobbyMap[c.Room].CurrentRound == lobby.LobbyMap[c.Room].Conf.NumRounds {
	// 		c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: "CANT_USE_LAST_ROUND"}}
	// 		break
	// 	}
	// 	clientReq.Powerup.Source = c.ID
	// 	target, err := lobby.UsePowerup(clientReq.Powerup, c.Room)
	// 	if err != nil {
	// 		c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: err.Error()}}
	// 		break
	// 	}
	// 	c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "OK", Type: "POWERUP_USED"}}
	// 	if target != "" {
	// 		c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Hub.Clients[c.Room][target], Data: logic.ClientResp{Status: "WRN", Type: "DUEL_FROM", User: c.ID}}
	// 	}

	// case "submit_location":
	// 	fmt.Println(*clientReq.Loc)
	// 	_, _, err := lobby.SubmitResult(c.Room, c.ID, *clientReq.Loc)
	// 	//err := lobby.AddToResults(c.Room, c.ID, clientReq.Location, distance)

	// 	if err != nil && err.Error() != "ROUND_FINISHED" {
	// 		c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: err.Error()}}
	// 		break
	// 	}
	// 	c.Hub.Broadcast <- logic.RouteMsg{Room: c.Room, Data: logic.ClientResp{Status: "OK", Type: "NEW_RESULT", User: c.ID, GuessRes: &lobby.LobbyMap[c.Room].RawResults[lobby.LobbyMap[c.Room].CurrentRound][c.ID][len(lobby.LobbyMap[c.Room].RawResults[lobby.LobbyMap[c.Room].CurrentRound][c.ID])-1]}}

	// 	// if round is finished notify lobby
	// 	if err != nil && err.Error() == "ROUND_FINISHED" {
	// 		lobby.LobbyMap[c.Room].Active = false
	// 		fmt.Println("STOP TIMER")
	// 		lobby.LobbyMap[c.Room].Timer.Stop()
	// 		c.Hub.Broadcast <- logic.RouteMsg{Room: c.Room, Data: logic.ClientResp{Status: "WRN", Type: err.Error()}}
	// 		lobby.ProcessBonus(c.Room)
	// 		lobby.ProcessPowerups(c.Room)
	// 		lobby.ProcessTotal(c.Room)

	// 		var message logic.ClientResp
	// 		if lobby.LobbyMap[c.Room].Conf.Mode == 2 {
	// 			message = logic.ClientResp{Status: "OK", Type: "ROUND_RESULT", FullRoundRes: lobby.LobbyMap[c.Room].RawResults[lobby.LobbyMap[c.Room].CurrentRound], Round: lobby.LobbyMap[c.Room].CurrentRound, PowerLog: lobby.LobbyMap[c.Room].PowerLogs[lobby.LobbyMap[c.Room].CurrentRound], Polygon: logic.PolyDB[lobby.LobbyMap[c.Room].CurrentCC], RoundRes: lobby.LobbyMap[c.Room].EndResults[lobby.LobbyMap[c.Room].CurrentRound], TotalResults: lobby.LobbyMap[c.Room].TotalResults}
	// 		} else {
	// 			message = logic.ClientResp{Status: "OK", Type: "ROUND_RESULT", RoundRes: lobby.LobbyMap[c.Room].EndResults[lobby.LobbyMap[c.Room].CurrentRound], Round: lobby.LobbyMap[c.Room].CurrentRound, PowerLog: lobby.LobbyMap[c.Room].PowerLogs[lobby.LobbyMap[c.Room].CurrentRound], TotalResults: lobby.LobbyMap[c.Room].TotalResults}
	// 		}
	// 		c.Hub.Broadcast <- logic.RouteMsg{Room: c.Room, Data: message}
	// 		// send end of game msg and cleanup lobby
	// 		if lobby.LobbyMap[c.Room].CurrentRound >= lobby.LobbyMap[c.Room].Conf.NumRounds {
	// 			message := logic.ClientResp{Status: "OK", Type: "GAME_END", AllRes: lobby.LobbyMap[c.Room].RawResults, TotalResults: lobby.LobbyMap[c.Room].TotalResults}
	// 			c.Hub.Broadcast <- logic.RouteMsg{Room: c.Room, Data: message}
	// 			lobby.ResetLobby(c.Room)
	// 		}
	// 	}
	// case "loc_to_cc":
	// 	cc, err := reverse.ReverseGeocode(clientReq.Loc.Lng, clientReq.Loc.Lat)
	// 	if err != nil {
	// 		c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: err.Error()}}
	// 		break
	// 	}
	// 	c.Hub.Broadcast <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "OK", Type: "CC", CC: cc, Polygon: logic.PolyDB[cc]}}

	// }

	switch clientReq.Cmd {
	case "test":
		// Call lobby functions as needed
	// ...
	default:
		c.Send <- clientReq
	}
}
