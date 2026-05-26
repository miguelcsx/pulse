package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/store"
)

const (
	tagCacheKeyList     = "tags:list"
	tagCacheKeyTrending = "tags:trending"
	tagCacheKeySearch   = "tags:search:"
)

// TagService handles user-defined hashtag creation and lookup.
type TagService struct {
	db       *gorm.DB
	graph    *store.GraphStore
	rdb      *redis.Client
	cacheTTL time.Duration
}

func NewTagService(db *gorm.DB, graph *store.GraphStore, rdb *redis.Client, cacheTTL time.Duration) *TagService {
	if cacheTTL <= 0 {
		cacheTTL = 5 * time.Minute
	}
	return &TagService{db: db, graph: graph, rdb: rdb, cacheTTL: cacheTTL}
}

// List returns tags ordered by usage count (most popular first).
// Results are cached in Redis for the configured TTL.
func (s *TagService) List() ([]model.Tag, error) {
	if cached, err := s.getTagsFromCache(tagCacheKeyList); err == nil && cached != nil {
		return cached, nil
	}

	var tags []model.Tag
	if err := s.db.Order("usage_count DESC, name ASC").Limit(200).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("failed to list tags: %w", err)
	}

	s.setTagsCache(tagCacheKeyList, tags)
	return tags, nil
}

// Search finds tags by name prefix (case-insensitive autocomplete).
// Short prefixes (≥2 chars) are cached briefly to reduce DB load from
// rapid autocomplete keystrokes.
func (s *TagService) Search(query string) ([]model.Tag, error) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil, nil
	}

	// Only cache searches with 2+ character prefixes to keep cardinality bounded.
	cacheKey := ""
	if len(query) >= 2 {
		cacheKey = tagCacheKeySearch + query
		if cached, err := s.getTagsFromCache(cacheKey); err == nil && cached != nil {
			return cached, nil
		}
	}

	var tags []model.Tag
	if err := s.db.Where("name ILIKE ?", query+"%").
		Order("usage_count DESC").
		Limit(20).
		Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("failed to search tags: %w", err)
	}

	if cacheKey != "" {
		// Use a shorter TTL for search results since they change more often.
		s.setTagsCacheWithTTL(cacheKey, tags, s.cacheTTL/2)
	}
	return tags, nil
}

// FindOrCreateByNames takes a list of tag name strings (as typed by users),
// normalizes them, and returns the corresponding Tag records. Creates new
// tags on first use via a single batch INSERT with ON CONFLICT DO NOTHING.
// Increments usage_count for all resolved tags.
// Invalidates the tag list and trending caches after mutation.
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
		// Batch-insert all tag names at once. Existing tags are left untouched
		// thanks to ON CONFLICT DO NOTHING.
		if err := tx.Exec(
			buildBatchInsertTagsSQL(normalized),
			buildBatchInsertTagsArgs(normalized)...,
		).Error; err != nil {
			return fmt.Errorf("failed to batch-insert tags: %w", err)
		}

		// Fetch all tags (both newly created and pre-existing) in a single query.
		if err := tx.Where("name IN ?", normalized).Find(&tags).Error; err != nil {
			return fmt.Errorf("failed to load tags: %w", err)
		}

		// Increment usage counts for all resolved tags in a single UPDATE.
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

	// Invalidate caches after mutation since usage counts changed.
	if err == nil {
		s.invalidateTagCaches()
		_ = s.syncTagsToGraph(tags)
	}

	return tags, err
}

// buildBatchInsertTagsSQL generates a single INSERT statement with multiple
// VALUE rows and ON CONFLICT DO NOTHING, avoiding N individual round-trips.
func buildBatchInsertTagsSQL(names []string) string {
	var b strings.Builder
	b.WriteString("INSERT INTO tags (id, name, usage_count, created_at) VALUES ")
	for i := range names {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("(gen_random_uuid(), $%d, 0, NOW())", i+1))
	}
	b.WriteString(" ON CONFLICT (name) DO NOTHING")
	return b.String()
}

