package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/pulse/stone/internal/middleware"
	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/service"
)

func (s *Server) CreateAsk(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req service.AskInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ask, bridges, err := s.adviceService.CreateAsk(userID, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"ask": ask, "bridges": bridges})
}

func (s *Server) GetAskBridges(c *gin.Context) {
	userID := middleware.GetUserID(c)
	askID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ask id"})
		return
	}

	bridges, err := s.adviceService.GetAskBridges(askID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrAdviceNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bridges)
}

func (s *Server) AskBridge(c *gin.Context) {
	s.bridgeAction(c, s.adviceService.AskBridge)
}

func (s *Server) RespondBridge(c *gin.Context) {
	s.bridgeAction(c, s.adviceService.RespondBridge)
}

func (s *Server) bridgeAction(
	c *gin.Context,
	fn func(uuid.UUID, uuid.UUID, service.BridgeActionInput) (*model.Bridge, error),
) {
	userID := middleware.GetUserID(c)
	bridgeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bridge id"})
		return
	}

	var req service.BridgeActionInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bridge, err := fn(bridgeID, userID, req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrAdviceNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, bridge)
}

func (s *Server) SignalBridge(c *gin.Context) {
	userID := middleware.GetUserID(c)
	bridgeID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid bridge id"})
		return
	}

	var req service.HelpSignalInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	signal, err := s.adviceService.RecordSignal(bridgeID, userID, req)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrAdviceNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, signal)
}

func (s *Server) GetToday(c *gin.Context) {
	userID := middleware.GetUserID(c)
	today, err := s.adviceService.GetToday(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load today"})
		return
	}
	c.JSON(http.StatusOK, today)
}

func (s *Server) ListHelpSessions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	sessions, err := s.adviceService.ListHelpSessions(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to load help sessions"})
		return
	}
	c.JSON(http.StatusOK, sessions)
}

func (s *Server) JoinHelpSession(c *gin.Context) {
	userID := middleware.GetUserID(c)
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid session id"})
		return
	}
	session, err := s.adviceService.JoinHelpSession(sessionID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		if errors.Is(err, service.ErrAdviceNotFound) {
			status = http.StatusNotFound
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, session)
}

func (s *Server) UpdateTrustProfile(c *gin.Context) {
	userID := middleware.GetUserID(c)
	var req service.TrustProfileInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	profile, err := s.adviceService.UpdateTrustProfile(userID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update trust profile"})
		return
	}
	c.JSON(http.StatusOK, profile)
}
