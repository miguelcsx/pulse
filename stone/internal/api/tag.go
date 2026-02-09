package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) ListTags(c *gin.Context) {
	tags, err := s.tagService.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list tags"})
		return
	}
	c.JSON(http.StatusOK, tags)
}

func (s *Server) SearchTags(c *gin.Context) {
	q := c.Query("q")
	if q == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "q parameter is required"})
		return
	}

	tags, err := s.tagService.Search(q)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search tags"})
		return
	}
	c.JSON(http.StatusOK, tags)
}
