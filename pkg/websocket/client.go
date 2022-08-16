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

type Message struct {
	Type int    `json:"type"`
	Body string `json:"body"`
	Room string `json:"room"`
}

func (c *Client) Read() {
	defer func() {
		fmt.Println("defer read unregister")
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()

	fmt.Println("reading")
	for {
		var clientReq logic.ClientReq
		err := c.Conn.ReadJSON(&clientReq)

		if err != nil {
			fmt.Println("read json 1", err)
			return
		}
		fmt.Println(clientReq)

		if clientReq.Command == "start" {
			fmt.Println("start was read")
			var location logic.Coordinates = logic.GenerateRndLocation()
			lobby.MarkGameActive(c.Room)
			lobby.UpdateCurrentLocation(c.Room, location)
			//logic.LastSentLoc = logic.GenerateRndLocation()
			message := logic.ResponseMsg{Status: "OK", Location: location, Room: c.Room}
			fmt.Println(message)
			c.Pool.Broadcast <- message
		}
		if clientReq.Command == "submit_location" {
			var distance = lobby.CalculateDistance(c.Room, clientReq.Location)
			lobby.AddToResults(c.Room, c.ID, distance)

			//fmt.Println(message)

			message := logic.ResponseMsg{Status: "OK", Results: lobby.LobbyMap[c.Room].Results, Room: c.Room}
			fmt.Println("sending results to whole lobby")
			c.Pool.Broadcast <- message
			// for _, value := range lobby.LobbyList {
			// 	if value.ID == c.Room {
			// 		message := logic.ResponseMsg{Status: "OK", Results: value.Results, Room: c.Room}
			// 		fmt.Println("sending results to whole lobby")
			// 		c.Pool.Broadcast <- message
			// 		break
			// 	}
			// }

		}

	}
}
