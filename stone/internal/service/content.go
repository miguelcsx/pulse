package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/store"
)

// ContentService handles content upload and retrieval for all content types.
type ContentService struct {
	db       *gorm.DB
	graph    *store.GraphStore
	storage  store.Storage
	tags     *TagService
	media    *MediaService
	rooms    *RoomService
	embedder Embedder
}

func NewContentService(db *gorm.DB, graph *store.GraphStore, storage store.Storage, tags *TagService, media *MediaService, rooms *RoomService, embedder Embedder) *ContentService {
	return &ContentService{
		db:       db,
		graph:    graph,
		storage:  storage,
		tags:     tags,
		media:    media,
		rooms:    rooms,
		embedder: embedder,
	}
}

// Create saves content. For image/video types, saves the file via storage.
// For text posts, no file is needed. Tags are user-defined strings (hashtags).
func (s *ContentService) Create(creatorID uuid.UUID, contentType string, filename string, file io.Reader, body string, tagNames []string) (*model.Content, error) {
	// Validate content type
	if !model.IsValidContentType(contentType) {
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

	// Auto-create a mood room for this tag combination (fire-and-forget).
	if s.rooms != nil && len(content.Tags) > 0 {
		tagIDs := make([]uuid.UUID, len(content.Tags))
		for i, t := range content.Tags {
			tagIDs[i] = t.ID
		}
		if _, err := s.rooms.FindOrCreateByTags(tagIDs); err != nil {
			slog.Error("failed to auto-create room for content", "content_id", content.ID, "error", err)
		}
	}

	if err := s.syncContentToGraph(content); err != nil {
		slog.Error("failed to sync content to graph", "content_id", content.ID, "error", err)
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

	err := s.db.Transaction(func(tx *gorm.DB) error {
		// Remove reactions referencing this content
		if err := tx.Where("content_id = ?", id).Delete(&model.Reaction{}).Error; err != nil {
			return fmt.Errorf("failed to delete reactions: %w", err)
		}
		// Remove content-tag associations
		if err := tx.Where("content_id = ?", id).Delete(&model.ContentTag{}).Error; err != nil {
			return fmt.Errorf("failed to delete content tags: %w", err)
		}
		// Remove path items referencing this content
		if err := tx.Where("content_id = ?", id).Delete(&model.PathItem{}).Error; err != nil {
			return fmt.Errorf("failed to delete path items: %w", err)
		}
		// Delete the content itself
		if err := tx.Delete(&content).Error; err != nil {
			return fmt.Errorf("failed to delete content: %w", err)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if mediaURL != "" {
		_ = s.storage.Delete(mediaURL)
	}

	if err := s.deleteContentFromGraph(content.ID); err != nil {
		slog.Error("failed to delete content from graph", "content_id", content.ID, "error", err)
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

	if err := s.syncReactionToGraph(userID, contentID, kind); err != nil {
		slog.Error("failed to sync reaction to graph", "content_id", contentID, "user_id", userID, "error", err)
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

	if err := s.removeReactionFromGraph(userID, contentID, kind); err != nil {
		slog.Error("failed to remove reaction from graph", "content_id", contentID, "user_id", userID, "error", err)
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

func (s *ContentService) syncContentToGraph(content *model.Content) error {
	if s.graph == nil {
		return nil
	}

	ctx := context.Background()

	var embedding any // nil when no model available — omits the property from Neo4j
	if s.embedder != nil {
		vec, err := s.embedder.EmbedText(buildContentEmbeddingText(content))
		if err != nil {
			slog.Warn("embedding unavailable, content stored without vector", "content_id", content.ID, "error", err)
		} else {
			embedding = vec
		}
	}

	tags := make([]map[string]any, 0, len(content.Tags))
	for _, t := range content.Tags {
		tags = append(tags, map[string]any{
			"id":         t.ID.String(),
			"name":       t.Name,
			"usageCount": t.UsageCount,
		})
	}

	user := content.Creator
	// FOREACH avoids UNWIND short-circuiting the whole query when tags is empty.
	// embedding is only SET when non-null so we don't violate the vector index dimensions.
	query := `
		MERGE (u:User {id: $user.id})
		SET u.handle = $user.handle,
		    u.displayName = $user.displayName,
		    u.avatarUrl = $user.avatarUrl,
		    u.bio = $user.bio,
		    u.updatedAt = $now,
		    u.createdAt = coalesce(u.createdAt, $now)
		MERGE (c:Content {id: $content.id})
		SET c.contentType = $content.contentType,
		    c.mediaUrl = $content.mediaUrl,
		    c.body = $content.body,
		    c.createdAt = $content.createdAt,
		    c.updatedAt = $content.updatedAt
		SET c.embedding = CASE WHEN $embedding IS NOT NULL THEN $embedding ELSE c.embedding END
		MERGE (u)-[:CREATED]->(c)
		WITH c
		FOREACH (tag IN $tags |
			MERGE (t:Tag {id: tag.id})
			SET t.name = tag.name,
			    t.usageCount = tag.usageCount
			MERGE (c)-[:TAGGED]->(t)
		)
	`

	params := map[string]any{
		"user": map[string]any{
			"id":          user.ID.String(),
			"handle":      user.Handle,
			"displayName": user.DisplayName,
			"avatarUrl":   user.AvatarURL,
			"bio":         user.Bio,
		},
		"content": map[string]any{
			"id":          content.ID.String(),
			"contentType": content.ContentType,
			"mediaUrl":    content.MediaURL,
			"body":        content.Body,
			"createdAt":   content.CreatedAt.UTC(),
			"updatedAt":   content.UpdatedAt.UTC(),
		},
		"tags":      tags,
		"embedding": embedding,
		"now":       time.Now().UTC(),
	}

	return s.graph.RunWrite(ctx, query, params)
}

func (s *ContentService) syncReactionToGraph(userID, contentID uuid.UUID, kind string) error {
	if s.graph == nil {
		return nil
	}

	ctx := context.Background()
	query := `
		MERGE (u:User {id: $userID})
		MERGE (c:Content {id: $contentID})
		MERGE (u)-[r:REACTED {kind: $kind}]->(c)
		SET r.createdAt = coalesce(r.createdAt, datetime())
	`
	params := map[string]any{
		"userID":    userID.String(),
		"contentID": contentID.String(),
		"kind":      kind,
	}

	return s.graph.RunWrite(ctx, query, params)
}

func (s *ContentService) removeReactionFromGraph(userID, contentID uuid.UUID, kind string) error {
	if s.graph == nil {
		return nil
	}

	ctx := context.Background()
	query := `
		MATCH (u:User {id: $userID})-[r:REACTED {kind: $kind}]->(c:Content {id: $contentID})
		DELETE r
	`
	params := map[string]any{
		"userID":    userID.String(),
		"contentID": contentID.String(),
		"kind":      kind,
	}

	_, err := s.graph.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})
	return err
}

func (s *ContentService) deleteContentFromGraph(contentID uuid.UUID) error {
	if s.graph == nil {
		return nil
	}

	ctx := context.Background()
	query := `
		MATCH (c:Content {id: $contentID})
		DETACH DELETE c
	`
	params := map[string]any{
		"contentID": contentID.String(),
	}

	return s.graph.RunWrite(ctx, query, params)
}

func buildContentEmbeddingText(content *model.Content) string {
	parts := make([]string, 0, 1+len(content.Tags))
	if strings.TrimSpace(content.Body) != "" {
		parts = append(parts, content.Body)
	}
	for _, t := range content.Tags {
		parts = append(parts, "#"+t.Name)
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}
