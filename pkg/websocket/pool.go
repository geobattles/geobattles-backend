package websocket

import (
	"example/web-service-gin/pkg/logic"
	"fmt"

	"github.com/gorilla/websocket"
)

type Pool struct {
	Register   chan *Client
	Unregister chan *Client
	Rooms      map[string]map[*websocket.Conn]bool
	Broadcast  chan logic.ResponseMsg
}

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Rooms:      make(map[string]map[*websocket.Conn]bool),
		Broadcast:  make(chan logic.ResponseMsg),
	}
}

func (pool *Pool) Start() {
	for {
		select {
		case client := <-pool.Register:

			connections := pool.Rooms[client.Room]

			if connections == nil {
				connections = make(map[*websocket.Conn]bool)
				pool.Rooms[client.Room] = connections
			}
			pool.Rooms[client.Room][client.Conn] = true
			fmt.Println(pool.Rooms)

			// pool.Clients[client] = true
			// fmt.Println("Register, Size of Connection Pool: ", len(pool.Rooms[id]))
			// for client, _ := range pool.Clients {
			// 	fmt.Println(client)
			// 	client.Conn.WriteJSON(Message{Type: 1, Body: "New User Joined..."})
			// }
			break
		case client := <-pool.Unregister:
			fmt.Println("UNREGISTERING")
			fmt.Println(client)
			delete(pool.Rooms[client.Room], client.Conn)
			fmt.Println("Unregister, Size of Connection Pool: ", len(pool.Rooms[client.Room]))
			// for client, _ := range pool.Clients {
			// 	client.Conn.WriteJSON(Message{Type: 1, Body: "User Disconnected..."})
			// }
			break
		case message := <-pool.Broadcast:
			fmt.Println(message)
			fmt.Println("Broadcast, Sending message to all clients in Pool")
			for client, _ := range pool.Rooms[message.Room] {
				if err := client.WriteJSON(message); err != nil {
					fmt.Println(err)
					return
				}
			}
		}
	}
}
