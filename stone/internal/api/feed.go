package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/pulse/stone/internal/middleware"
	"github.com/pulse/stone/internal/service"
)

func (s *Server) GetFeed(c *gin.Context) {
	userID := middleware.GetUserID(c)
	cursor := c.Query("cursor")

	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	start := time.Now()
	items, nextCursor, hasMore, err := s.feedService.GetFeed(userID, cursor, limit)
	elapsed := time.Since(start)

	c.Header("X-Feed-Duration-Ms", fmt.Sprintf("%.1f", float64(elapsed.Microseconds())/1000.0))

	if err != nil {
		if strings.Contains(err.Error(), "invalid cursor") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid cursor"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load feed"})
		return
	}

	// Inject suggestions on first page only
	var suggestions []service.SuggestionResult
	if cursor == "" {
		suggestions, _ = s.affinityService.GetSuggestions(userID, 3)
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
		"suggestions": suggestions,
	})
}
