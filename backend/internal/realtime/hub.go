package realtime

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = 50 * time.Second
	sendBuffer = 32
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// Open for development. In production, only the frontend's address should be allowed.
	CheckOrigin: func(r *http.Request) bool { return true },
}

type client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

// Hub keeps all connected clients and sends messages to them.
type Hub struct {
	mu      sync.RWMutex
	clients map[*client]struct{}
}

func NewHub() *Hub {
	return &Hub{clients: make(map[*client]struct{})}
}

func (h *Hub) add(c *client) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = struct{}{}
	return len(h.clients)
}

func (h *Hub) remove(c *client) int {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c]; ok {
		delete(h.clients, c)
		close(c.send)
	}
	return len(h.clients)
}

// Broadcast sends a message to every connected client without blocking.
func (h *Hub) Broadcast(message []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.clients {
		select {
		case c.send <- message:
		default:
			log.Println("WebSocket client too slow, message skipped")
		}
	}
}

// ServeWS handles GET /ws/alerts and upgrades the connection to a WebSocket.
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	c := &client{hub: h, conn: conn, send: make(chan []byte, sendBuffer)}
	total := h.add(c)
	log.Printf("WebSocket client connected (%d connected)", total)

	go c.writePump()
	go c.readPump()
}

// readPump notices when the client disconnects and keeps the connection alive.
func (c *client) readPump() {
	defer func() {
		total := c.hub.remove(c)
		c.conn.Close()
		log.Printf("WebSocket client disconnected (%d connected)", total)
	}()

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

// writePump sends queued messages and regular pings to the client.
func (c *client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}