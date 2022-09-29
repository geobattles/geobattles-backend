package websocket

import (
	"example/web-service-gin/pkg/lobby"
	"example/web-service-gin/pkg/logic"
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
	Pool *Pool
	Room string
	Name string
}

func (c *Client) Read() {
	defer func() {
		//fmt.Println("defer read unregister")
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()

	for {
		var clientReq logic.ClientReq
		err := c.Conn.ReadJSON(&clientReq)

		if err != nil {
			fmt.Println("error reading client json: ", err)
			// if connection was closed unregister client, on other error (egwrong json fields) just break current loop
			if err.Error() == "websocket: close 1001 (going away)" {
				fmt.Println("ws closed")
				return
			}

			c.Pool.Transmit <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: err.Error()}}
			break
		}
		fmt.Println("Client msg: ", clientReq)

		switch clientReq.Cmd {
		case "update_lobby_settings":
			fmt.Println(clientReq.Conf)
			lobby, err := lobby.UpdateLobby(c.ID, c.Room, clientReq.Conf)
			if err != nil {
				c.Pool.Transmit <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: err.Error()}}
			} else {
				c.Pool.Transmit <- logic.RouteMsg{Room: c.Room, Data: logic.ClientResp{Status: "OK", Type: "UPDATED_LOBBY", Lobby: lobby}}
			}
		case "start":
			// if user is lobby admin send coordinates, otherwise return error
			if c.ID != lobby.LobbyMap[c.Room].Admin {
				c.Pool.Transmit <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: "NOT_ADMIN"}}
				break
			}
			if lobby.LobbyMap[c.Room].Active {
				c.Pool.Transmit <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: "ALREADY_ACTIVE"}}
				break
			}

			fmt.Println("USER IS ADMIN")
			location, ccode := logic.RndLocation(lobby.LobbyMap[c.Room].Conf.CCList, lobby.LobbyMap[c.Room].CCSize)
			lobby.UpdateCurrentLocation(c.Room, location, ccode)
			fmt.Println("start timer")
			message := logic.ClientResp{Status: "OK", Type: "START_ROUND", Loc: &location, Players: lobby.LobbyMap[c.Room].PlayerMap, PowerLog: lobby.LobbyMap[c.Room].PowerLogs[lobby.LobbyMap[c.Room].CurrentRound]}
			c.Pool.Transmit <- logic.RouteMsg{Room: c.Room, Data: message}

			// 3 sec added to timer for countdown
			c.Pool.Timer = time.AfterFunc(time.Second*time.Duration(lobby.LobbyMap[c.Room].Conf.RoundTime+3), func() {
				fmt.Println("times up")
				if lobby.LobbyMap[c.Room] == nil {
					fmt.Println("LOBBY ne obstaja vec")
					return
				}
				lobby.LobbyMap[c.Room].Active = false

				c.Pool.Transmit <- logic.RouteMsg{Room: c.Room, Data: logic.ClientResp{Status: "WRN", Type: "TIMES_UP"}}
				lobby.ProcessBonus(c.Room)
				lobby.ProcessPowerups(c.Room)
				message := logic.ClientResp{Status: "OK", Type: "ROUND_RESULT", RoundRes: lobby.LobbyMap[c.Room].EndResults[lobby.LobbyMap[c.Room].CurrentRound], Round: lobby.LobbyMap[c.Room].CurrentRound, PowerLog: lobby.LobbyMap[c.Room].PowerLogs[lobby.LobbyMap[c.Room].CurrentRound]}
				c.Pool.Transmit <- logic.RouteMsg{Room: c.Room, Data: message}
				// send end of game msg and cleanup lobby
				if lobby.LobbyMap[c.Room].CurrentRound >= lobby.LobbyMap[c.Room].Conf.NumRounds {
					message := logic.ClientResp{Status: "OK", Type: "GAME_END", AllRes: lobby.LobbyMap[c.Room].RawResults}
					c.Pool.Transmit <- logic.RouteMsg{Room: c.Room, Data: message}
					lobby.ResetLobby(c.Room)
				}
			})

		case "use_powerup":
			if lobby.LobbyMap[c.Room].CurrentRound == 0 {
				c.Pool.Transmit <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: "GAME_NOT_ACTIVE"}}
				break
			}
			if lobby.LobbyMap[c.Room].CurrentRound == lobby.LobbyMap[c.Room].Conf.NumRounds {
				c.Pool.Transmit <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: "CANT_USE_LAST_ROUND"}}
				break
			}
			clientReq.Powerup.Source = c.ID
			target, err := lobby.UsePowerup(clientReq.Powerup, c.Room)
			if err != nil {
				c.Pool.Transmit <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: err.Error()}}
				break
			}
			c.Pool.Transmit <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "OK", Type: "POWERUP_USED"}}
			if target != "" {
				c.Pool.Transmit <- logic.RouteMsg{Conn: c.Pool.Rooms[c.Room][target], Data: logic.ClientResp{Status: "WRN", Type: "DUEL_FROM", User: c.ID}}
			}

		case "submit_location":
			fmt.Println(*clientReq.Loc)
			_, _, err := lobby.SubmitResult(c.Room, c.ID, *clientReq.Loc)
			//err := lobby.AddToResults(c.Room, c.ID, clientReq.Location, distance)

			if err != nil && err.Error() != "ROUND_FINISHED" {
				c.Pool.Transmit <- logic.RouteMsg{Conn: c.Conn, Data: logic.ClientResp{Status: "ERR", Type: err.Error()}}
				break
			}
			//c.Pool.Transmit <- logic.Message{Room: c.Room, Data: logic.ResponseMsg{Status: "OK", Type: "NEW_RESULT", User: c.ID, Distance: dist, Score: score, Location: clientReq.Location}}
			c.Pool.Transmit <- logic.RouteMsg{Room: c.Room, Data: logic.ClientResp{Status: "OK", Type: "NEW_RESULT", User: c.ID, GuessRes: &lobby.LobbyMap[c.Room].RawResults[lobby.LobbyMap[c.Room].CurrentRound][c.ID][len(lobby.LobbyMap[c.Room].RawResults[lobby.LobbyMap[c.Room].CurrentRound][c.ID])-1]}}

			// message := logic.ResponseMsg{Status: "OK", Type: "ALL_RESULTS", Results: lobby.LobbyMap[c.Room].Results}
			// c.Pool.Transmit <- logic.Message{Room: c.Room, Data: message}
			// if round is finished notify lobby
			if err != nil && err.Error() == "ROUND_FINISHED" {
				lobby.LobbyMap[c.Room].Active = false
				fmt.Println("STOP TIMER")
				c.Pool.Timer.Stop()
				c.Pool.Transmit <- logic.RouteMsg{Room: c.Room, Data: logic.ClientResp{Status: "WRN", Type: err.Error()}}
				lobby.ProcessBonus(c.Room)
				lobby.ProcessPowerups(c.Room)
				message := logic.ClientResp{Status: "OK", Type: "ROUND_RESULT", RoundRes: lobby.LobbyMap[c.Room].EndResults[lobby.LobbyMap[c.Room].CurrentRound], Round: lobby.LobbyMap[c.Room].CurrentRound, PowerLog: lobby.LobbyMap[c.Room].PowerLogs[lobby.LobbyMap[c.Room].CurrentRound]}
				c.Pool.Transmit <- logic.RouteMsg{Room: c.Room, Data: message}
				// send end of game msg and cleanup lobby
				if lobby.LobbyMap[c.Room].CurrentRound >= lobby.LobbyMap[c.Room].Conf.NumRounds {
					message := logic.ClientResp{Status: "OK", Type: "GAME_END", AllRes: lobby.LobbyMap[c.Room].RawResults}
					c.Pool.Transmit <- logic.RouteMsg{Room: c.Room, Data: message}
					lobby.ResetLobby(c.Room)
				}
			}
		}
	}
}
