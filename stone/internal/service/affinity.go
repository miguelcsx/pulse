package service

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/model"
)

// SuggestionResult represents a user suggestion with a "Bridge" — the reason
// two people are suggested to connect (shared tags, compatible aesthetics, etc).
type SuggestionResult struct {
	User       model.User  `json:"user"`
	SharedTags int         `json:"shared_tags"`
	CommonTags []model.Tag `json:"common_tags"`
	Bridge     string      `json:"bridge"` // human-readable explanation
	Affinity   float64     `json:"affinity_score,omitempty"`
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

// AffinityService provides user suggestions based on shared tag overlap.
type AffinityService struct {
	db *gorm.DB
}

func NewAffinityService(db *gorm.DB) *AffinityService {
	return &AffinityService{db: db}
}

// GetSuggestions returns user suggestions ordered by shared tag count.
// Excludes users already followed or blocked. Each result includes a Bridge
// explaining why the connection is suggested.
func (s *AffinityService) GetSuggestions(userID uuid.UUID, limit int) ([]SuggestionResult, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}

	seen := map[uuid.UUID]struct{}{userID: {}}
	results := make([]SuggestionResult, 0, limit)
	addRows := func(rows []suggestionRow) error {
		for _, r := range rows {
			if len(results) >= limit {
				break
			}
			if _, ok := seen[r.ID]; ok {
				continue
			}
			seen[r.ID] = struct{}{}

			user := model.User{
				ID:          r.ID,
				Handle:      r.Handle,
				DisplayName: r.DisplayName,
				AvatarURL:   r.AvatarURL,
				Bio:         r.Bio,
			}

			commonTags, err := s.getCommonTags(userID, r.ID)
			if err != nil {
				return fmt.Errorf("failed to get common tags for user %s: %w", r.ID, err)
			}

			// Filter tags for bridge privacy: exclude very niche tags
			// that could reveal personal interests.
			bridgeTags := filterBridgeTags(commonTags)

			sharedTags := r.SharedTags
			if sharedTags == 0 {
				sharedTags = len(commonTags)
			}
			bridge := buildBridge(bridgeTags, sharedTags)
			if len(bridgeTags) == 0 && r.AffinityScore > 0 {
				bridge = "You have similar viewing patterns"
			}

			results = append(results, SuggestionResult{
				User:       user,
				SharedTags: sharedTags,
				CommonTags: commonTags,
				Bridge:     bridge,
				Affinity:   r.AffinityScore,
			})
		}
		return nil
	}

	edgeRows, err := s.getEdgeRows(userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get edge-based suggestions: %w", err)
	}
	if err := addRows(edgeRows); err != nil {
		return nil, err
	}

	if len(results) < limit {
		fallbackRows, err := s.getTagOverlapRows(userID, limit*3)
		if err != nil {
			return nil, fmt.Errorf("failed to get fallback suggestions: %w", err)
		}
		if err := addRows(fallbackRows); err != nil {
			return nil, err
		}
	}

	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// getEdgeRows retrieves suggestion candidates from pre-computed affinity edges.
// Uses NOT EXISTS instead of NOT IN for optimal query performance.
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
// Uses NOT EXISTS instead of NOT IN for optimal query performance.
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

// filterBridgeTags removes tags with very low usage counts from bridge
// explanations to protect user privacy. Tags that are used by only a few
// people could inadvertently reveal very specific personal interests.
func filterBridgeTags(tags []model.Tag) []model.Tag {
	filtered := make([]model.Tag, 0, len(tags))
	for _, t := range tags {
		if t.UsageCount >= model.BridgeMinTagUsage {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

// buildBridge creates a human-readable explanation for why two users are connected.
func buildBridge(commonTags []model.Tag, sharedCount int) string {
	if len(commonTags) == 0 {
		return "You share similar interests"
	}

	names := make([]string, 0, 3)
	for i, t := range commonTags {
		if i >= 3 {
			break
		}
		names = append(names, "#"+t.Name)
	}

	tagList := names[0]
	for i := 1; i < len(names); i++ {
		if i == len(names)-1 {
			tagList += " and " + names[i]
		} else {
			tagList += ", " + names[i]
		}
	}

	if sharedCount > len(names) {
		return fmt.Sprintf("You both post about %s and %d more shared tags", tagList, sharedCount-len(names))
	}
	return fmt.Sprintf("You both post about %s", tagList)
}
