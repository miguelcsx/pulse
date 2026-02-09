package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/pulse/stone/internal/middleware"
	"github.com/pulse/stone/internal/service"
	"github.com/pulse/stone/internal/ws"
)

func (s *Server) ListRooms(c *gin.Context) {
	rooms, err := s.roomService.ListActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list rooms"})
		return
	}
	c.JSON(http.StatusOK, rooms)
}

func (s *Server) EnterRoom(c *gin.Context) {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	userID := middleware.GetUserID(c)
	if err := s.roomService.Enter(roomID, userID); err != nil {
		if errors.Is(err, service.ErrRoomNotFoundOrExpired) {
			c.JSON(http.StatusNotFound, gin.H{"error": "room not found or expired"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	count, _ := s.roomService.GetMemberCount(roomID)
	s.hub.BroadcastToRoom(roomID.String(), &ws.OutgoingMessage{
		Type:        ws.TypeRoomPresence,
		RoomID:      roomID.String(),
		MemberCount: int(count),
		Timestamp:   time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "entered room", "member_count": count})
}

func (s *Server) LeaveRoom(c *gin.Context) {
	roomID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	userID := middleware.GetUserID(c)
	if err := s.roomService.Leave(roomID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	count, _ := s.roomService.GetMemberCount(roomID)
	s.hub.BroadcastToRoom(roomID.String(), &ws.OutgoingMessage{
		Type:        ws.TypeRoomPresence,
		RoomID:      roomID.String(),
		MemberCount: int(count),
		Timestamp:   time.Now(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "left room", "member_count": count})
}
