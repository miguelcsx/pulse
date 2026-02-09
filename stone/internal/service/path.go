package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/model"
)

// PathItemInput represents input for creating a path item.
type PathItemInput struct {
	ContentID uuid.UUID `json:"content_id"`
	Note      string    `json:"note"`
}

// PathService handles curated content paths.
type PathService struct {
	db *gorm.DB
}

// NewPathService creates a new PathService.
func NewPathService(db *gorm.DB) *PathService {
	return &PathService{db: db}
}

// Create creates a new path with sequential path items.
func (s *PathService) Create(creatorID uuid.UUID, title, description string, items []PathItemInput) (*model.Path, error) {
	path := &model.Path{
		CreatorID:   creatorID,
		Title:       title,
		Description: description,
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(path).Error; err != nil {
			return fmt.Errorf("failed to create path: %w", err)
		}

		for i, item := range items {
			pathItem := model.PathItem{
				PathID:    path.ID,
				ContentID: item.ContentID,
				Position:  i + 1,
				Note:      item.Note,
			}
			if err := tx.Create(&pathItem).Error; err != nil {
				return fmt.Errorf("failed to create path item at position %d: %w", i+1, err)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Reload with all associations.
	if err := s.db.Preload("Creator").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Items.Content").
		Preload("Items.Content.Tags").
		First(path, "id = ?", path.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload path: %w", err)
	}

	return path, nil
}

// GetByID returns a path by ID with Creator, Items, and Items.Content.Tags preloaded.
func (s *PathService) GetByID(id uuid.UUID) (*model.Path, error) {
	var path model.Path
	err := s.db.Preload("Creator").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Items.Content").
		Preload("Items.Content.Tags").
		First(&path, "id = ?", id).Error
	if err != nil {
		return nil, fmt.Errorf("failed to find path: %w", err)
	}
	return &path, nil
}

// List returns paths using cursor-based pagination by ID.
// Returns the paths, a next cursor, and whether there are more results.
func (s *PathService) List(limit int, cursor string) ([]model.Path, string, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	query := s.db.Preload("Creator").
		Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("position ASC")
		}).
		Preload("Items.Content").
		Preload("Items.Content.Tags").
		Order("created_at DESC").
		Limit(limit + 1)

	if cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339Nano, cursor)
		if err != nil {
			return nil, "", false, fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("created_at < ?", cursorTime)
	}

	var paths []model.Path
	if err := query.Find(&paths).Error; err != nil {
		return nil, "", false, fmt.Errorf("failed to list paths: %w", err)
	}

	hasMore := len(paths) > limit
	var nextCursor string
	if hasMore {
		paths = paths[:limit]
		nextCursor = paths[len(paths)-1].CreatedAt.Format(time.RFC3339Nano)
	}

	return paths, nextCursor, hasMore, nil
}

// Follow creates a path follow and increments the follower count.
func (s *PathService) Follow(pathID, userID uuid.UUID) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		follow := model.PathFollow{
			PathID:    pathID,
			UserID:    userID,
			CreatedAt: time.Now(),
		}
		if err := tx.Create(&follow).Error; err != nil {
			return fmt.Errorf("failed to follow path: %w", err)
		}

		if err := tx.Model(&model.Path{}).
			Where("id = ?", pathID).
			UpdateColumn("follower_count", gorm.Expr("follower_count + 1")).Error; err != nil {
			return fmt.Errorf("failed to increment follower count: %w", err)
		}

		return nil
	})
}

// IsFollowing checks whether a user is following a specific path.
func (s *PathService) IsFollowing(pathID, userID uuid.UUID) (bool, error) {
	var count int64
	err := s.db.Model(&model.PathFollow{}).
		Where("path_id = ? AND user_id = ?", pathID, userID).
		Count(&count).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check path follow: %w", err)
	}
	return count > 0, nil
}
