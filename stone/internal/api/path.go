package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/pulse/stone/internal/middleware"
	"github.com/pulse/stone/internal/service"
)

type createPathRequest struct {
	Title       string          `json:"title" binding:"required"`
	Description string          `json:"description"`
	Items       []pathItemInput `json:"items" binding:"required,min=1"`
}

type pathItemInput struct {
	ContentID string `json:"content_id" binding:"required"`
	Note      string `json:"note"`
}

func (s *Server) CreatePath(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var req createPathRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	items := make([]service.PathItemInput, len(req.Items))
	for i, item := range req.Items {
		contentID, err := uuid.Parse(item.ContentID)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content_id"})
			return
		}
		items[i] = service.PathItemInput{
			ContentID: contentID,
			Note:      item.Note,
		}
	}

	path, err := s.pathService.Create(userID, req.Title, req.Description, items)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create path"})
		return
	}
	c.JSON(http.StatusCreated, path)
}

func (s *Server) ListPaths(c *gin.Context) {
	cursor := c.Query("cursor")
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	paths, nextCursor, hasMore, err := s.pathService.List(limit, cursor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list paths"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       paths,
		"next_cursor": nextCursor,
		"has_more":    hasMore,
	})
}

func (s *Server) GetPath(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path id"})
		return
	}

	path, err := s.pathService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "path not found"})
		return
	}
	c.JSON(http.StatusOK, path)
}

func (s *Server) FollowPath(c *gin.Context) {
	pathID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path id"})
		return
	}

	userID := middleware.GetUserID(c)
	if err := s.pathService.Follow(pathID, userID); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "following path"})
}

func (s *Server) UnfollowPath(c *gin.Context) {
	pathID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid path id"})
		return
	}

	userID := middleware.GetUserID(c)
	if err := s.pathService.Unfollow(pathID, userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "unfollowed path"})
}
