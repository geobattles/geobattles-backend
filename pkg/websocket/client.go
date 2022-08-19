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

// type Message struct {
// 	Type int    `json:"type"`
// 	Body string `json:"body"`
// 	Room string `json:"room"`
// }

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

		if clientReq.Command == "start" {
			var location logic.Coordinates = logic.GenerateRndLocation()
			lobby.MarkGameActive(c.Room)
			lobby.UpdateCurrentLocation(c.Room, location)
			message := logic.ResponseMsg{Status: "OK", Location: location}
			c.Pool.Broadcast <- message
		}
		if clientReq.Command == "submit_location" {
			var distance = lobby.CalculateDistance(c.Room, clientReq.Location)
			lobby.AddToResults(c.Room, c.ID, distance)

			message := logic.ResponseMsg{Status: "OK", Results: lobby.LobbyMap[c.Room].Results}
			c.Pool.Broadcast <- message
		}
	}
}
