package websocket

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

type Client struct {
	ID             string
	Room           string
	Name           string
	Hub            *Hub
	Conn           *websocket.Conn
	Send           chan interface{}
	MessageHandler func(c *Client, message []byte)
}

// goroutine to read messages from client
func (c *Client) Read() {
	defer func() {
		slog.Info("Defer read")
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading message: ", err)
			return
		}
		slog.Info("Received message", "message", string(message))
		if c.MessageHandler != nil {
			slog.Info("Calling message handler")
			c.MessageHandler(c, message)
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
	}
}

func (c *Client) Write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		slog.Info("Defer write")
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				slog.Info("Hub closed channel")
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				fmt.Println("error writing unicast", err)
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			slog.Info("Sending ping")
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Error("Error sending ping", "error", err)
				return
			}
		}
	}
}
