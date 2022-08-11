package websocket

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	Conn *websocket.Conn
	Pool *Pool
}

type Message struct {
	Type int    `json:"type"`
	Body string `json:"body"`
}

// func (c *Client) Read(context *gin.Context) {
func (c *Client) Read() {
	defer func() {
		fmt.Printf("Unregister funkcija")
		c.Pool.Unregister <- c
		c.Conn.Close()
	}()
	fmt.Printf("reading")
	for {
		//var clientReq logic.ClientReq
		fmt.Printf("reading inside for")
		time.Sleep(2000)

		// if err := context.BindJSON(&clientReq); err != nil {
		// 	return
		// }
		// fmt.Println(clientReq)
		// if clientReq.Message == "start" {
		// 	message := logic.ResponseMsg{Status: "OK", Location: logic.GenerateRndLocation()}
		// 	c.Pool.Broadcast <- message
		// 	fmt.Printf("Message Received: %+v\n", message)

		// }

	}
}
