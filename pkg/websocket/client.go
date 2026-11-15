package websocket

import (
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait      = 5 * time.Second
	PongWait       = 10 * time.Second
	pingPeriod     = 3 * time.Second
	closeWait      = 250 * time.Millisecond
	maxMessageSize = 16 << 10
	outboundLimit  = 64
)

type commandMsg struct {
	Cmd string `json:"command"`
}

// Identity, callbacks and lobby reference are immutable after construction.
// Only Write owns data writes; Close may interrupt I/O concurrently. send is
// private and never closed, so producers cannot race a close into a send panic.
type Client struct {
	ID    string
	Name  string
	Lobby interface{}

	hub          *Hub
	conn         *websocket.Conn
	send         chan []byte
	done         chan struct{}
	closeOnce    sync.Once
	unavailable  atomic.Bool
	overloaded   atomic.Bool
	handler      func(*Client, []byte)
	disconnected func(*Client)
}

func NewClient(id, name string, hub *Hub, conn *websocket.Conn, lobby interface{}, handler func(*Client, []byte), disconnected func(*Client)) *Client {
	return &Client{ID: id, Name: name, Lobby: lobby, hub: hub, conn: conn,
		send: make(chan []byte, outboundLimit), done: make(chan struct{}),
		handler: handler, disconnected: disconnected}
}

func (c *Client) Done() <-chan struct{} { return c.done }

func (c *Client) Closed() bool {
	if c.unavailable.Load() {
		return true
	}
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

// message must be an immutable, owned encoding. Only Hub calls this method.
func (c *Client) tryEnqueue(message []byte) bool {
	if c.Closed() {
		return false
	}
	select {
	case c.send <- message:
		return true
	default:
		c.overloaded.Store(true)
		c.unavailable.Store(true)
		return false
	}
}

// Enqueue is for stateless replies outside a game transaction. Game events use
// Hub.Send/Broadcast and defer Close until all game/transport locks are released.
func (c *Client) Enqueue(message []byte) {
	if !c.hub.Send(c, message) {
		c.Close()
	}
}

func (c *Client) Close() {
	if c.overloaded.Load() {
		c.CloseWithCode(websocket.CloseTryAgainLater, "Slow connection; reconnect for a snapshot")
		return
	}
	c.CloseWithCode(websocket.CloseGoingAway, "Connection closed")
}

// Close control frames may run concurrently with the sole data writer (Gorilla
// permits this). The short deadline also bounds a blocked writer/close race.
// Membership callbacks run once, after transport locks and Once.Do are released.
func (c *Client) CloseWithCode(code int, reason string) {
	// Callback is outside Once.Do: a game cleanup may request this same Close.
	notify := false
	c.closeOnce.Do(func() {
		c.unavailable.Store(true)
		close(c.done)
		c.hub.UnregisterClient(c)
		_ = c.conn.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(code, reason), time.Now().Add(closeWait))
		_ = c.conn.Close()
		notify = true
	})
	if notify && c.disconnected != nil {
		c.disconnected(c)
	}
}

func (c *Client) Read() {
	defer c.Close()
	c.conn.SetReadLimit(maxMessageSize)
	refreshDeadline := func() error { return c.conn.SetReadDeadline(time.Now().Add(PongWait)) }
	if refreshDeadline() != nil {
		return
	}
	c.conn.SetPongHandler(func(string) error { return refreshDeadline() })
	// Acknowledge peer close from the common bounded close path, not twice.
	c.conn.SetCloseHandler(func(int, string) error { return nil })
	for {
		kind, message, err := c.conn.ReadMessage()
		if err != nil {
			var peerClose *websocket.CloseError
			if errors.As(err, &peerClose) {
				switch peerClose.Code {
				case websocket.CloseNormalClosure, websocket.CloseGoingAway:
					c.CloseWithCode(peerClose.Code, "Peer closed")
				case websocket.CloseNoStatusReceived:
					c.CloseWithCode(websocket.CloseNormalClosure, "Peer closed")
				}
			}
			return
		}
		if c.Closed() {
			return
		}
		if kind != websocket.TextMessage {
			c.CloseWithCode(websocket.CloseUnsupportedData, "JSON text commands required")
			return
		}
		var cmd commandMsg
		if json.Unmarshal(message, &cmd) == nil && cmd.Cmd == "pong" {
			if refreshDeadline() != nil {
				return
			}
			continue
		}
		c.handler(c, message)
	}
}

func (c *Client) Write() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	defer c.Close()
	for {
		if c.Closed() {
			return
		}
		select {
		case <-c.done:
			return
		case message := <-c.send:
			if c.Closed() || c.conn.SetWriteDeadline(time.Now().Add(writeWait)) != nil {
				return
			}
			if c.conn.WriteMessage(websocket.TextMessage, message) != nil {
				return
			}
		case <-ticker.C:
			if c.Closed() || c.conn.SetWriteDeadline(time.Now().Add(writeWait)) != nil {
				return
			}
			// Browsers answer control pings even when JavaScript is backgrounded.
			// Keep the established JSON heartbeat for older clients as well.
			if c.conn.WriteMessage(websocket.PingMessage, nil) != nil {
				return
			}
			if c.conn.WriteJSON(commandMsg{Cmd: "ping"}) != nil {
				return
			}
		}
	}
}
