package api

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/pulse/stone/internal/middleware"
	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/service"
)

func (s *Server) GetDiscover(c *gin.Context) {
	userID := middleware.GetUserID(c)

	var (
		bucketed *service.BucketedSuggestions
		rooms    []model.Room
		paths    []model.Path

		suggestionsErr error
		roomsErr       error
		pathsErr       error

		wg sync.WaitGroup
	)

	wg.Add(3)

	go func() {
		defer wg.Done()
		bucketed, suggestionsErr = s.affinityService.GetSuggestionsBucketed(userID, 5)
	}()

	go func() {
		defer wg.Done()
		rooms, roomsErr = s.roomService.ListActive()
	}()

	go func() {
		defer wg.Done()
		paths, _, _, pathsErr = s.pathService.List(5, "")
	}()

	wg.Wait()

	if suggestionsErr != nil || bucketed == nil {
		bucketed = &service.BucketedSuggestions{}
	}
	if roomsErr != nil {
		rooms = []model.Room{}
	}
	if pathsErr != nil {
		paths = []model.Path{}
	}

	// Flatten for backward compatibility
	flatSuggestions := make([]service.SuggestionResult, 0)
	for _, bucket := range [][]service.SuggestionResult{
		bucketed.PathAffinity,
		bucketed.ClosestTwins,
		bucketed.AdjacentTaste,
		bucketed.Serendipity,
	} {
		flatSuggestions = append(flatSuggestions, bucket...)
	}

	c.JSON(http.StatusOK, gin.H{
		"suggestions":    flatSuggestions,
		"closest_twins":  bucketed.ClosestTwins,
		"adjacent_taste": bucketed.AdjacentTaste,
		"serendipity":    bucketed.Serendipity,
		"rooms":          rooms,
		"paths":          paths,
	})
}
