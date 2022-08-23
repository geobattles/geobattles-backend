package websocket

import (
	"example/web-service-gin/pkg/lobby"
	"example/web-service-gin/pkg/logic"
	"fmt"

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
			return
		}
		fmt.Println("Client msg: ", clientReq)

		switch clientReq.Command {
		case "start":
			// if user is lobby admin send coordinates, otherwise return error
			if c.ID == lobby.LobbyMap[c.Room].Admin {
				fmt.Println("USER IS ADMIN")
				var location logic.Coordinates = logic.GenerateRndLocation()
				//lobby.MarkGameActive(c.Room)
				lobby.UpdateCurrentLocation(c.Room, location)
				message := logic.ResponseMsg{Status: "OK", Location: location}
				c.Pool.Transmit <- logic.Message{Room: c.Room, Data: message}
			} else {
				c.Pool.Transmit <- logic.Message{Conn: c.Conn, Data: logic.ResponseMsg{Status: "NOT_ADMIN"}}
			}

		case "submit_location":
			var distance = lobby.CalculateDistance(c.Room, clientReq.Location)
			lobby.AddToResults(c.Room, c.ID, distance)

			c.Pool.Transmit <- logic.Message{Conn: c.Conn, Data: logic.ResponseMsg{Status: "OK", Distance: distance}}
			// TODO: only send results of current round
			message := logic.ResponseMsg{Status: "OK", Results: lobby.LobbyMap[c.Room].Results}

			c.Pool.Transmit <- logic.Message{Room: c.Room, Data: message}
		}
		// if clientReq.Command == "start" {
		// 	if c.ID == lobby.LobbyMap[c.Room].Admin {
		// 		fmt.Println("USER IS ADMIN")
		// 	}
		// 	var location logic.Coordinates = logic.GenerateRndLocation()
		// 	lobby.MarkGameActive(c.Room)
		// 	lobby.UpdateCurrentLocation(c.Room, location)
		// 	message := logic.ResponseMsg{Status: "OK", Location: location}
		// 	c.Pool.Transmit <- logic.Message{Room: c.Room, Data: message}
		// }
		// if clientReq.Command == "submit_location" {
		// 	var distance = lobby.CalculateDistance(c.Room, clientReq.Location)
		// 	lobby.AddToResults(c.Room, c.ID, distance)

		// 	message := logic.ResponseMsg{Status: "OK", Results: lobby.LobbyMap[c.Room].Results}
		// 	c.Pool.Transmit <- logic.Message{Room: c.Room, Data: message}
		// }
	}
}
