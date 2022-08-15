package websocket

import (
	"example/web-service-gin/pkg/lobby"
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
	defer func() {
		fmt.Println("defer return pool")
	}()

	for {
		select {
		case client := <-pool.Register:
			fmt.Println("register new client")

			connections := pool.Rooms[client.Room]

			if connections == nil {
				connections = make(map[*websocket.Conn]bool)
				pool.Rooms[client.Room] = connections
			}
			pool.Rooms[client.Room][client.Conn] = true
			fmt.Println("pool.rooms LOOOG ", pool.Rooms)

			// pool.Clients[client] = true
			// fmt.Println("Register, Size of Connection Pool: ", len(pool.Rooms[id]))
			for clientConn, _ := range pool.Rooms[client.Room] {
				fmt.Println("looog client", client)
				for _, value := range lobby.LobbyList {
					if value.ID == client.Room {
						clientConn.WriteJSON(value)
						break
					}
				}
			}
			break
		case client := <-pool.Unregister:
			fmt.Println("UNREGISTERING")
			fmt.Println(client)
			delete(pool.Rooms[client.Room], client.Conn)
			fmt.Println("Unregister, Size of Connection Pool: ", len(pool.Rooms[client.Room]))
			fmt.Println(client.Room)
			lobby.RemovePlayerFromLobby(lobby.LobbyList, client.Name, client.Room)

			for clientConn, _ := range pool.Rooms[client.Room] {
				fmt.Println("looog client", client)
				for _, value := range lobby.LobbyList {
					if value.ID == client.Room {
						clientConn.WriteJSON(value)
						break
					}
				}
			}
			// for client, _ := range pool.Clients {
			// 	client.Conn.WriteJSON(Message{Type: 1, Body: "User Disconnected..."})
			// }
			break
		case message := <-pool.Broadcast:
			fmt.Println(message)
			fmt.Println("Broadcast, Sending message to all clients in Pool")
			for client, _ := range pool.Rooms[message.Room] {
				if err := client.WriteJSON(message); err != nil {
					fmt.Println("error writing broadcast", err)
					//return
				}
			}
		}
	}
}
