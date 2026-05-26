package service

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"

	"github.com/pulse/stone/internal/config"
	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/store"
)

// SuggestionType categorises how a suggestion was derived.
type SuggestionType string

const (
	SuggestionClosestTwin   SuggestionType = "closest_twin"
	SuggestionAdjacentTaste SuggestionType = "adjacent_taste"
	SuggestionPathAffinity  SuggestionType = "path_affinity"
	SuggestionSerendipity   SuggestionType = "serendipity"
)

// SuggestionResult represents a user suggestion with a "Bridge" — the reason
// two people are suggested to connect (shared tags, compatible aesthetics, etc).
type SuggestionResult struct {
	User           model.User     `json:"user"`
	SharedTags     int            `json:"shared_tags"`
	CommonTags     []model.Tag    `json:"common_tags"`
	Bridge         string         `json:"bridge"`
	Affinity       float64        `json:"affinity_score,omitempty"`
	SuggestionType SuggestionType `json:"suggestion_type,omitempty"`
	PathCount      int            `json:"path_count,omitempty"`
}

// BucketedSuggestions holds suggestions categorised into four discovery buckets.
type BucketedSuggestions struct {
	ClosestTwins  []SuggestionResult `json:"closest_twins"`
	AdjacentTaste []SuggestionResult `json:"adjacent_taste"`
	PathAffinity  []SuggestionResult `json:"path_affinity"`
	Serendipity   []SuggestionResult `json:"serendipity"`
}

type AffinityService struct {
	graph *store.GraphStore
	cfg   *config.Config
}

func NewAffinityService(graph *store.GraphStore, cfg *config.Config) *AffinityService {
	return &AffinityService{graph: graph, cfg: cfg}
}

// GetSuggestionsBucketed returns suggestions in four categorised buckets:
// path affinity, closest twins, adjacent taste, and serendipity.
func (s *AffinityService) GetSuggestionsBucketed(userID uuid.UUID, limit int) (*BucketedSuggestions, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}

	seen := map[uuid.UUID]struct{}{userID: {}}
	result := &BucketedSuggestions{}

	ctx := context.Background()

	// 1. Path affinity (max 3) — highest priority
	pathRows, err := s.queryPathAffinity(ctx, userID, 3)
	if err != nil {
		return nil, fmt.Errorf("path affinity: %w", err)
	}
	for _, row := range pathRows {
		if _, ok := seen[row.user.ID]; ok {
			continue
		}
		seen[row.user.ID] = struct{}{}

		commonTags, _ := s.getCommonTags(ctx, userID, row.user.ID)
		bridge := s.buildPathAffinityBridge(ctx, userID, row.user.ID, row.pathCount, commonTags)

		result.PathAffinity = append(result.PathAffinity, SuggestionResult{
			User:           row.user,
			SharedTags:     row.sharedTags,
			CommonTags:     commonTags,
			Bridge:         bridge,
			SuggestionType: SuggestionPathAffinity,
			PathCount:      row.pathCount,
		})
	}

	// 2. Closest twins — behavioral affinity from reactions (with decay)
	closest, err := s.queryClosestTwins(ctx, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("closest twins: %w", err)
	}
	for _, row := range closest {
		if _, ok := seen[row.user.ID]; ok {
			continue
		}
		seen[row.user.ID] = struct{}{}

		commonTags, _ := s.getCommonTags(ctx, userID, row.user.ID)
		bridgeTags := filterBridgeTags(commonTags)
		bridge := buildClosestTwinBridge(bridgeTags, len(commonTags))

		result.ClosestTwins = append(result.ClosestTwins, SuggestionResult{
			User:           row.user,
			SharedTags:     len(commonTags),
			CommonTags:     commonTags,
			Bridge:         bridge,
			Affinity:       row.affinityScore,
			SuggestionType: SuggestionClosestTwin,
		})
	}

	// 3. Adjacent taste — tag overlap
	adjacent, err := s.queryAdjacentTaste(ctx, userID, limit*3)
	if err != nil {
		return nil, fmt.Errorf("adjacent taste: %w", err)
	}
	for _, row := range adjacent {
		if len(result.AdjacentTaste) >= limit {
			break
		}
		if _, ok := seen[row.user.ID]; ok {
			continue
		}
		seen[row.user.ID] = struct{}{}

		commonTags, _ := s.getCommonTags(ctx, userID, row.user.ID)
		bridgeTags := filterBridgeTags(commonTags)
		sharedTags := row.sharedTags
		if sharedTags == 0 {
			sharedTags = len(commonTags)
		}
		bridge := buildBridge(bridgeTags, sharedTags)

		result.AdjacentTaste = append(result.AdjacentTaste, SuggestionResult{
			User:           row.user,
			SharedTags:     sharedTags,
			CommonTags:     commonTags,
			Bridge:         bridge,
			SuggestionType: SuggestionAdjacentTaste,
		})
	}

	// 4. Serendipity (max 3) — filter bubble prevention
	serendipity, err := s.querySerendipity(ctx, userID, 3)
	if err != nil {
		return nil, fmt.Errorf("serendipity: %w", err)
	}
	for _, row := range serendipity {
		if _, ok := seen[row.user.ID]; ok {
			continue
		}
		seen[row.user.ID] = struct{}{}

		commonTags, _ := s.getCommonTags(ctx, userID, row.user.ID)
		bridgeTags := filterBridgeTags(commonTags)
		bridge := buildSerendipityBridge(bridgeTags)

		result.Serendipity = append(result.Serendipity, SuggestionResult{
			User:           row.user,
			SharedTags:     len(commonTags),
			CommonTags:     commonTags,
			Bridge:         bridge,
			SuggestionType: SuggestionSerendipity,
		})
	}

	return result, nil
}

