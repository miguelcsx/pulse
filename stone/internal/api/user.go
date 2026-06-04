package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/pulse/stone/internal/middleware"
)

type updateMeRequest struct {
	DisplayName *string `json:"display_name"`
	Bio         *string `json:"bio"`
	Location    *string `json:"location"`
	AvatarURL   *string `json:"avatar_url"`
}

func (s *Server) GetMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	user, err := s.userService.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, toSelfUserResponse(user))
}

func (s *Server) UpdateMe(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req updateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := s.userService.Update(userID, req.DisplayName, req.Bio, req.Location, req.AvatarURL)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
		return
	}
	c.JSON(http.StatusOK, toSelfUserResponse(user))
}

type publicUserResponse struct {
	ID          string `json:"id"`
	Handle      string `json:"handle"`
	DisplayName string `json:"display_name"`
	Bio         string `json:"bio"`
	AvatarURL   string `json:"avatar_url"`
	Location    string `json:"location"`
}

func (s *Server) SearchUsers(c *gin.Context) {
	viewerID := middleware.GetUserID(c)
	query := c.Query("q")

	users, err := s.userService.Search(viewerID, query, 25)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to search people"})
		return
	}

	results := make([]publicUserResponse, len(users))
	for i, u := range users {
		results[i] = publicUserResponse{
			ID:          u.ID.String(),
			Handle:      u.Handle,
			DisplayName: u.DisplayName,
			Bio:         u.Bio,
			AvatarURL:   u.AvatarURL,
			Location:    u.Location,
		}
	}
	c.JSON(http.StatusOK, results)
}

func (s *Server) GetUser(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	viewerID := middleware.GetUserID(c)
	profile, err := s.userService.GetProfile(id, viewerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, profile)
}
