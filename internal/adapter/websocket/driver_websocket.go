package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID       string
	Conn     *websocket.Conn
	Send     chan []byte
	UserType string // "driver" or "passenger"
}

type Hub struct {
	Clients    map[string]*Client // driver_id -> Client
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan BroadcastMessage
}

type BroadcastMessage struct {
	DriverIDs []string
	Message   interface{}
	Type      string
}

type WSMessage struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp time.Time   `json:"timestamp"`
}

func NewHub() *Hub {
	return &Hub{
		Clients:    make(map[string]*Client),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan BroadcastMessage),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.Clients[client.ID] = client
			log.Printf("Client registered: %s (%s)", client.ID, client.UserType)

		case client := <-h.Unregister:
			if _, ok := h.Clients[client.ID]; ok {
				delete(h.Clients, client.ID)
				close(client.Send)
				log.Printf("Client unregistered: %s", client.ID)
			}

		case message := <-h.Broadcast:
			h.broadcastToDrivers(message)
		}
	}
}

func (h *Hub) broadcastToDrivers(message BroadcastMessage) {
	wsMessage := WSMessage{
		Type:      message.Type,
		Payload:   message.Message,
		Timestamp: time.Now(),
	}

	data, err := json.Marshal(wsMessage)
	if err != nil {
		log.Printf("Failed to marshal broadcast message: %v", err)
		return
	}

	for _, driverID := range message.DriverIDs {
		if client, ok := h.Clients[driverID]; ok {
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(h.Clients, driverID)
			}
		}
	}
}

func (h *Hub) SendToDriver(driverID string, messageType string, payload interface{}) {
	if client, ok := h.Clients[driverID]; ok {
		wsMessage := WSMessage{
			Type:      messageType,
			Payload:   payload,
			Timestamp: time.Now(),
		}

		data, err := json.Marshal(wsMessage)
		if err != nil {
			log.Printf("Failed to marshal message for driver %s: %v", driverID, err)
			return
		}

		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(h.Clients, driverID)
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second) // Ping interval
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				// Channel closed
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			// Send ping message
			if err := c.Conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512) // 512 bytes
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			break
		}

		// Handle incoming messages
		var wsMsg WSMessage
		if err := json.Unmarshal(message, &wsMsg); err != nil {
			log.Printf("Failed to unmarshal WebSocket message: %v", err)
			continue
		}

		c.handleMessage(wsMsg)
	}
}

func (c *Client) handleMessage(msg WSMessage) {
	switch msg.Type {
	case "pong":
		// Update read deadline
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	case "location_update":
		// Handle location update from driver
		log.Printf("Location update from driver %s", c.ID)
	case "ride_offer_response":
		// Handle driver's response to ride offer
		log.Printf("Ride offer response from driver %s", c.ID)
	default:
		log.Printf("Unknown message type from driver %s: %s", c.ID, msg.Type)
	}
}