// GetSuggestions returns a flat list of suggestions ordered by priority.
// Backward-compatible wrapper around GetSuggestionsBucketed.
func (s *AffinityService) GetSuggestions(userID uuid.UUID, limit int) ([]SuggestionResult, error) {
	bucketed, err := s.GetSuggestionsBucketed(userID, limit)
	if err != nil {
		return nil, err
	}

	results := make([]SuggestionResult, 0, limit)
	for _, bucket := range [][]SuggestionResult{
		bucketed.PathAffinity,
		bucketed.ClosestTwins,
		bucketed.AdjacentTaste,
		bucketed.Serendipity,
	} {
		for _, r := range bucket {
			if len(results) >= limit {
				return results, nil
			}
			results = append(results, r)
		}
	}
	return results, nil
}

type affinityRow struct {
	user          model.User
	sharedTags    int
	pathCount     int
	affinityScore float64
}

func (s *AffinityService) queryPathAffinity(ctx context.Context, userID uuid.UUID, limit int) ([]affinityRow, error) {
	query := `
		MATCH (u1:User {id: $userID})-[:CREATED_PATH]->(:Path)-[:HAS_ITEM]->(:Content)-[:TAGGED]->(t:Tag)
		MATCH (u2:User)-[:CREATED_PATH]->(p2:Path)-[:HAS_ITEM]->(:Content)-[:TAGGED]->(t)
		WHERE u2.id <> $userID
		  AND NOT (u1)-[:FOLLOWS|BLOCKS]->(u2)
		  AND NOT (u2)-[:BLOCKS]->(u1)
		WITH u2, count(DISTINCT t) AS sharedTags, count(DISTINCT p2) AS pathCount
		WHERE sharedTags >= 2
		RETURN u2 { .id, .handle, .displayName, .avatarUrl, .bio } AS user,
		       sharedTags AS sharedTags,
		       pathCount AS pathCount
		ORDER BY sharedTags DESC, pathCount DESC
		LIMIT $limit
	`

	rows, err := s.readRows(ctx, query, map[string]any{
		"userID": userID.String(),
		"limit":  limit,
	})
	if err != nil {
		return nil, err
	}

	out := make([]affinityRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, affinityRow{
			user:       row.user,
			sharedTags: row.sharedTags,
			pathCount:  row.pathCount,
		})
	}
	return out, nil
}

