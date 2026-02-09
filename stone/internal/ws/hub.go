package ws

import (
	"sync"
)

// Hub maintains the set of active clients and broadcasts messages to rooms.
type Hub struct {
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool
	register   chan *Client
	unregister chan *Client
	join       chan *roomMembershipChange
	leave      chan *roomMembershipChange
	broadcast  chan *roomMessage
	mu         sync.RWMutex
}

type roomMessage struct {
	roomID  string
	message *OutgoingMessage
	exclude *Client
}

type roomMembershipChange struct {
	roomID string
	client *Client
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		join:       make(chan *roomMembershipChange, 256),
		leave:      make(chan *roomMembershipChange, 256),
		broadcast:  make(chan *roomMessage, 256),
	}
}

// Run starts the hub's main event loop. Must be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

		case client := <-h.unregister:
			if !h.removeClient(client) {
				continue
			}
			close(client.send)

		case change := <-h.join:
			h.joinRoom(change.roomID, change.client)

		case change := <-h.leave:
			h.leaveRoom(change.roomID, change.client)

		case msg := <-h.broadcast:
			h.broadcastToRoom(msg.roomID, msg.message, msg.exclude)
		}
	}
}

func (h *Hub) removeClient(client *Client) bool {
	h.mu.Lock()
	if _, ok := h.clients[client]; !ok {
		h.mu.Unlock()
		return false
	}
	delete(h.clients, client)

	for roomID, members := range h.rooms {
		if _, ok := members[client]; ok {
			delete(members, client)
			if len(members) == 0 {
				delete(h.rooms, roomID)
			}
		}
	}
	h.mu.Unlock()

	return true
}

func (h *Hub) joinRoom(roomID string, client *Client) {
	h.mu.Lock()
	if _, ok := h.rooms[roomID]; !ok {
		h.rooms[roomID] = make(map[*Client]bool)
	}
	h.rooms[roomID][client] = true
	h.mu.Unlock()
}

func (h *Hub) leaveRoom(roomID string, client *Client) {
	h.mu.Lock()
	members, ok := h.rooms[roomID]
	if !ok {
		h.mu.Unlock()
		return
	}
	delete(members, client)
	if len(members) == 0 {
		delete(h.rooms, roomID)
	}
	h.mu.Unlock()
}

func (h *Hub) broadcastToRoom(roomID string, msg *OutgoingMessage, exclude *Client) {
	h.mu.RLock()
	members, ok := h.rooms[roomID]
	if !ok {
		h.mu.RUnlock()
		return
	}
	receivers := make([]*Client, 0, len(members))
	for client := range members {
		if client == exclude {
			continue
		}
		receivers = append(receivers, client)
	}
	h.mu.RUnlock()

	for _, client := range receivers {
		select {
		case client.send <- msg:
		default:
			h.mu.Lock()
			if members, ok := h.rooms[roomID]; ok {
				delete(members, client)
				if len(members) == 0 {
					delete(h.rooms, roomID)
				}
			}
			if _, exists := h.clients[client]; exists {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()
		}
	}
}

// JoinRoom adds a client subscription to a room channel.
func (h *Hub) JoinRoom(roomID string, client *Client) {
	h.join <- &roomMembershipChange{
		roomID: roomID,
		client: client,
	}
}

// LeaveRoom removes a client subscription from a room channel.
func (h *Hub) LeaveRoom(roomID string, client *Client) {
	h.leave <- &roomMembershipChange{
		roomID: roomID,
		client: client,
	}
}

// BroadcastToRoom sends a message to all clients in a room.
func (h *Hub) BroadcastToRoom(roomID string, msg *OutgoingMessage) {
	h.broadcast <- &roomMessage{
		roomID:  roomID,
		message: msg,
	}
}

// GetRoomMemberCount returns the number of connected clients in a room.
func (h *Hub) GetRoomMemberCount(roomID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if members, ok := h.rooms[roomID]; ok {
		return len(members)
	}
	return 0
}
