package ws

import (
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 4096
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub           *Hub
	conn          *websocket.Conn
	send          chan *OutgoingMessage
	userID        uuid.UUID
	canAccessRoom func(roomID string) bool
}

func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID, canAccessRoom func(roomID string) bool) *Client {
	if canAccessRoom == nil {
		canAccessRoom = func(string) bool { return false }
	}
	return &Client{
		hub:           hub,
		conn:          conn,
		send:          make(chan *OutgoingMessage, 256),
		userID:        userID,
		canAccessRoom: canAccessRoom,
	}
}

func (c *Client) UserID() uuid.UUID {
	return c.userID
}

// ReadPump pumps messages from the websocket connection to the hub.
func (c *Client) ReadPump() {
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
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("ws error: %v", err)
			}
			break
		}

		var msg IncomingMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg.Type {
		case TypeJoinRoom:
			if msg.RoomID != "" && c.canAccessRoom(msg.RoomID) {
				c.hub.JoinRoom(msg.RoomID, c)
			}
		case TypeLeaveRoom:
			if msg.RoomID != "" {
				c.hub.LeaveRoom(msg.RoomID, c)
			}
		}
	}
}

// WritePump pumps messages from the hub to the websocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteJSON(msg); err != nil {
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

// Register registers the client with the hub.
func (c *Client) Register() {
	c.hub.register <- c
}
