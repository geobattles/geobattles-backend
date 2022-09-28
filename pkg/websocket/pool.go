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
	//Rooms      map[string]map[*websocket.Conn]bool
	Rooms    map[string]map[string]*websocket.Conn
	Transmit chan logic.RouteMsg
}

func NewPool() *Pool {
	return &Pool{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		//Rooms:      make(map[string]map[*websocket.Conn]bool),
		Rooms:    make(map[string]map[string]*websocket.Conn),
		Transmit: make(chan logic.RouteMsg),
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
			// if room doesnt exist yet create it, otherwise just add client to it
			if connections == nil {
				fmt.Println("creating new connection room")
				//connections = make(map[*websocket.Conn]bool)
				connections = make(map[string]*websocket.Conn)
				pool.Rooms[client.Room] = connections
			}
			//pool.Rooms[client.Room][client.Conn] = true
			pool.Rooms[client.Room][client.ID] = client.Conn
			fmt.Println("pool.rooms LOOOG ", pool.Rooms)

			// send updated list of players to every member of the lobby
			go func() {
				client.Pool.Transmit <- logic.RouteMsg{Room: client.Room, Data: logic.ClientResp{Status: "OK", Type: "JOINED_LOBBY", User: client.ID, Lobby: lobby.LobbyMap[client.Room]}}
			}()

		case client := <-pool.Unregister:
			fmt.Println("UNREGISTERING")
			delete(pool.Rooms[client.Room], client.ID)
			lobby.RemovePlayerFromLobby(client.ID, client.Room)
			//fmt.Println("pool.rooms LOOOG ", pool.Rooms)
			go func() {
				client.Pool.Transmit <- logic.RouteMsg{Room: client.Room, Data: logic.ClientResp{Status: "OK", Type: "LEFT_LOBBY", User: client.ID, Lobby: lobby.LobbyMap[client.Room]}}
			}()

		case message := <-pool.Transmit:
			fmt.Println("msg to send: ", message)
			// if message doesnt have connection field broadcast it
			// otherwise only send it to the connection given
			if message.Conn == nil {
				for _, client := range pool.Rooms[message.Room] {
					if err := client.WriteJSON(message.Data); err != nil {
						fmt.Println("error writing broadcast", err)
					}
				}
			} else {
				if err := message.Conn.WriteJSON(message.Data); err != nil {
					fmt.Println("error writing unicast", err)
				}
			}
		}
	}
}
