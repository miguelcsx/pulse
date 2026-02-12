package api

import (
	"encoding/json"
	"errors"
	"io"
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

	// Read raw body so we can try two formats.
	body, err := io.ReadAll(c.Request.Body)
	if err != nil || len(body) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "empty request body"})
		return
	}

	// Accept both `{ "events": [...] }` and a bare `[...]` array.
	var events []eventInput
	var req recordEventsRequest
	if err := json.Unmarshal(body, &req); err == nil && len(req.Events) > 0 {
		events = req.Events
	} else if err := json.Unmarshal(body, &events); err != nil || len(events) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "expected array of events or {\"events\": [...]}"})
		return
	}

	inputs := make([]service.EventInput, len(events))
	for i, e := range events {
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
