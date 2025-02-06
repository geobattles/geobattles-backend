package websocket

import (
	"log/slog"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 1024
)

type Client struct {
	ID string
	// Room           string
	Name           string
	Hub            *Hub
	Conn           *websocket.Conn
	Send           chan interface{}
	MessageHandler func(c *Client, message []byte)
	Lobby          interface{}
}

// goroutine to read messages from client
func (c *Client) Read() {
	defer func() {
		slog.Debug("Defer read")
		c.MessageHandler(c, []byte("{\"command\":\"disconnect\"}"))
		c.Hub.Unregister <- c
		// go func() {
		// 	client.Hub.Broadcast <- &models.ClientResp{Status: "OK", Type: "LEFT_LOBBY", User: client.ID, Lobby: lobby.LobbyMap[client.Room]}
		// }()
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error { c.Conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			slog.Warn("Error reading message", "error", err.Error())
			return
		}
		slog.Debug("Received message", "message", string(message))
		if c.MessageHandler != nil {
			c.MessageHandler(c, message)
		}
	}
}

func (c *Client) Write() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		slog.Debug("Defer write")
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				slog.Debug("Hub closed channel")
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteJSON(message); err != nil {
				slog.Warn("Error writing unicast", "error", err)
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			slog.Debug("Sending ping")
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				slog.Error("Error sending ping", "error", err)
				return
			}
		}
	}
}
