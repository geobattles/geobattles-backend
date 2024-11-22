package websocket

import (
	"fmt"
	"log/slog"
)

type Hub struct {
	Register   chan *Client
	Unregister chan *Client
	Clients    map[*Client]bool
	Broadcast  chan interface{}
}

func NewHub() *Hub {
	return &Hub{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan interface{}),
	}
}

func (hub *Hub) Start() {
	defer func() {
		fmt.Println("defer return hub")
	}()

	for {
		select {
		case client := <-hub.Register:
			slog.Info("Hub: register new client", "client", client.ID)
			hub.Clients[client] = true

		case client := <-hub.Unregister:
			slog.Info("Hub: unregister client", "client", client.ID)
			if _, ok := hub.Clients[client]; ok {
				delete(hub.Clients, client)
				close(client.Send)
			}

			// TODO: sending broadcast should be moved to wherever unregister is called
			// game.RemovePlayerFromLobby(client.ID, client.Room)
			//fmt.Println("pool.rooms LOOOG ", pool.Rooms)
			// go func() {
			// 	client.Hub.Broadcast <- &models.ClientResp{Status: "OK", Type: "LEFT_LOBBY", User: client.ID, Lobby: lobby.LobbyMap[client.Room]}
			// }()

		case message := <-hub.Broadcast:
			// fmt.Println("msg to send: ", message)
			// if message doesnt have connection field broadcast it
			// otherwise only send it to the connection given
			for client := range hub.Clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(hub.Clients, client)
				}
			}

		}
	}
}
