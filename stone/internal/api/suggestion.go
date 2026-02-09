package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/pulse/stone/internal/middleware"
)

func (s *Server) GetSuggestions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	limit := 5
	if raw := c.Query("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	suggestions, err := s.affinityService.GetSuggestions(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get suggestions"})
		return
	}

	c.JSON(http.StatusOK, suggestions)
}
