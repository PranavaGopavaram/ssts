package api

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

// WSClient represents a WebSocket client
type WSClient struct {
	hub  *WebSocketHub
	conn *websocket.Conn
	send chan []byte
}

// WebSocketHub maintains the set of active clients and broadcasts messages to them
type WebSocketHub struct {
	// Registered clients
	clients map[*WSClient]bool

	// Inbound messages from the clients
	broadcast chan []byte

	// Register requests from the clients
	register chan *WSClient

	// Unregister requests from clients
	unregister chan *WSClient
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		broadcast:  make(chan []byte),
		register:   make(chan *WSClient),
		unregister: make(chan *WSClient),
		clients:    make(map[*WSClient]bool),
	}
}

// Run starts the WebSocket hub
func (h *WebSocketHub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("WebSocket client connected, total clients: %d", len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				log.Printf("WebSocket client disconnected, total clients: %d", len(h.clients))
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}

// BroadcastMessage broadcasts a message to all connected clients
func (h *WebSocketHub) BroadcastMessage(messageType string, data interface{}) {
	message := WSMessage{
		Type:      messageType,
		Timestamp: time.Now(),
		Data:      data,
	}

	jsonData, err := json.Marshal(message)
	if err != nil {
		log.Printf("Error marshaling WebSocket message: %v", err)
		return
	}

	select {
	case h.broadcast <- jsonData:
	default:
		log.Printf("WebSocket broadcast channel full, dropping message")
	}
}

// BroadcastTestUpdate broadcasts test execution updates
func (h *WebSocketHub) BroadcastTestUpdate(testID string, status string, data interface{}) {
	h.BroadcastMessage("test_update", map[string]interface{}{
		"test_id": testID,
		"status":  status,
		"data":    data,
	})
}

// BroadcastMetrics broadcasts real-time metrics
func (h *WebSocketHub) BroadcastMetrics(testID string, metrics interface{}) {
	h.BroadcastMessage("metrics_update", map[string]interface{}{
		"test_id": testID,
		"metrics": metrics,
	})
}

// BroadcastSystemMetrics broadcasts system-wide metrics
func (h *WebSocketHub) BroadcastSystemMetrics(metrics interface{}) {
	h.BroadcastMessage("system_metrics", metrics)
}

// BroadcastAlert broadcasts alert messages
func (h *WebSocketHub) BroadcastAlert(alertType string, message string, severity string) {
	h.BroadcastMessage("alert", map[string]interface{}{
		"type":     alertType,
		"message":  message,
		"severity": severity,
	})
}

// readPump pumps messages from the websocket connection to the hub
func (c *WSClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages from client
		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *WSClient) writePump() {
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

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Batch messages if available
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
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

// handleMessage handles incoming messages from WebSocket clients
func (c *WSClient) handleMessage(message []byte) {
	var msg WSMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("Error unmarshaling WebSocket message: %v", err)
		return
	}

	switch msg.Type {
	case "subscribe":
		// Handle subscription to specific test updates
		if testID, ok := msg.Data.(map[string]interface{})["test_id"].(string); ok {
			log.Printf("Client subscribed to test: %s", testID)
			// TODO: Implement per-test subscriptions
		}

	case "unsubscribe":
		// Handle unsubscription
		if testID, ok := msg.Data.(map[string]interface{})["test_id"].(string); ok {
			log.Printf("Client unsubscribed from test: %s", testID)
			// TODO: Implement per-test unsubscriptions
		}

	case "ping":
		// Respond to ping with pong
		pongMessage := WSMessage{
			Type:      "pong",
			Timestamp: time.Now(),
			Data:      msg.Data,
		}
		if jsonData, err := json.Marshal(pongMessage); err == nil {
			select {
			case c.send <- jsonData:
			default:
				close(c.send)
			}
		}

	default:
		log.Printf("Unknown WebSocket message type: %s", msg.Type)
	}
}

// WSMessage represents a WebSocket message
type WSMessage struct {
	Type      string      `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Data      interface{} `json:"data"`
}
