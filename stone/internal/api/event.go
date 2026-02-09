package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/pulse/stone/internal/middleware"
	"github.com/pulse/stone/internal/service"
)

type recordEventsRequest struct {
	Events []eventInput `json:"events" binding:"required,min=1"`
}

type eventInput struct {
	Type       string          `json:"type" binding:"required"`
	TargetType string          `json:"target_type"`
	TargetID   string          `json:"target_id"`
	Metadata   json.RawMessage `json:"metadata"`
}

func (s *Server) RecordEvents(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req recordEventsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	inputs := make([]service.EventInput, len(req.Events))
	for i, e := range req.Events {
		inputs[i] = service.EventInput{
			Type:       e.Type,
			TargetType: e.TargetType,
			TargetID:   e.TargetID,
			Metadata:   string(e.Metadata),
		}
	}

	if err := s.eventService.RecordBatch(userID, inputs); err != nil {
		if errors.Is(err, service.ErrInvalidEventInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record events"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "events recorded"})
}
