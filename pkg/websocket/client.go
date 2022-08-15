package websocket

import (
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

		if clientReq.Message == "start" {
			fmt.Println("start was read")

			logic.LastSentLoc = logic.GenerateRndLocation()
			message := logic.ResponseMsg{Status: "OK", Location: logic.LastSentLoc, Room: c.Room}
			fmt.Println(message)
			c.Pool.Broadcast <- message
		}

	}
}