// buildBatchInsertTagsArgs converts normalized tag names into a slice of
// interface{} arguments matching the positional parameters in the SQL.
func buildBatchInsertTagsArgs(names []string) []interface{} {
	args := make([]interface{}, len(names))
	for i, n := range names {
		args[i] = n
	}
	return args
}

// Trending returns the most-used tags.
// Results are cached in Redis for the configured TTL.
func (s *TagService) Trending(limit int) ([]model.Tag, error) {
	if limit <= 0 {
		limit = 20
	}

	cacheKey := fmt.Sprintf("%s:%d", tagCacheKeyTrending, limit)
	if cached, err := s.getTagsFromCache(cacheKey); err == nil && cached != nil {
		return cached, nil
	}

	var tags []model.Tag
	if err := s.db.Order("usage_count DESC").Limit(limit).Find(&tags).Error; err != nil {
		return nil, fmt.Errorf("failed to get trending tags: %w", err)
	}

	s.setTagsCache(cacheKey, tags)
	return tags, nil
}

func (s *TagService) syncTagsToGraph(tags []model.Tag) error {
	if s.graph == nil || len(tags) == 0 {
		return nil
	}

	payload := make([]map[string]any, 0, len(tags))
	for _, t := range tags {
		payload = append(payload, map[string]any{
			"id":         t.ID.String(),
			"name":       t.Name,
			"usageCount": t.UsageCount,
		})
	}

	_, err := s.graph.ExecuteWrite(context.Background(), func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(context.Background(), `
			UNWIND $tags AS tag
			MERGE (t:Tag {id: tag.id})
			SET t.name = tag.name,
			    t.usageCount = tag.usageCount
		`, map[string]any{"tags": payload})
		return nil, err
	})

	return err
}

// --- Redis cache helpers ---

// getTagsFromCache attempts to read a cached tag slice from Redis.
// Returns (nil, nil) if the cache is unavailable or the key does not exist.
func (s *TagService) getTagsFromCache(key string) ([]model.Tag, error) {
	if s.rdb == nil {
		return nil, nil
	}

	data, err := s.rdb.Get(context.Background(), key).Bytes()
	if err != nil {
		return nil, err
	}

	var tags []model.Tag
	if err := json.Unmarshal(data, &tags); err != nil {
		return nil, err
	}
	return tags, nil
}

// setTagsCache writes a tag slice to Redis with the default TTL.
func (s *TagService) setTagsCache(key string, tags []model.Tag) {
	s.setTagsCacheWithTTL(key, tags, s.cacheTTL)
}

// setTagsCacheWithTTL writes a tag slice to Redis with a specific TTL.
func (s *TagService) setTagsCacheWithTTL(key string, tags []model.Tag, ttl time.Duration) {
	if s.rdb == nil {
		return
	}

	data, err := json.Marshal(tags)
	if err != nil {
		return
	}

	s.rdb.Set(context.Background(), key, data, ttl)
}

// invalidateTagCaches removes all known tag cache keys so the next read
// fetches fresh data from the database. Search caches use a prefix scan.
func (s *TagService) invalidateTagCaches() {
	if s.rdb == nil {
		return
	}
	ctx := context.Background()

	// Delete the well-known keys.
	s.rdb.Del(ctx, tagCacheKeyList)

	// Delete trending keys (there could be multiple with different limits).
	trendingKeys, err := s.rdb.Keys(ctx, tagCacheKeyTrending+"*").Result()
	if err == nil && len(trendingKeys) > 0 {
		s.rdb.Del(ctx, trendingKeys...)
	}

	// Delete search prefix cache keys.
	searchKeys, err := s.rdb.Keys(ctx, tagCacheKeySearch+"*").Result()
	if err == nil && len(searchKeys) > 0 {
		s.rdb.Del(ctx, searchKeys...)
	}
}
