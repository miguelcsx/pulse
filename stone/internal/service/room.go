package service

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/pulse/stone/internal/config"
	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/store"
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

// GetRoomContent returns a mixed set of content: affinity-first with
// exploration injected to avoid a closed loop.
func (s *RoomService) GetRoomContent(ctx context.Context, graph *store.GraphStore, embedder Embedder, roomID uuid.UUID, userID uuid.UUID, limit int) ([]model.Content, error) {
	if graph == nil {
		return nil, errors.New("graph store not configured")
	}
	if embedder == nil {
		return nil, errors.New("embedder not configured")
	}
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

	tagIDs := make([]string, 0, len(room.Tags))
	for _, t := range room.Tags {
		tagIDs = append(tagIDs, t.ID.String())
	}

	userEmbedding, err := s.userEmbedding(ctx, graph, userID, embedder.Dimensions())
	if err != nil {
		return nil, err
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
	affinityCount := limit - exploreCount

	affinityItems, err := s.queryRoomAffinity(ctx, graph, tagIDs, userEmbedding, affinityCount)
	if err != nil {
		return nil, err
	}
	exploreItems, err := s.queryRoomExploration(ctx, graph, tagIDs, exploreCount)
	if err != nil {
		return nil, err
	}

	orderedIDs := make([]uuid.UUID, 0, len(affinityItems)+len(exploreItems))
	seen := make(map[uuid.UUID]struct{}, len(affinityItems)+len(exploreItems))
	for _, item := range append(affinityItems, exploreItems...) {
		if _, ok := seen[item.ID]; ok {
			continue
		}
		seen[item.ID] = struct{}{}
		orderedIDs = append(orderedIDs, item.ID)
	}

	return s.hydrateRoomContents(orderedIDs)
}

func (s *RoomService) userEmbedding(ctx context.Context, graph *store.GraphStore, userID uuid.UUID, dim int) ([]float64, error) {
	query := `
		MATCH (u:User {id: $userID})-[:CREATED]->(c:Content)
		WHERE c.embedding IS NOT NULL
		RETURN c.embedding AS embedding
		ORDER BY c.createdAt DESC
		LIMIT 20
	`

	raw, err := graph.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, map[string]any{
			"userID": userID.String(),
		})
		if err != nil {
			return nil, err
		}

		embeddings := make([][]float64, 0)
		for result.Next(ctx) {
			value, _ := result.Record().Get("embedding")
			vec := toFloatSlice(value, dim)
			if len(vec) > 0 {
				embeddings = append(embeddings, vec)
			}
		}
		return embeddings, result.Err()
	})
	if err != nil {
		return nil, err
	}

	items := raw.([][]float64)
	if len(items) == 0 {
		return make([]float64, dim), nil
	}

	return averageVectors(items, dim), nil
}

func (s *RoomService) queryRoomAffinity(ctx context.Context, graph *store.GraphStore, tagIDs []string, embedding []float64, limit int) ([]model.Content, error) {
	query := `
		CALL db.index.vector.queryNodes('content_embedding', $k, $embedding)
		YIELD node AS c, score
		MATCH (c)-[:TAGGED]->(t:Tag)
		WHERE t.id IN $tagIDs
		RETURN c { .id, .contentType, .mediaUrl, .body, .createdAt, .updatedAt } AS content
		ORDER BY score DESC
		LIMIT $limit
	`

	raw, err := graph.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, map[string]any{
			"k":         limit,
			"embedding": embedding,
			"tagIDs":    tagIDs,
			"limit":     limit,
		})
		if err != nil {
			return nil, err
		}

		contents := make([]model.Content, 0)
		for result.Next(ctx) {
			node, _ := result.Record().Get("content")
			content, err := mapRoomContent(node)
			if err != nil {
				return nil, err
			}
			contents = append(contents, content)
		}
		return contents, result.Err()
	})
	if err != nil {
		return nil, err
	}

	return raw.([]model.Content), nil
}

func (s *RoomService) queryRoomExploration(ctx context.Context, graph *store.GraphStore, tagIDs []string, limit int) ([]model.Content, error) {
	query := `
		MATCH (c:Content)-[:TAGGED]->(t:Tag)
		WHERE NOT t.id IN $tagIDs
		WITH c ORDER BY rand()
		LIMIT $limit
		RETURN c { .id, .contentType, .mediaUrl, .body, .createdAt, .updatedAt } AS content
	`

	raw, err := graph.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, map[string]any{
			"tagIDs": tagIDs,
			"limit":  limit,
		})
		if err != nil {
			return nil, err
		}

		contents := make([]model.Content, 0)
		for result.Next(ctx) {
			node, _ := result.Record().Get("content")
			content, err := mapRoomContent(node)
			if err != nil {
				return nil, err
			}
			contents = append(contents, content)
		}
		return contents, result.Err()
	})
	if err != nil {
		return nil, err
	}

	return raw.([]model.Content), nil
}

func mapRoomContent(node any) (model.Content, error) {
	props, ok := node.(map[string]any)
	if !ok {
		return model.Content{}, fmt.Errorf("invalid content payload")
	}

	idStr, _ := props["id"].(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return model.Content{}, fmt.Errorf("invalid content id: %w", err)
	}

	return model.Content{
		ID:          id,
		ContentType: toString(props["contentType"]),
		MediaURL:    toString(props["mediaUrl"]),
		Body:        toString(props["body"]),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}, nil
}

func (s *RoomService) hydrateRoomContents(ids []uuid.UUID) ([]model.Content, error) {
	if len(ids) == 0 {
		return []model.Content{}, nil
	}

	var contents []model.Content
	if err := s.db.Preload("Creator").Preload("Tags").Where("id IN ?", ids).Find(&contents).Error; err != nil {
		return nil, fmt.Errorf("failed to load room contents: %w", err)
	}

	contentByID := make(map[uuid.UUID]model.Content, len(contents))
	for _, c := range contents {
		contentByID[c.ID] = c
	}

	ordered := make([]model.Content, 0, len(ids))
	for _, id := range ids {
		if c, ok := contentByID[id]; ok {
			ordered = append(ordered, c)
		}
	}

	return ordered, nil
}

func toFloatSlice(value any, dim int) []float64 {
	switch v := value.(type) {
	case []float64:
		return v
	case []any:
		out := make([]float64, 0, len(v))
		for _, item := range v {
			switch n := item.(type) {
			case float64:
				out = append(out, n)
			case int:
				out = append(out, float64(n))
			case int64:
				out = append(out, float64(n))
			}
		}
		return out
	default:
		if dim > 0 {
			return make([]float64, dim)
		}
		return nil
	}
}

func averageVectors(items [][]float64, dim int) []float64 {
	if dim <= 0 {
		return nil
	}
	acc := make([]float64, dim)
	for _, vec := range items {
		for i := 0; i < dim && i < len(vec); i++ {
			acc[i] += vec[i]
		}
	}
	for i := range acc {
		acc[i] = acc[i] / float64(len(items))
	}
	return acc
}
