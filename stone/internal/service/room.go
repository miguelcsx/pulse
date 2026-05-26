package service

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pulse/stone/internal/config"
	"github.com/pulse/stone/internal/model"
)

// RoomService handles mood room lifecycle.
type RoomService struct {
	db  *gorm.DB
	cfg *config.Config
}

var ErrRoomNotFoundOrExpired = errors.New("room not found or expired")

// NewRoomService creates a new RoomService.
func NewRoomService(db *gorm.DB, cfg *config.Config) *RoomService {
	return &RoomService{db: db, cfg: cfg}
}

// roomTTL returns the configured room time-to-live duration.
func (s *RoomService) roomTTL() time.Duration {
	if s.cfg != nil && s.cfg.RoomTTL > 0 {
		return s.cfg.RoomTTL
	}
	return 24 * time.Hour
}

// FindOrCreateByTags finds an existing room matching the given tag set and date bucket,
// or creates a new one with the configured TTL.
func (s *RoomService) FindOrCreateByTags(tagIDs []uuid.UUID) (*model.Room, error) {
	if len(tagIDs) == 0 {
		return nil, fmt.Errorf("at least one tag is required")
	}

	clusterKey := BuildClusterKey(tagIDs)

	var room model.Room
	err := s.db.Where("cluster_key = ? AND expires_at > ?", clusterKey, time.Now()).
		Preload("Tags").
		First(&room).Error

	if err == nil {
		return &room, nil
	}

	if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to find room: %w", err)
	}

	// Create a new room with configurable TTL.
	room = model.Room{
		ClusterKey: clusterKey,
		ExpiresAt:  time.Now().Add(s.roomTTL()),
	}

	err = s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&room).Error; err != nil {
			return fmt.Errorf("failed to create room: %w", err)
		}

		var tags []model.Tag
		if err := tx.Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
			return fmt.Errorf("failed to find tags: %w", err)
		}
		if err := tx.Model(&room).Association("Tags").Append(&tags); err != nil {
			return fmt.Errorf("failed to associate tags: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Reload with tags.
	if err := s.db.Preload("Tags").First(&room, "id = ?", room.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload room: %w", err)
	}

	return &room, nil
}

// ListActive returns all non-expired rooms, ordered by member count descending.
func (s *RoomService) ListActive() ([]model.Room, error) {
	var rooms []model.Room
	err := s.db.Preload("Tags").
		Where("expires_at > ?", time.Now()).
		Select("rooms.*, (SELECT COUNT(*) FROM room_members rm WHERE rm.room_id = rooms.id) as member_count").
		Order("member_count DESC").
		Find(&rooms).Error
	if err != nil {
		return nil, fmt.Errorf("failed to list active rooms: %w", err)
	}
	return rooms, nil
}

// Enter adds a user to a room.
func (s *RoomService) Enter(roomID, userID uuid.UUID) error {
	var room model.Room
	if err := s.db.Where("id = ? AND expires_at > ?", roomID, time.Now()).First(&room).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRoomNotFoundOrExpired
		}
		return fmt.Errorf("failed to find room: %w", err)
	}

	member := model.RoomMember{
		RoomID:   roomID,
		UserID:   userID,
		JoinedAt: time.Now(),
	}
	if err := s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&member).Error; err != nil {
		return fmt.Errorf("failed to enter room: %w", err)
	}
	return nil
}

// Leave removes a user from a room.
func (s *RoomService) Leave(roomID, userID uuid.UUID) error {
	result := s.db.Where("room_id = ? AND user_id = ?", roomID, userID).
		Delete(&model.RoomMember{})
	if result.Error != nil {
		return fmt.Errorf("failed to leave room: %w", result.Error)
	}
	return nil
}

// IsActiveMember reports whether a user is currently a member of a non-expired room.
func (s *RoomService) IsActiveMember(roomID string, userID uuid.UUID) (bool, error) {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return false, nil
	}
	parsedRoomID, err := uuid.Parse(roomID)
	if err != nil {
		return false, nil
	}

	var count int64
	err = s.db.Model(&model.RoomMember{}).
		Joins("JOIN rooms ON rooms.id = room_members.room_id").
		Where("room_members.room_id = ? AND room_members.user_id = ? AND rooms.expires_at > ?", parsedRoomID, userID, time.Now()).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to verify room membership: %w", err)
	}
	return count > 0, nil
}

