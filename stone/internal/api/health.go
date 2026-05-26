package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (s *Server) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	status := "ok"
	checks := gin.H{}

	// Check Postgres
	sqlDB, err := s.db.DB()
	if err != nil {
		status = "degraded"
		checks["postgres"] = "error: " + err.Error()
	} else if err := sqlDB.PingContext(ctx); err != nil {
		status = "degraded"
		checks["postgres"] = "error: " + err.Error()
	} else {
		checks["postgres"] = "ok"
	}

	// Check Neo4j
	if s.graph == nil {
		status = "degraded"
		checks["neo4j"] = "error: graph store not configured"
	} else if err := s.graph.Driver.VerifyConnectivity(ctx); err != nil {
		status = "degraded"
		checks["neo4j"] = "error: " + err.Error()
	} else {
		checks["neo4j"] = "ok"
	}

	// Check Redis
	if err := s.redis.Ping(ctx).Err(); err != nil {
		status = "degraded"
		checks["redis"] = "error: " + err.Error()
	} else {
		checks["redis"] = "ok"
	}

	code := http.StatusOK
	if status != "ok" {
		code = http.StatusServiceUnavailable
	}

	c.JSON(code, gin.H{
		"status": status,
		"checks": checks,
	})
}
