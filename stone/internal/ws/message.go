package ws

import "time"

// Message types
const (
	TypeJoinRoom      = "join_room"
	TypeLeaveRoom     = "leave_room"
	TypeRoomPresence  = "room_presence"
	TypeUserJoined    = "user_joined"
	TypeUserLeft      = "user_left"
	TypeNotification  = "notification"
)

// IncomingMessage is a message from a client.
type IncomingMessage struct {
	Type   string `json:"type"`
	RoomID string `json:"room_id,omitempty"`
}

// OutgoingMessage is a message sent to clients.
type OutgoingMessage struct {
	Type        string    `json:"type"`
	RoomID      string    `json:"room_id,omitempty"`
	UserID      string    `json:"user_id,omitempty"`
	MemberCount int       `json:"member_count,omitempty"`
	Data        any       `json:"data,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}