func (s *AffinityService) queryClosestTwins(ctx context.Context, userID uuid.UUID, limit int) ([]affinityRow, error) {
	halfLife := s.cfg.AffinityHalfLife30DDays
	if halfLife <= 0 {
		halfLife = 15
	}
	lambda := math.Ln2 / halfLife

	query := `
		MATCH (u1:User {id: $userID})-[r1:REACTED]->(c:Content)<-[r2:REACTED]-(u2:User)
		WHERE u2.id <> $userID
		  AND NOT (u1)-[:FOLLOWS|BLOCKS]->(u2)
		  AND NOT (u2)-[:BLOCKS]->(u1)
		WITH u2,
		     sum(exp(-$lambda * duration.inDays(r1.createdAt, datetime()).days)) AS score
		WHERE score > 0
		RETURN u2 { .id, .handle, .displayName, .avatarUrl, .bio } AS user,
		       score AS affinityScore
		ORDER BY score DESC
		LIMIT $limit
	`

	rows, err := s.readRows(ctx, query, map[string]any{
		"userID": userID.String(),
		"lambda": lambda,
		"limit":  limit,
	})
	if err != nil {
		return nil, err
	}

	out := make([]affinityRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, affinityRow{
			user:          row.user,
			affinityScore: row.affinityScore,
		})
	}
	return out, nil
}

func (s *AffinityService) queryAdjacentTaste(ctx context.Context, userID uuid.UUID, limit int) ([]affinityRow, error) {
	query := `
		MATCH (u1:User {id: $userID})-[:CREATED]->(:Content)-[:TAGGED]->(t:Tag)<-[:TAGGED]-(:Content)<-[:CREATED]-(u2:User)
		WHERE u2.id <> $userID
		  AND NOT (u1)-[:FOLLOWS|BLOCKS]->(u2)
		  AND NOT (u2)-[:BLOCKS]->(u1)
		WITH u2, count(DISTINCT t) AS sharedTags
		RETURN u2 { .id, .handle, .displayName, .avatarUrl, .bio } AS user,
		       sharedTags AS sharedTags
		ORDER BY sharedTags DESC
		LIMIT $limit
	`

	return s.readRows(ctx, query, map[string]any{
		"userID": userID.String(),
		"limit":  limit,
	})
}

func (s *AffinityService) querySerendipity(ctx context.Context, userID uuid.UUID, limit int) ([]affinityRow, error) {
	query := `
		MATCH (u1:User {id: $userID})-[:CREATED]->(:Content)-[:TAGGED]->(t:Tag)<-[:TAGGED]-(:Content)<-[:CREATED]-(u2:User)
		WHERE u2.id <> $userID
		  AND NOT (u1)-[:FOLLOWS|BLOCKS]->(u2)
		  AND NOT (u2)-[:BLOCKS]->(u1)
		WITH u2, collect(DISTINCT t) AS tags, count(DISTINCT t) AS sharedTags, max(t.usageCount) AS maxUsage
		WHERE sharedTags >= 1 AND sharedTags <= 2 AND maxUsage >= 5
		RETURN u2 { .id, .handle, .displayName, .avatarUrl, .bio } AS user,
		       sharedTags AS sharedTags
		ORDER BY maxUsage DESC, rand()
		LIMIT $limit
	`

	return s.readRows(ctx, query, map[string]any{
		"userID": userID.String(),
		"limit":  limit,
	})
}

type mappedRow struct {
	user          model.User
	sharedTags    int
	pathCount     int
	affinityScore float64
}

