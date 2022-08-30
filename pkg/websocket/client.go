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
			c.Pool.Transmit <- logic.Message{Conn: c.Conn, Data: logic.ResponseMsg{Status: "ERR", Type: err.Error()}}
			return
		}
		fmt.Println("Client msg: ", clientReq)

		switch clientReq.Command {
		case "update_lobby_settings":
			fmt.Println(clientReq.Conf)
			lobby, err := lobby.UpdateLobby(c.ID, c.Room, clientReq.Conf)
			if err != nil {
				c.Pool.Transmit <- logic.Message{Conn: c.Conn, Data: logic.ResponseMsg{Status: "ERR", Type: err.Error()}}
			} else {
				c.Pool.Transmit <- logic.Message{Room: c.Room, Data: logic.ResponseMsg{Status: "OK", Type: "UPDATED_LOBBY", Lobby: lobby}}
			}
		case "start":
			// if user is lobby admin send coordinates, otherwise return error
			if c.ID != lobby.LobbyMap[c.Room].Admin {
				c.Pool.Transmit <- logic.Message{Conn: c.Conn, Data: logic.ResponseMsg{Status: "ERR", Type: "NOT_ADMIN"}}
				break
			}
			if lobby.LobbyMap[c.Room].Timer == true {
				c.Pool.Transmit <- logic.Message{Conn: c.Conn, Data: logic.ResponseMsg{Status: "ERR", Type: "ALREADY_ACTIVE"}}
				break
			}

			fmt.Println("USER IS ADMIN")
			var location logic.Coordinates = logic.RndLocation()
			lobby.UpdateCurrentLocation(c.Room, location)
			fmt.Println("start timer")
			c.Pool.Timer = time.AfterFunc(time.Second*time.Duration(lobby.LobbyMap[c.Room].Conf.RoundTime), func() {
				fmt.Println("times up")
				lobby.LobbyMap[c.Room].Timer = false

				c.Pool.Transmit <- logic.Message{Room: c.Room, Data: logic.ResponseMsg{Status: "WRN", Type: "TIMES_UP"}}
				message := logic.ResponseMsg{Status: "OK", Type: "ROUND_RESULT", RoundRes: lobby.LobbyMap[c.Room].Results[lobby.LobbyMap[c.Room].CurrentRound]}
				c.Pool.Transmit <- logic.Message{Room: c.Room, Data: message}
			})
			message := logic.ResponseMsg{Status: "OK", Type: "START_ROUND", Location: &location}
			c.Pool.Transmit <- logic.Message{Room: c.Room, Data: message}

		case "submit_location":
			fmt.Println(*clientReq.Location)
			dist, score, err := lobby.SubmitResult(c.Room, c.ID, *clientReq.Location)
			//err := lobby.AddToResults(c.Room, c.ID, clientReq.Location, distance)

			if err != nil && err.Error() != "ROUND_FINISHED" {
				c.Pool.Transmit <- logic.Message{Conn: c.Conn, Data: logic.ResponseMsg{Status: "ERR", Type: err.Error()}}
				break
			}
			c.Pool.Transmit <- logic.Message{Room: c.Room, Data: logic.ResponseMsg{Status: "OK", Type: "NEW_RESULT", User: c.ID, Distance: dist, Score: score, Location: clientReq.Location}}

			// message := logic.ResponseMsg{Status: "OK", Type: "ALL_RESULTS", Results: lobby.LobbyMap[c.Room].Results}
			// c.Pool.Transmit <- logic.Message{Room: c.Room, Data: message}
			// if round is finished notify lobby
			if err != nil && err.Error() == "ROUND_FINISHED" {
				lobby.LobbyMap[c.Room].Timer = false
				fmt.Println("STOP TIMER")
				c.Pool.Timer.Stop()
				c.Pool.Transmit <- logic.Message{Room: c.Room, Data: logic.ResponseMsg{Status: "WRN", Type: err.Error()}}
				message := logic.ResponseMsg{Status: "OK", Type: "ROUND_RESULT", RoundRes: lobby.LobbyMap[c.Room].Results[lobby.LobbyMap[c.Room].CurrentRound]}
				c.Pool.Transmit <- logic.Message{Room: c.Room, Data: message}
			}
		}
	}
}
