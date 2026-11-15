package websocket

import "sync"

// Hub owns only transport membership. Callers may enqueue while holding a game
// lock: these methods never wait on peers, call game callbacks or close sockets.
// Lock order is lobby -> hub. Transport never calls into a lobby under hub.mu.
type Hub struct {
	mu      sync.Mutex
	clients map[*Client]struct{}
	done    chan struct{}
	closed  bool
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*Client]struct{}), done: make(chan struct{})}
}

func (h *Hub) RegisterClient(c *Client) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || c == nil || c.Closed() {
		return false
	}
	h.clients[c] = struct{}{}
	return true
}

func (h *Hub) UnregisterClient(c *Client) {
	h.mu.Lock()
	delete(h.clients, c)
	h.mu.Unlock()
}

// Broadcast enqueues one immutable encoded event per peer in caller order.
// A full/closed peer is returned for disconnection AFTER the game lock is released;
// it is never allowed to keep consuming a stream with an omitted event.
func (h *Hub) Broadcast(message []byte) (slow []*Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		if !c.tryEnqueue(message) {
			delete(h.clients, c)
			slow = append(slow, c)
		}
	}
	return slow
}

func (h *Hub) Send(c *Client, message []byte) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed || c == nil {
		return false
	}
	if _, ok := h.clients[c]; !ok {
		return false
	}
	if !c.tryEnqueue(message) {
		delete(h.clients, c)
		return false
	}
	return true
}

func (h *Hub) Done() <-chan struct{} { return h.done }

// Close has no socket I/O or game callback under its mutex. There is no hub
// goroutine/channel to leak when the last membership expires.
func (h *Hub) Close() {
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		return
	}
	h.closed = true
	close(h.done)
	clients := make([]*Client, 0, len(h.clients))
	for c := range h.clients {
		clients = append(clients, c)
	}
	h.clients = make(map[*Client]struct{})
	h.mu.Unlock()
	for _, c := range clients {
		c.Close()
	}
}
