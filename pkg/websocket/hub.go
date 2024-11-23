package websocket

import (
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
		slog.Warn("Hub: defer return")
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

		case message := <-hub.Broadcast:
			for client := range hub.Clients {
				// TODO: use buffered channel to prevent blocking
				client.Send <- message

				// select {
				// case client.Send <- message:
				// default:
				// 	slog.Warn("client.Send full, disconnecting client")
				// 	close(client.Send)
				// 	delete(hub.Clients, client)
				// }
			}

		}
	}
}
