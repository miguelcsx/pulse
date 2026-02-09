package service

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/model"
)

// TagService handles user-defined hashtag creation and lookup.
type TagService struct {
	db *gorm.DB
}

func NewTagService(db *gorm.DB) *TagService {
	return &TagService{db: db}
}

// List returns tags ordered by usage count (most popular first).
func (s *TagService) List() ([]model.Tag, error) {
	var tags []model.Tag
	if err := s.db.Order("usage_count DESC, name ASC").Limit(200).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}
	return tags, nil
}

// Search finds tags by name prefix (case-insensitive autocomplete).
func (s *TagService) Search(query string) ([]model.Tag, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, nil
	}
	var tags []model.Tag
	if err := s.db.Where("name ILIKE ?", query+"%").
		Order("usage_count DESC").
		Limit(20).
		Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("failed to search tags: %w", err)
	}
	return tags, nil
}

// FindOrCreateByNames takes a list of tag name strings (as typed by users),
// normalizes them, and returns the corresponding Tag records. Creates new
// tags on first use. Increments usage_count for all resolved tags.
func (s *TagService) FindOrCreateByNames(names []string) ([]model.Tag, error) {
	if len(names) == 0 {
		return nil, nil
	}

	// Normalize: lowercase, trim, strip leading #, deduplicate
	seen := make(map[string]bool)
	var normalized []string
	for _, raw := range names {
		name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "#")))
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		normalized = append(normalized, name)
	}

	if len(normalized) == 0 {
		return nil, nil
	}

	var tags []model.Tag
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Find existing tags
		var existing []model.Tag
		if err := tx.Where("name IN ?", normalized).Find(&existing).Error; err != nil {
			return fmt.Errorf("failed to find existing tags: %w", err)
		}

		existingMap := make(map[string]model.Tag)
		for _, t := range existing {
			existingMap[t.Name] = t
		}

		// Create missing tags
		for _, name := range normalized {
			if t, ok := existingMap[name]; ok {
				tags = append(tags, t)
			} else {
				newTag := model.Tag{Name: name}
				if err := tx.Create(&newTag).Error; err != nil {
					return fmt.Errorf("failed to create tag %q: %w", name, err)
				}
				tags = append(tags, newTag)
			}
		}

		// Increment usage counts
		ids := make([]uuid.UUID, len(tags))
		for i, t := range tags {
			ids[i] = t.ID
		}
		if err := tx.Model(&model.Tag{}).Where("id IN ?", ids).
			UpdateColumn("usage_count", gorm.Expr("usage_count + 1")).Error; err != nil {
			return fmt.Errorf("failed to update tag usage: %w", err)
		}

		return nil
	})

	return tags, err
}

// Trending returns the most-used tags.
func (s *TagService) Trending(limit int) ([]model.Tag, error) {
	if limit <= 0 {
		limit = 20
	}
	var tags []model.Tag
	if err := s.db.Order("usage_count DESC").Limit(limit).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("failed to get trending tags: %w", err)
	}
	return tags, nil
}
