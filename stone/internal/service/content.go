package service

import (
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/store"
)

// ContentService handles content upload and retrieval for all content types.
type ContentService struct {
	db      *gorm.DB
	storage store.Storage
	tags    *TagService
	media   *MediaService
}

func NewContentService(db *gorm.DB, storage store.Storage, tags *TagService, media *MediaService) *ContentService {
	return &ContentService{db: db, storage: storage, tags: tags, media: media}
}

// Create saves content. For image/video types, saves the file via storage.
// For text posts, no file is needed. Tags are user-defined strings (hashtags).
func (s *ContentService) Create(creatorID uuid.UUID, contentType string, filename string, file io.Reader, body string, tagNames []string) (*model.Content, error) {
	// Validate content type
	switch contentType {
	case model.ContentTypeImage, model.ContentTypeVideo, model.ContentTypeShortVideo, model.ContentTypeText:
	default:
		return nil, fmt.Errorf("invalid content type: %s", contentType)
	}

	var mediaURL string
	var storagePath string
	if contentType != model.ContentTypeText && file != nil {
		var err error
		storagePath, err = s.storage.Save(filename, file)
		if err != nil {
			return nil, fmt.Errorf("failed to save file: %w", err)
		}
		mediaURL = s.storage.URL(storagePath)
	}

	// Resolve tags (find-or-create user-defined hashtags)
	tags, err := s.tags.FindOrCreateByNames(tagNames)
	if err != nil {
		if storagePath != "" {
			_ = s.storage.Delete(storagePath)
		}
		return nil, fmt.Errorf("failed to resolve tags: %w", err)
	}

	content := &model.Content{CreatorID: creatorID, ContentType: contentType, MediaURL: mediaURL, Body: body}

	return s.createContentRecord(content, tags, storagePath)
}

// CreateFromMediaAsset creates content from a pre-uploaded, ready media asset.
func (s *ContentService) CreateFromMediaAsset(creatorID uuid.UUID, contentType string, mediaAssetID uuid.UUID, body string, tagNames []string) (*model.Content, error) {
	if s.media == nil {
		return nil, errors.New("media service unavailable")
	}
	if contentType == model.ContentTypeText {
		return nil, errors.New("text content cannot reference a media asset")
	}

	asset, err := s.media.GetReadyForOwner(mediaAssetID, creatorID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve media asset: %w", err)
	}

	tags, err := s.tags.FindOrCreateByNames(tagNames)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve tags: %w", err)
	}

	content := &model.Content{
		CreatorID:    creatorID,
		ContentType:  contentType,
		MediaAssetID: &asset.ID,
		MediaURL:     asset.PlaybackURL,
		Body:         body,
	}

	return s.createContentRecord(content, tags, "")
}

func (s *ContentService) createContentRecord(content *model.Content, tags []model.Tag, cleanupStoragePath string) (*model.Content, error) {
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(content).Error; err != nil {
			return fmt.Errorf("failed to create content: %w", err)
		}

		if len(tags) > 0 {
			if err := tx.Model(content).Association("Tags").Append(&tags); err != nil {
				return fmt.Errorf("failed to associate tags: %w", err)
			}
		}

		return nil
	})
	if err != nil {
		if cleanupStoragePath != "" {
			_ = s.storage.Delete(cleanupStoragePath)
		}
		return nil, err
	}

	// Reload with associations
	if err := s.db.Preload("Creator").Preload("Tags").First(content, "id = ?", content.ID).Error; err != nil {
		return nil, fmt.Errorf("failed to reload content: %w", err)
	}

	return content, nil
}

// GetByID returns a content record by ID with Creator and Tags preloaded.
func (s *ContentService) GetByID(id uuid.UUID) (*model.Content, error) {
	var content model.Content
	if err := s.db.Preload("Creator").Preload("Tags").First(&content, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to find content: %w", err)
	}
	return &content, nil
}

// Delete removes a content record and its file from storage. Only the owner can delete.
func (s *ContentService) Delete(id uuid.UUID, userID uuid.UUID) error {
	var content model.Content
	if err := s.db.First(&content, "id = ?", id).Error; err != nil {
		return fmt.Errorf("failed to find content: %w", err)
	}

	if content.CreatorID != userID {
		return errors.New("unauthorized: only the owner can delete this content")
	}

	mediaURL := content.MediaURL

	if err := s.db.Delete(&content).Error; err != nil {
		return fmt.Errorf("failed to delete content: %w", err)
	}

	if mediaURL != "" {
		_ = s.storage.Delete(mediaURL)
	}

	return nil
}

// ListByUser returns content for a given user using cursor-based pagination.
func (s *ContentService) ListByUser(userID uuid.UUID, limit int, cursor string) ([]model.Content, string, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	query := s.db.Preload("Creator").Preload("Tags").
		Where("creator_id = ?", userID).
		Order("created_at DESC").
		Limit(limit + 1)

	if cursor != "" {
		cursorTime, err := time.Parse(time.RFC3339Nano, cursor)
		if err != nil {
			return nil, "", fmt.Errorf("invalid cursor: %w", err)
		}
		query = query.Where("created_at < ?", cursorTime)
	}

	var contents []model.Content
	if err := query.Find(&contents).Error; err != nil {
		return nil, "", fmt.Errorf("failed to list content: %w", err)
	}

	var nextCursor string
	if len(contents) > limit {
		nextCursor = contents[limit-1].CreatedAt.Format(time.RFC3339Nano)
		contents = contents[:limit]
	}

	return contents, nextCursor, nil
}

// React adds a semantic reaction to content.
func (s *ContentService) React(userID, contentID uuid.UUID, kind string) error {
	reaction := model.Reaction{
		UserID:    userID,
		ContentID: contentID,
		Kind:      kind,
	}
	if err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}, {Name: "content_id"}, {Name: "kind"}},
		DoNothing: true,
	}).Create(&reaction).Error; err != nil {
		return fmt.Errorf("failed to add reaction: %w", err)
	}
	return nil
}

// RemoveReaction removes a semantic reaction.
func (s *ContentService) RemoveReaction(userID, contentID uuid.UUID, kind string) error {
	result := s.db.Where("user_id = ? AND content_id = ? AND kind = ?", userID, contentID, kind).
		Delete(&model.Reaction{})
	if result.Error != nil {
		return fmt.Errorf("failed to remove reaction: %w", result.Error)
	}
	return nil
}

// GetReactions returns the reaction counts for a piece of content.
func (s *ContentService) GetReactions(contentID uuid.UUID) (map[string]int, error) {
	type result struct {
		Kind  string
		Count int
	}
	var results []result
	if err := s.db.Model(&model.Reaction{}).
		Select("kind, count(*) as count").
		Where("content_id = ?", contentID).
		Group("kind").
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get reactions: %w", err)
	}

	counts := make(map[string]int)
	for _, r := range results {
		counts[r.Kind] = r.Count
	}
	return counts, nil
}