// GetMemberCount returns the number of members in a room.
func (s *RoomService) GetMemberCount(roomID uuid.UUID) (int64, error) {
	var count int64
	err := s.db.Model(&model.RoomMember{}).
		Where("room_id = ?", roomID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to get member count: %w", err)
	}
	return count, nil
}

// CleanupExpired deletes rooms that have passed their expiration time.
func (s *RoomService) CleanupExpired() error {
	if err := s.db.Where("expires_at <= ?", time.Now()).Delete(&model.Room{}).Error; err != nil {
		return fmt.Errorf("failed to cleanup expired rooms: %w", err)
	}
	return nil
}

// BuildClusterKey creates a deterministic key from sorted tag IDs and the current date bucket.
func BuildClusterKey(tagIDs []uuid.UUID) string {
	// Sort tag IDs for deterministic ordering.
	sorted := make([]string, len(tagIDs))
	for i, id := range tagIDs {
		sorted[i] = id.String()
	}
	sort.Strings(sorted)

	dateBucket := time.Now().Format("2006-01-02")

	h := sha256.New()
	for _, id := range sorted {
		h.Write([]byte(id))
	}
	h.Write([]byte(dateBucket))

	return fmt.Sprintf("%x", h.Sum(nil))
}

// GetRoomContent returns recent content matching the room context, with a small
// exploration segment so sessions do not become repetitive.
func (s *RoomService) GetRoomContent(roomID uuid.UUID, userID uuid.UUID, limit int) ([]model.Content, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	var room model.Room
	if err := s.db.Preload("Tags").
		Where("id = ? AND expires_at > ?", roomID, time.Now()).
		First(&room).Error; err != nil {
		return nil, ErrRoomNotFoundOrExpired
	}

	if len(room.Tags) == 0 {
		return []model.Content{}, nil
	}

	tagIDs := make([]uuid.UUID, 0, len(room.Tags))
	for _, t := range room.Tags {
		tagIDs = append(tagIDs, t.ID)
	}

	explorationRatio := 0.2
	if s.cfg != nil && s.cfg.RoomExplorationRatio > 0 {
		explorationRatio = s.cfg.RoomExplorationRatio
	}
	exploreCount := int(math.Ceil(float64(limit) * explorationRatio))
	if exploreCount < 1 {
		exploreCount = 1
	}
	if exploreCount >= limit {
		exploreCount = limit - 1
	}
	mainCount := limit - exploreCount

	var primary []model.Content
	if err := s.db.Preload("Creator").Preload("Tags").
		Joins("JOIN content_tags ct ON ct.content_id = contents.id").
		Where("ct.tag_id IN ?", tagIDs).
		Where("contents.creator_id <> ?", userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocker_id = ? AND blocked_id = contents.creator_id
		)`, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocked_id = ? AND blocker_id = contents.creator_id
		)`, userID).
		Group("contents.id").
		Order("MAX(contents.created_at) DESC").
		Limit(mainCount).
		Find(&primary).Error; err != nil {
		return nil, fmt.Errorf("failed to load room content: %w", err)
	}

	var excludeIDs []uuid.UUID
	for _, item := range primary {
		excludeIDs = append(excludeIDs, item.ID)
	}

	var exploration []model.Content
	exploreQuery := s.db.Preload("Creator").Preload("Tags").
		Where("creator_id <> ?", userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocker_id = ? AND blocked_id = contents.creator_id
		)`, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocked_id = ? AND blocker_id = contents.creator_id
		)`, userID).
		Order("created_at DESC").
		Limit(exploreCount)
	if len(excludeIDs) > 0 {
		exploreQuery = exploreQuery.Where("id NOT IN ?", excludeIDs)
	}
	if err := exploreQuery.Find(&exploration).Error; err != nil {
		return nil, fmt.Errorf("failed to load room exploration content: %w", err)
	}

	return append(primary, exploration...), nil
}
