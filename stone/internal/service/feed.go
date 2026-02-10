package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/model"
)

// FeedService provides the main content feed.
type FeedService struct {
	db *gorm.DB
}

// NewFeedService creates a new FeedService.
func NewFeedService(db *gorm.DB) *FeedService {
	return &FeedService{db: db}
}

// GetFeed returns content from followed users and the user's own content,
// using cursor-based pagination by (created_at,id).
// Default limit is 20, max is 50.
// Block filtering uses NOT EXISTS for optimal query performance.
func (s *FeedService) GetFeed(userID uuid.UUID, cursor string, limit int) ([]model.Content, string, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	query := s.db.Preload("Creator").Preload("Tags").
		Where(`(creator_id = ? OR creator_id IN (
			SELECT followee_id FROM follows WHERE follower_id = ?
		))`, userID, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocker_id = ? AND blocked_id = contents.creator_id
		)`, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocked_id = ? AND blocker_id = contents.creator_id
		)`, userID).
		Order("created_at DESC, id DESC").
		Limit(limit + 1)

	if cursor != "" {
		cursorTime, cursorID, err := decodeFeedCursor(cursor)
		if err != nil {
			return nil, "", false, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("(created_at, id) < (?, ?)", cursorTime, cursorID)
	}

	var contents []model.Content
	if err := query.Find(&contents).Error; err != nil {
		return nil, "", false, fmt.Errorf("failed to get feed: %w", err)
	}

	if err := s.hydrateReactionCounts(contents); err != nil {
		return nil, "", false, err
	}

	hasMore := len(contents) > limit
	var nextCursor string
	if hasMore {
		contents = contents[:limit]
		nextCursor = encodeFeedCursor(contents[len(contents)-1].CreatedAt, contents[len(contents)-1].ID)
	}

	return contents, nextCursor, hasMore, nil
}

func encodeFeedCursor(createdAt time.Time, id uuid.UUID) string {
	return createdAt.UTC().Format(time.RFC3339Nano) + "|" + id.String()
}

func decodeFeedCursor(cursor string) (time.Time, uuid.UUID, error) {
	parts := strings.Split(strings.TrimSpace(cursor), "|")
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, fmt.Errorf("malformed cursor")
	}
	cursorTime, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	cursorID, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, err
	}
	return cursorTime, cursorID, nil
}

func (s *FeedService) hydrateReactionCounts(contents []model.Content) error {
	if len(contents) == 0 {
		return nil
	}

	contentIDs := make([]uuid.UUID, len(contents))
	for i, c := range contents {
		contentIDs[i] = c.ID
	}

	type row struct {
		ContentID uuid.UUID `gorm:"column:content_id"`
		Kind      string    `gorm:"column:kind"`
		Count     int       `gorm:"column:count"`
	}

	var rows []row
	if err := s.db.Model(&model.Reaction{}).
		Select("content_id, kind, COUNT(*) as count").
		Where("content_id IN ?", contentIDs).
		Group("content_id, kind").
		Scan(&rows).Error; err != nil {
		return fmt.Errorf("failed to load reaction counts: %w", err)
	}

	reactionMap := make(map[uuid.UUID]map[string]int)
	for _, r := range rows {
		if reactionMap[r.ContentID] == nil {
			reactionMap[r.ContentID] = make(map[string]int)
		}
		reactionMap[r.ContentID][r.Kind] = r.Count
	}

	for i := range contents {
		contents[i].Reactions = reactionMap[contents[i].ID]
	}

	return nil
}
