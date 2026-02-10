package ws

import (
	"encoding/json"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"github.com/pulse/stone/internal/middleware"
)

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10

	// defaultMaxMessageSize is used when no explicit size is provided.
	defaultMaxMessageSize int64 = 4096
)

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	hub            *Hub
	conn           *websocket.Conn
	send           chan *OutgoingMessage
	userID         uuid.UUID
	canAccessRoom  func(roomID string) bool
	maxMessageSize int64
}

// ClientOption configures optional Client parameters.
type ClientOption func(*Client)

// WithMaxMessageSize sets the maximum inbound WebSocket message size in bytes.
func WithMaxMessageSize(size int64) ClientOption {
	return func(c *Client) {
		if size > 0 {
			c.maxMessageSize = size
		}
	}
}

func NewClient(hub *Hub, conn *websocket.Conn, userID uuid.UUID, canAccessRoom func(roomID string) bool, opts ...ClientOption) *Client {
	if canAccessRoom == nil {
		canAccessRoom = func(string) bool { return false }
	}
	c := &Client{
		hub:            hub,
		conn:           conn,
		send:           make(chan *OutgoingMessage, 256),
		userID:         userID,
		canAccessRoom:  canAccessRoom,
		maxMessageSize: defaultMaxMessageSize,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

func (c *Client) UserID() uuid.UUID {
	return c.userID
}

// ReadPump pumps messages from the websocket connection to the hub.
func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		middleware.WSConnectionClosed()
		c.conn.Close()
	}()

	c.conn.SetReadLimit(c.maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				slog.Warn("ws unexpected close", "error", err, "user_id", c.userID)
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
	middleware.WSConnectionOpened()
	c.hub.register <- c
}
