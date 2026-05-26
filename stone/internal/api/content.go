package api

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/pulse/stone/internal/middleware"
	"github.com/pulse/stone/internal/model"
	// model.IsValidReactionKind used for reaction validation
)

func (s *Server) CreateContent(c *gin.Context) {
	userID := middleware.GetUserID(c)
	maxBodyBytes := int64(s.cfg.StorageMaxSizeMB+1) * 1024 * 1024
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBodyBytes)

	contentType := c.PostForm("content_type")
	if contentType == "" {
		contentType = model.ContentTypeImage
	}

	body := c.PostForm("body")
	mediaAssetIDRaw := strings.TrimSpace(c.PostForm("media_asset_id"))

	// Tags are user-defined hashtags, sent as comma-separated or multiple form values
	var tagNames []string
	for _, raw := range c.PostFormArray("tags") {
		for _, t := range strings.Split(raw, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tagNames = append(tagNames, t)
			}
		}
	}

	// For text posts, no file needed. Media posts use either direct multipart
	// upload or a ready media_asset_id from the async media pipeline.
	var filename string
	file, header, err := c.Request.FormFile("file")
	if err == nil {
		defer file.Close()
		filename = header.Filename
	} else if contentType != model.ContentTypeText && mediaAssetIDRaw == "" {
		if strings.Contains(err.Error(), "http: request body too large") {
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "payload too large"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required for non-text content"})
		return
	}

	var content *model.Content
	if mediaAssetIDRaw != "" {
		mediaAssetID, err := uuid.Parse(mediaAssetIDRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid media_asset_id"})
			return
		}
		content, err = s.contentService.CreateFromMediaAsset(userID, contentType, mediaAssetID, body, tagNames)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	} else {
		content, err = s.contentService.Create(userID, contentType, filename, file, body, tagNames)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, content)
}

func (s *Server) GetContent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content id"})
		return
	}

	content, err := s.contentService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "content not found"})
		return
	}
	c.JSON(http.StatusOK, content)
}

func (s *Server) DeleteContent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content id"})
		return
	}

	userID := middleware.GetUserID(c)
	if err := s.contentService.Delete(id, userID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// ListUserContent lists content for a given user with cursor-based pagination.
func (s *Server) ListUserContent(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	cursor := c.Query("cursor")
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	items, nextCursor, err := s.contentService.ListByUser(userID, limit, cursor)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load user content"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items":       items,
		"next_cursor": nextCursor,
		"has_more":    nextCursor != "",
	})
}

// ReactToContent adds a semantic reaction to content.
func (s *Server) ReactToContent(c *gin.Context) {
	contentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content id"})
		return
	}

	var req struct {
		Kind string `json:"kind" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate reaction kind against the single source of truth.
	if !model.IsValidReactionKind(req.Kind) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid reaction kind"})
		return
	}

	userID := middleware.GetUserID(c)
	if err := s.contentService.React(userID, contentID, req.Kind); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "already reacted"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "reaction added"})
}

// RemoveReaction removes a semantic reaction.
func (s *Server) RemoveReaction(c *gin.Context) {
	contentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid content id"})
		return
	}

	kind := c.Query("kind")
	if kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kind parameter required"})
		return
	}

	userID := middleware.GetUserID(c)
	if err := s.contentService.RemoveReaction(userID, contentID, kind); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "reaction removed"})
}
