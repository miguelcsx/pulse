package service

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/model"
)

// SuggestionType categorises how a suggestion was derived.
type SuggestionType string

const (
	SuggestionClosestTwin  SuggestionType = "closest_twin"
	SuggestionAdjacentTaste SuggestionType = "adjacent_taste"
	SuggestionPathAffinity SuggestionType = "path_affinity"
	SuggestionSerendipity  SuggestionType = "serendipity"
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

type suggestionRow struct {
	ID            uuid.UUID `gorm:"column:id"`
	Handle        string    `gorm:"column:handle"`
	DisplayName   string    `gorm:"column:display_name"`
	AvatarURL     string    `gorm:"column:avatar_url"`
	Bio           string    `gorm:"column:bio"`
	SharedTags    int       `gorm:"column:shared_tags"`
	AffinityScore float64   `gorm:"column:affinity_score"`
}

type pathAffinityRow struct {
	ID          uuid.UUID `gorm:"column:id"`
	Handle      string    `gorm:"column:handle"`
	DisplayName string    `gorm:"column:display_name"`
	AvatarURL   string    `gorm:"column:avatar_url"`
	Bio         string    `gorm:"column:bio"`
	PathCount   int       `gorm:"column:path_count"`
}

// AffinityService provides user suggestions based on shared tag overlap.
type AffinityService struct {
	db *gorm.DB
}

func NewAffinityService(db *gorm.DB) *AffinityService {
	return &AffinityService{db: db}
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

	// 1. Path affinity (max 3) — highest priority
	pathRows, err := s.getPathAffinityRows(userID, 3)
	if err != nil {
		return nil, fmt.Errorf("path affinity: %w", err)
	}
	for _, r := range pathRows {
		if _, ok := seen[r.ID]; ok {
			continue
		}
		seen[r.ID] = struct{}{}

		user := model.User{
			ID: r.ID, Handle: r.Handle, DisplayName: r.DisplayName,
			AvatarURL: r.AvatarURL, Bio: r.Bio,
		}

		commonTags, _ := s.getCommonTags(userID, r.ID)
		bridge := s.buildPathAffinityBridge(userID, r.ID, r.PathCount, commonTags)

		result.PathAffinity = append(result.PathAffinity, SuggestionResult{
			User:           user,
			SharedTags:     len(commonTags),
			CommonTags:     commonTags,
			Bridge:         bridge,
			SuggestionType: SuggestionPathAffinity,
			PathCount:      r.PathCount,
		})
	}

	// 2. Closest twins — behavioral affinity from edges
	edgeRows, err := s.getEdgeRows(userID, limit)
	if err != nil {
		return nil, fmt.Errorf("closest twins: %w", err)
	}
	for _, r := range edgeRows {
		if _, ok := seen[r.ID]; ok {
			continue
		}
		seen[r.ID] = struct{}{}

		user := model.User{
			ID: r.ID, Handle: r.Handle, DisplayName: r.DisplayName,
			AvatarURL: r.AvatarURL, Bio: r.Bio,
		}

		commonTags, _ := s.getCommonTags(userID, r.ID)
		bridgeTags := filterBridgeTags(commonTags)
		bridge := buildClosestTwinBridge(bridgeTags, len(commonTags))

		result.ClosestTwins = append(result.ClosestTwins, SuggestionResult{
			User:           user,
			SharedTags:     len(commonTags),
			CommonTags:     commonTags,
			Bridge:         bridge,
			Affinity:       r.AffinityScore,
			SuggestionType: SuggestionClosestTwin,
		})
	}

	// 3. Adjacent taste — tag overlap
	tagRows, err := s.getTagOverlapRows(userID, limit*3)
	if err != nil {
		return nil, fmt.Errorf("adjacent taste: %w", err)
	}
	for _, r := range tagRows {
		if len(result.AdjacentTaste) >= limit {
			break
		}
		if _, ok := seen[r.ID]; ok {
			continue
		}
		seen[r.ID] = struct{}{}

		user := model.User{
			ID: r.ID, Handle: r.Handle, DisplayName: r.DisplayName,
			AvatarURL: r.AvatarURL, Bio: r.Bio,
		}

		commonTags, _ := s.getCommonTags(userID, r.ID)
		bridgeTags := filterBridgeTags(commonTags)
		sharedTags := r.SharedTags
		if sharedTags == 0 {
			sharedTags = len(commonTags)
		}
		bridge := buildBridge(bridgeTags, sharedTags)

		result.AdjacentTaste = append(result.AdjacentTaste, SuggestionResult{
			User:           user,
			SharedTags:     sharedTags,
			CommonTags:     commonTags,
			Bridge:         bridge,
			SuggestionType: SuggestionAdjacentTaste,
		})
	}

	// 4. Serendipity (max 3) — filter bubble prevention
	serendipityRows, err := s.getSerendipityRows(userID, 3)
	if err != nil {
		return nil, fmt.Errorf("serendipity: %w", err)
	}
	for _, r := range serendipityRows {
		if _, ok := seen[r.ID]; ok {
			continue
		}
		seen[r.ID] = struct{}{}

		user := model.User{
			ID: r.ID, Handle: r.Handle, DisplayName: r.DisplayName,
			AvatarURL: r.AvatarURL, Bio: r.Bio,
		}

		commonTags, _ := s.getCommonTags(userID, r.ID)
		bridgeTags := filterBridgeTags(commonTags)
		bridge := buildSerendipityBridge(bridgeTags)

		result.Serendipity = append(result.Serendipity, SuggestionResult{
			User:           user,
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

// --- Query methods ---

// getPathAffinityRows finds creators whose paths the user has followed (>=2).
func (s *AffinityService) getPathAffinityRows(userID uuid.UUID, limit int) ([]pathAffinityRow, error) {
	var rows []pathAffinityRow
	err := s.db.Raw(`
		SELECT p.creator_id AS id, u.handle, u.display_name, u.avatar_url, u.bio,
		       COUNT(DISTINCT pf.path_id) AS path_count
		FROM path_follows pf
		JOIN paths p ON p.id = pf.path_id
		JOIN users u ON u.id = p.creator_id
		WHERE pf.user_id = ? AND p.creator_id != ?
		  AND NOT EXISTS (
		      SELECT 1 FROM follows WHERE follower_id = ? AND followee_id = p.creator_id
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM blocks WHERE blocker_id = ? AND blocked_id = p.creator_id
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM blocks WHERE blocked_id = ? AND blocker_id = p.creator_id
		  )
		GROUP BY p.creator_id, u.handle, u.display_name, u.avatar_url, u.bio
		HAVING COUNT(DISTINCT pf.path_id) >= 2
		ORDER BY path_count DESC
		LIMIT ?
	`, userID, userID, userID, userID, userID, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// getSerendipityRows finds users with low tag overlap (1-2 shared) but at
// least one high-quality tag, preventing filter bubbles.
func (s *AffinityService) getSerendipityRows(userID uuid.UUID, limit int) ([]suggestionRow, error) {
	var rows []suggestionRow
	err := s.db.Raw(`
		SELECT u.id, u.handle, u.display_name, u.avatar_url, u.bio,
		       COUNT(DISTINCT ct.tag_id) AS shared_tags,
		       0.0 AS affinity_score
		FROM users u
		JOIN contents c ON c.creator_id = u.id
		JOIN content_tags ct ON ct.content_id = c.id
		JOIN tags t ON t.id = ct.tag_id
		WHERE ct.tag_id IN (
			SELECT DISTINCT ct2.tag_id FROM contents c2
			JOIN content_tags ct2 ON ct2.content_id = c2.id
			WHERE c2.creator_id = ?
		)
		  AND u.id != ?
		  AND NOT EXISTS (
		      SELECT 1 FROM follows WHERE follower_id = ? AND followee_id = u.id
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM blocks WHERE blocker_id = ? AND blocked_id = u.id
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM blocks WHERE blocked_id = ? AND blocker_id = u.id
		  )
		GROUP BY u.id, u.handle, u.display_name, u.avatar_url, u.bio
		HAVING COUNT(DISTINCT ct.tag_id) BETWEEN 1 AND 2
		   AND MAX(t.usage_count) >= 5
		ORDER BY MAX(t.usage_count) DESC, RANDOM()
		LIMIT ?
	`, userID, userID, userID, userID, userID, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// getEdgeRows retrieves suggestion candidates from pre-computed affinity edges.
func (s *AffinityService) getEdgeRows(userID uuid.UUID, limit int) ([]suggestionRow, error) {
	var rows []suggestionRow
	err := s.db.Raw(`
		SELECT u.id, u.handle, u.display_name, u.avatar_url, u.bio,
		       0 AS shared_tags,
		       e.score_30d AS affinity_score
		FROM user_affinity_edges e
		JOIN users u ON u.id = e.other_user_id
		WHERE e.user_id = ?
		  AND u.id != ?
		  AND e.score_30d > 0
		  AND NOT EXISTS (
		      SELECT 1 FROM follows WHERE follower_id = ? AND followee_id = u.id
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM blocks WHERE blocker_id = ? AND blocked_id = u.id
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM blocks WHERE blocked_id = ? AND blocker_id = u.id
		  )
		ORDER BY e.score_30d DESC, e.updated_at DESC
		LIMIT ?
	`, userID, userID, userID, userID, userID, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// getTagOverlapRows retrieves suggestion candidates based on shared tag usage.
func (s *AffinityService) getTagOverlapRows(userID uuid.UUID, limit int) ([]suggestionRow, error) {
	var rows []suggestionRow
	err := s.db.Raw(`
		SELECT u.id, u.handle, u.display_name, u.avatar_url, u.bio,
		       COUNT(DISTINCT ct.tag_id) AS shared_tags,
		       0.0 AS affinity_score
		FROM users u
		JOIN contents c ON c.creator_id = u.id
		JOIN content_tags ct ON ct.content_id = c.id
		WHERE ct.tag_id IN (
			SELECT DISTINCT ct2.tag_id FROM contents c2
			JOIN content_tags ct2 ON ct2.content_id = c2.id
			WHERE c2.creator_id = ?
		)
		  AND u.id != ?
		  AND NOT EXISTS (
		      SELECT 1 FROM follows WHERE follower_id = ? AND followee_id = u.id
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM blocks WHERE blocker_id = ? AND blocked_id = u.id
		  )
		  AND NOT EXISTS (
		      SELECT 1 FROM blocks WHERE blocked_id = ? AND blocker_id = u.id
		  )
		GROUP BY u.id, u.handle, u.display_name, u.avatar_url, u.bio
		ORDER BY shared_tags DESC, u.created_at DESC
		LIMIT ?
	`, userID, userID, userID, userID, userID, limit).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *AffinityService) getCommonTags(userA, userB uuid.UUID) ([]model.Tag, error) {
	var tags []model.Tag
	err := s.db.Raw(`
		SELECT DISTINCT t.*
		FROM tags t
		WHERE t.id IN (
			SELECT ct.tag_id FROM contents c
			JOIN content_tags ct ON ct.content_id = c.id
			WHERE c.creator_id = ?
		)
		AND t.id IN (
			SELECT ct.tag_id FROM contents c
			JOIN content_tags ct ON ct.content_id = c.id
			WHERE c.creator_id = ?
		)
		ORDER BY t.usage_count DESC
		LIMIT 10
	`, userA, userB).Scan(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
}

// getPathTags returns the top tags from paths a user follows by a given creator.
func (s *AffinityService) getPathTags(userID, creatorID uuid.UUID) ([]model.Tag, error) {
	var tags []model.Tag
	err := s.db.Raw(`
		SELECT DISTINCT t.*
		FROM tags t
		JOIN content_tags ct ON ct.tag_id = t.id
		JOIN path_items pi ON pi.content_id = ct.content_id
		JOIN path_follows pf ON pf.path_id = pi.path_id
		JOIN paths p ON p.id = pi.path_id
		WHERE pf.user_id = ? AND p.creator_id = ?
		ORDER BY t.usage_count DESC
		LIMIT 5
	`, userID, creatorID).Scan(&tags).Error
	if err != nil {
		return nil, err
	}
	return tags, nil
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

func (s *AffinityService) buildPathAffinityBridge(userID, creatorID uuid.UUID, pathCount int, commonTags []model.Tag) string {
	pathTags, _ := s.getPathTags(userID, creatorID)
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
