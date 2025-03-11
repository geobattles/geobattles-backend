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
	PongWait = 10 * time.Second

	// Send pings to peer with this period. Must be less than PongWait.
	pingPeriod = 1 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 1024
)

type commandMsg struct {
	Cmd string `json:"command"`
}

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
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(PongWait))

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			slog.Warn("Error reading message", "error", err.Error())
			return
		}

		// // Currently handled by c.MessageHandler, might be moved here in the future
		// // Check if this is a pong message
		// var cmdMsg commandMsg
		// if err := json.Unmarshal(message, &cmdMsg); err == nil && cmdMsg.Cmd == "pong" {
		//     // This is a pong response, reset the deadline
		//     c.Conn.SetReadDeadline(time.Now().Add(PongWait))
		//     slog.Debug("Received pong message")
		//     continue // Skip further processing for pong messages
		// }

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

			pingMsg := commandMsg{Cmd: "ping"}
			if err := c.Conn.WriteJSON(pingMsg); err != nil {
				slog.Error("Error sending ping", "error", err)
				return
			}
		}
	}
}
