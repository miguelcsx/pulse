package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/pulse/stone/internal/middleware"
	"github.com/pulse/stone/internal/security"
	"github.com/pulse/stone/internal/ws"
)

func (s *Server) HandleWebSocket(c *gin.Context) {
	checkOrigin := middleware.NewOriginMatcher(s.cfg.WSOrigins)
	upgrader := websocket.Upgrader{
		Subprotocols: []string{"bearer"},
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			// Non-browser clients might not send Origin.
			if origin == "" {
				return true
			}
			return checkOrigin(origin)
		},
	}

	tokenStr := extractWebSocketToken(c)
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "token required"})
		return
	}

	parsed, err := security.ParseAndValidateJWT(tokenStr, s.cfg.JWTSecret, "access")
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}

	client := ws.NewClient(s.hub, conn, parsed.UserID, func(roomID string) bool {
		ok, err := s.roomService.IsActiveMember(roomID, parsed.UserID)
		return err == nil && ok
	})
	client.Register()

	go client.WritePump()
	go client.ReadPump()
}

func extractWebSocketToken(c *gin.Context) string {
	subprotocols := websocket.Subprotocols(c.Request)
	// Preferred format from browser:
	// new WebSocket(url, ["bearer", "<jwt>"])
	if len(subprotocols) >= 2 && strings.EqualFold(strings.TrimSpace(subprotocols[0]), "bearer") {
		return strings.TrimSpace(subprotocols[1])
	}
	return ""
}