func (s *AffinityService) readRows(ctx context.Context, query string, params map[string]any) ([]affinityRow, error) {
	raw, err := s.graph.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, params)
		if err != nil {
			return nil, err
		}

		rows := make([]affinityRow, 0)
		for result.Next(ctx) {
			record := result.Record()

			userAny, _ := record.Get("user")
			user, err := mapUser(userAny)
			if err != nil {
				return nil, err
			}

			row := affinityRow{user: user}

			if v, ok := record.Get("sharedTags"); ok {
				row.sharedTags = toInt(v)
			}
			if v, ok := record.Get("pathCount"); ok {
				row.pathCount = toInt(v)
			}
			if v, ok := record.Get("affinityScore"); ok {
				row.affinityScore = toFloat(v)
			}

			rows = append(rows, row)
		}
		if err := result.Err(); err != nil {
			return nil, err
		}
		return rows, nil
	})
	if err != nil {
		return nil, err
	}

	rows, ok := raw.([]affinityRow)
	if !ok {
		return nil, fmt.Errorf("failed to decode affinity rows")
	}
	return rows, nil
}

func (s *AffinityService) getCommonTags(ctx context.Context, userA, userB uuid.UUID) ([]model.Tag, error) {
	query := `
		MATCH (u1:User {id: $userA})-[:CREATED]->(:Content)-[:TAGGED]->(t:Tag)<-[:TAGGED]-(:Content)<-[:CREATED]-(u2:User {id: $userB})
		RETURN collect(DISTINCT t { .id, .name, .usageCount }) AS tags
	`

	raw, err := s.graph.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, map[string]any{
			"userA": userA.String(),
			"userB": userB.String(),
		})
		if err != nil {
			return nil, err
		}
		if !result.Next(ctx) {
			return []model.Tag{}, nil
		}
		tagsAny, _ := result.Record().Get("tags")
		tags, err := mapTags(tagsAny)
		if err != nil {
			return nil, err
		}
		return tags, nil
	})
	if err != nil {
		return nil, err
	}
	return raw.([]model.Tag), nil
}

// getPathTags returns the top tags from paths a user follows by a given creator.
func (s *AffinityService) getPathTags(ctx context.Context, userID, creatorID uuid.UUID) ([]model.Tag, error) {
	query := `
		MATCH (viewer:User {id: $viewerID})-[:FOLLOWS_PATH]->(p:Path)<-[:CREATED_PATH]-(creator:User {id: $creatorID})
		MATCH (p)-[:HAS_ITEM]->(:Content)-[:TAGGED]->(t:Tag)
		RETURN collect(DISTINCT t { .id, .name, .usageCount }) AS tags
	`

	raw, err := s.graph.ExecuteRead(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		result, err := tx.Run(ctx, query, map[string]any{
			"viewerID":  userID.String(),
			"creatorID": creatorID.String(),
		})
		if err != nil {
			return nil, err
		}
		if !result.Next(ctx) {
			return []model.Tag{}, nil
		}
		tagsAny, _ := result.Record().Get("tags")
		tags, err := mapTags(tagsAny)
		if err != nil {
			return nil, err
		}
		return tags, nil
	})
	if err != nil {
		return nil, err
	}
	return raw.([]model.Tag), nil
}

// --- Bridge builders ---

