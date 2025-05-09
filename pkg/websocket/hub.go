package websocket

import (
	"log/slog"
	"sync"
)

type Hub struct {
	Register   chan *Client
	Unregister chan *Client
	Clients    map[*Client]bool
	Broadcast  chan interface{}
	mu         sync.Mutex
}

// Creates a new pool of client connections
func NewHub() *Hub {
	return &Hub{
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Clients:    make(map[*Client]bool),
		Broadcast:  make(chan interface{}),
	}
}

// Sends a message to all clients of the hub
func (hub *Hub) BroadcastMessage(message interface{}) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	for client := range hub.Clients {
		client.Send <- message
	}
}

// Regsiters a new client connection with the hub
func (hub *Hub) RegisterClient(client *Client) {
	hub.mu.Lock()
	defer hub.mu.Unlock()

	hub.Clients[client] = true
	slog.Info("Hub: registered new client", "client", client.ID)
}

// Starts the hub which handles client connections and broadcasts messages
func (hub *Hub) Start() {
	defer func() {
		slog.Warn("Hub: defer return")
	}()

	for {
		select {
		case client := <-hub.Register:
			hub.RegisterClient(client)

		case client := <-hub.Unregister:
			slog.Info("Hub: unregister client", "client", client.ID)
			if _, ok := hub.Clients[client]; ok {
				delete(hub.Clients, client)
				close(client.Send)
			}

		case message := <-hub.Broadcast:
			hub.BroadcastMessage(message)
		}
	}
}