// filterBridgeTags removes tags with very low usage counts from bridge
// explanations to protect user privacy.
func filterBridgeTags(tags []model.Tag) []model.Tag {
	filtered := make([]model.Tag, 0, len(tags))
	for _, t := range tags {
		if t.UsageCount >= model.BridgeMinTagUsage {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// tagNames extracts up to max tag names prefixed with #.
func tagNames(tags []model.Tag, max int) []string {
	names := make([]string, 0, max)
	for i, t := range tags {
		if i >= max {
			break
		}
		names = append(names, "#"+t.Name)
	}
	return names
}

// joinTagNames joins tag names with commas and "and".
func joinTagNames(names []string) string {
	if len(names) == 0 {
		return ""
	}
	if len(names) == 1 {
		return names[0]
	}
	result := names[0]
	for i := 1; i < len(names); i++ {
		if i == len(names)-1 {
			result += " and " + names[i]
		} else {
			result += ", " + names[i]
		}
	}
	return result
}

func (s *AffinityService) buildPathAffinityBridge(ctx context.Context, userID, creatorID uuid.UUID, pathCount int, commonTags []model.Tag) string {
	pathTags, _ := s.getPathTags(ctx, userID, creatorID)
	bridgeTags := filterBridgeTags(pathTags)
	names := tagNames(bridgeTags, 3)
	if len(names) > 0 {
		return fmt.Sprintf("You followed %d of their paths about %s", pathCount, joinTagNames(names))
	}
	if len(commonTags) > 0 {
		filtered := filterBridgeTags(commonTags)
		cnames := tagNames(filtered, 2)
		if len(cnames) > 0 {
			return fmt.Sprintf("You followed %d of their paths — you also share %s", pathCount, joinTagNames(cnames))
		}
	}
	return fmt.Sprintf("You followed %d of their paths", pathCount)
}

func buildClosestTwinBridge(tags []model.Tag, totalShared int) string {
	names := tagNames(tags, 3)
	if len(names) == 0 {
		return "You have similar viewing patterns"
	}
	tagList := joinTagNames(names)
	if totalShared > len(names) {
		return fmt.Sprintf("You both linger on similar %s and %d more shared tags", tagList, totalShared-len(names))
	}
	return fmt.Sprintf("You both linger on similar %s content", tagList)
}

// buildBridge creates a human-readable explanation for adjacent taste suggestions.
func buildBridge(commonTags []model.Tag, sharedCount int) string {
	if len(commonTags) == 0 {
		return "You share similar interests"
	}

	names := tagNames(commonTags, 3)
	tagList := joinTagNames(names)

	if sharedCount > len(names) {
		return fmt.Sprintf("You both post about %s and %d more shared tags", tagList, sharedCount-len(names))
	}
	return fmt.Sprintf("You both post about %s", tagList)
}

func buildSerendipityBridge(tags []model.Tag) string {
	names := tagNames(tags, 1)
	if len(names) == 0 {
		return "A fresh perspective worth exploring"
	}
	return fmt.Sprintf("Different style, but you're both deep into %s", names[0])
}

// --- Mapping helpers ---

func mapUser(node any) (model.User, error) {
	props, ok := node.(map[string]any)
	if !ok {
		return model.User{}, fmt.Errorf("invalid user payload")
	}

	idStr, _ := props["id"].(string)
	id, err := uuid.Parse(idStr)
	if err != nil {
		return model.User{}, fmt.Errorf("invalid user id: %w", err)
	}

	user := model.User{
		ID:          id,
		Handle:      toString(props["handle"]),
		DisplayName: toString(props["displayName"]),
		AvatarURL:   toString(props["avatarUrl"]),
		Bio:         toString(props["bio"]),
	}

	return user, nil
}

func mapTags(tagsAny any) ([]model.Tag, error) {
	raw, ok := tagsAny.([]any)
	if !ok {
		return []model.Tag{}, nil
	}

	tags := make([]model.Tag, 0, len(raw))
	for _, item := range raw {
		props, ok := item.(map[string]any)
		if !ok {
			continue
		}
		idStr, _ := props["id"].(string)
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		tags = append(tags, model.Tag{
			ID:         id,
			Name:       toString(props["name"]),
			UsageCount: toInt(props["usageCount"]),
			CreatedAt:  time.Time{},
		})
	}
	return tags, nil
}

func toString(v any) string {
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

func toInt(v any) int {
	switch val := v.(type) {
	case int:
		return val
	case int64:
		return int(val)
	case float64:
		return int(val)
	default:
		return 0
	}
}

func toFloat(v any) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int:
		return float64(val)
	case int64:
		return float64(val)
	default:
		return 0
	}
}
