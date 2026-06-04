package service

import (
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/config"
	"github.com/pulse/stone/internal/model"
)

// RoomContext provides room information for a feed item whose tags match an active room.
type RoomContext struct {
	RoomID      uuid.UUID `json:"room_id"`
	Tags        []string  `json:"tags"`
	MemberCount int64     `json:"member_count"`
}

const (
	FeedUnitMoment = "moment"
	FeedUnitAsk    = "ask"
)

// FeedItem is one unit in the affinity path. Moments are content cards; asks
// are bridge cards addressed to people whose context overlaps the question.
type FeedItem struct {
	ID          uuid.UUID      `json:"id"`
	UnitType    string         `json:"unit_type"`
	Content     *model.Content `json:"content,omitempty"`
	Bridge      *model.Bridge  `json:"bridge,omitempty"`
	RoomContext *RoomContext   `json:"room_context,omitempty"`
	Reason      string         `json:"reason,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
}

type scoredContent struct {
	model.Content
	score float64
	tier  int // 1=primary, 2=discovery, 3=trending
}

// FeedService provides the main content feed using a three-tier scoring pipeline:
// primary (followed users), discovery (tag-affinity), and trending (cold-start backfill).
type FeedService struct {
	db    *gorm.DB
	rooms *RoomService
	cfg   *config.Config
}

// NewFeedService creates a new FeedService.
func NewFeedService(db *gorm.DB, rooms *RoomService, cfg *config.Config) *FeedService {
	return &FeedService{db: db, rooms: rooms, cfg: cfg}
}

// GetFeed returns a scored, mixed feed using a three-tier candidate pipeline:
//
//	Tier 1 (primary):   content from followed users + self, ranked by affinity
//	Tier 2 (discovery): content from non-followed users sharing tags, last 7 days
//	Tier 3 (trending):  recent content from anyone as cold-start backfill
//
// Scoring: 0.4*affinity + 0.35*recency + 0.25*quality (configurable weights).
// Diversity: 4th+ item from the same creator gets a 50% score penalty.
// Mix ratio: ~70% primary / ~30% discovery, trending fills any gaps.
func (s *FeedService) GetFeed(userID uuid.UUID, cursor string, limit int) ([]FeedItem, string, bool, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	page := 0
	if cursor != "" {
		parsed, err := decodeScoredCursor(cursor)
		if err != nil {
			return nil, "", false, err
		}
		page = parsed
	}

	now := time.Now()

	// Fetch enough candidates to fill all pages up to current + hasMore check
	needed := (page+1)*limit + 1
	headroom := 3
	fetchCount := needed * headroom

	primaryFetch := int(math.Ceil(float64(fetchCount) * (1 - s.cfg.FeedDiscoveryRatio)))
	discoveryFetch := fetchCount - primaryFetch

	// Tier 1: content from followed users + self
	primaryCandidates := s.fetchPrimaryCandidates(userID, primaryFetch)

	// Tier 2: tag-affinity discovery from non-followed users
	primaryIDs := extractContentIDs(primaryCandidates)
	discoveryCandidates := s.fetchDiscoveryCandidates(userID, discoveryFetch, primaryIDs)

	// Tier 3: trending backfill (cold-start safety net)
	allExcludeIDs := append(primaryIDs, extractContentIDs(discoveryCandidates)...)
	trendingCandidates := s.fetchTrendingBackfill(userID, needed, allExcludeIDs)

	// Batch-load affinity scores for all candidate creators
	creatorIDs := collectCreatorIDs(primaryCandidates, discoveryCandidates, trendingCandidates)
	affinityMap := s.loadAffinityScores(userID, creatorIDs)
	followedSet := s.loadFollowedSet(userID)

	// Batch-load reaction counts for scoring
	allContentIDs := collectContentIDs(primaryCandidates, discoveryCandidates, trendingCandidates)
	reactionCounts := s.loadReactionCounts(allContentIDs)

	// Score each tier
	scoredPrimary := s.scoreSlice(primaryCandidates, affinityMap, followedSet, reactionCounts, now, 1)
	scoredDiscovery := s.scoreSlice(discoveryCandidates, affinityMap, followedSet, reactionCounts, now, 2)
	scoredTrending := s.scoreSlice(trendingCandidates, affinityMap, followedSet, reactionCounts, now, 3)

	sortScored(scoredPrimary)
	sortScored(scoredDiscovery)
	sortScored(scoredTrending)

	// Enforce mix ratio: ~70% primary, ~30% discovery, trending fills gaps
	primaryTarget := int(math.Ceil(float64(needed) * (1 - s.cfg.FeedDiscoveryRatio)))
	discoveryTarget := needed - primaryTarget

	merged := takeN(scoredPrimary, primaryTarget)
	primaryShortfall := primaryTarget - len(merged)

	discoveryTake := discoveryTarget + primaryShortfall
	merged = append(merged, takeN(scoredDiscovery, discoveryTake)...)

	totalShortfall := needed - len(merged)
	if totalShortfall > 0 {
		merged = append(merged, takeN(scoredTrending, totalShortfall)...)
	}

	// Final sort by score, then apply diversity penalty
	sortScored(merged)
	merged = s.applyDiversity(merged)

	// Paginate
	offset := page * limit
	if offset >= len(merged) {
		return []FeedItem{}, "", false, nil
	}
	end := offset + limit + 1
	if end > len(merged) {
		end = len(merged)
	}

	pageSlice := merged[offset:end]
	hasMore := len(pageSlice) > limit
	if hasMore {
		pageSlice = pageSlice[:limit]
	}

	// Extract contents for hydration
	contents := make([]model.Content, len(pageSlice))
	for i, sc := range pageSlice {
		contents[i] = sc.Content
	}

	if err := s.hydrateReactionCounts(contents); err != nil {
		return nil, "", false, err
	}

	roomMap := s.enrichWithRoomContext(contents)

	items := make([]FeedItem, 0, len(contents)+3)
	for i, c := range contents {
		content := contents[i]
		items = append(items, FeedItem{
			ID:          c.ID,
			UnitType:    FeedUnitMoment,
			Content:     &content,
			RoomContext: roomMap[c.ID],
			CreatedAt:   c.CreatedAt,
		})
	}
	items = s.mixAffinityAsks(userID, items)

	var nextCursor string
	if hasMore {
		nextCursor = encodeScoredCursor(page + 1)
	}

	return items, nextCursor, hasMore, nil
}

func (s *FeedService) mixAffinityAsks(userID uuid.UUID, items []FeedItem) []FeedItem {
	var bridges []model.Bridge
	if err := s.db.Preload("Ask.User").
		Preload("RecommendedUser").
		Where("recommended_user_id = ? AND status <> ?", userID, model.BridgeStatusDismissed).
		Order("created_at DESC").
		Limit(3).
		Find(&bridges).Error; err != nil || len(bridges) == 0 {
		return items
	}

	units := make([]FeedItem, 0, len(items)+len(bridges))
	bridgeIndex := 0
	for i, item := range items {
		if i == 1 || (i == 4 && len(bridges) > 1) || (i == 8 && len(bridges) > 2) {
			bridge := bridges[bridgeIndex]
			units = append(units, bridgeFeedItem(bridge))
			bridgeIndex++
		}
		units = append(units, item)
	}
	for bridgeIndex < len(bridges) {
		bridge := bridges[bridgeIndex]
		units = append(units, bridgeFeedItem(bridge))
		bridgeIndex++
	}
	return units
}

func bridgeFeedItem(bridge model.Bridge) FeedItem {
	return FeedItem{
		ID:        bridge.ID,
		UnitType:  FeedUnitAsk,
		Bridge:    &bridge,
		Reason:    bridge.Reason,
		CreatedAt: bridge.CreatedAt,
	}
}

// ---------------------------------------------------------------------------
// Candidate fetchers
// ---------------------------------------------------------------------------

// fetchPrimaryCandidates retrieves content from followed users and self.
func (s *FeedService) fetchPrimaryCandidates(userID uuid.UUID, limit int) []model.Content {
	var contents []model.Content
	s.db.Preload("Creator").Preload("Tags").
		Where(`(creator_id = ? OR creator_id IN (
			SELECT followee_id FROM follows WHERE follower_id = ?
		))`, userID, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocker_id = ? AND blocked_id = contents.creator_id
		)`, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocked_id = ? AND blocker_id = contents.creator_id
		)`, userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&contents)
	return contents
}

// fetchDiscoveryCandidates retrieves content from non-followed users whose
// content shares tags with the current user's content (last 7 days).
func (s *FeedService) fetchDiscoveryCandidates(userID uuid.UUID, limit int, excludeIDs []uuid.UUID) []model.Content {
	var contents []model.Content
	query := s.db.Preload("Creator").Preload("Tags").
		Where(`creator_id != ?`, userID).
		Where(`creator_id NOT IN (
			SELECT followee_id FROM follows WHERE follower_id = ?
		)`, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocker_id = ? AND blocked_id = contents.creator_id
		)`, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocked_id = ? AND blocker_id = contents.creator_id
		)`, userID).
		Where(`created_at > ?`, time.Now().Add(-7*24*time.Hour)).
		Where(`id IN (
			SELECT ct.content_id FROM content_tags ct
			WHERE ct.tag_id IN (
				SELECT ct2.tag_id FROM contents c2
				JOIN content_tags ct2 ON ct2.content_id = c2.id
				WHERE c2.creator_id = ?
			)
		)`, userID).
		Order("created_at DESC").
		Limit(limit)

	if len(excludeIDs) > 0 {
		query = query.Where("id NOT IN ?", excludeIDs)
	}

	query.Find(&contents)
	return contents
}

// fetchTrendingBackfill retrieves recent content from any non-blocked user
// as a cold-start safety net. The scoring formula naturally boosts items
// with more reactions via the quality component.
func (s *FeedService) fetchTrendingBackfill(userID uuid.UUID, limit int, excludeIDs []uuid.UUID) []model.Content {
	var contents []model.Content
	query := s.db.Preload("Creator").Preload("Tags").
		Where(`creator_id != ?`, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocker_id = ? AND blocked_id = contents.creator_id
		)`, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocked_id = ? AND blocker_id = contents.creator_id
		)`, userID).
		Where(`created_at > ?`, time.Now().Add(-s.cfg.FeedTrendingMaxAge)).
		Order("created_at DESC").
		Limit(limit)

	if len(excludeIDs) > 0 {
		query = query.Where("id NOT IN ?", excludeIDs)
	}

	query.Find(&contents)
	return contents
}

// ---------------------------------------------------------------------------
// Scoring
// ---------------------------------------------------------------------------

// loadAffinityScores batch-loads score_7d values for the given creator IDs.
func (s *FeedService) loadAffinityScores(userID uuid.UUID, creatorIDs []uuid.UUID) map[uuid.UUID]float64 {
	result := make(map[uuid.UUID]float64, len(creatorIDs))
	if len(creatorIDs) == 0 {
		return result
	}

	type row struct {
		OtherUserID uuid.UUID `gorm:"column:other_user_id"`
		Score7D     float64   `gorm:"column:score_7d"`
	}

	var rows []row
	s.db.Raw(`
		SELECT other_user_id, score_7d
		FROM user_affinity_edges
		WHERE user_id = ? AND other_user_id IN ?
	`, userID, creatorIDs).Scan(&rows)

	for _, r := range rows {
		result[r.OtherUserID] = r.Score7D
	}
	return result
}

// loadFollowedSet returns the set of user IDs that userID follows.
func (s *FeedService) loadFollowedSet(userID uuid.UUID) map[uuid.UUID]bool {
	type row struct {
		FolloweeID uuid.UUID `gorm:"column:followee_id"`
	}

	var rows []row
	s.db.Raw(`SELECT followee_id FROM follows WHERE follower_id = ?`, userID).Scan(&rows)

	result := make(map[uuid.UUID]bool, len(rows))
	for _, r := range rows {
		result[r.FolloweeID] = true
	}
	return result
}

// loadReactionCounts batch-loads total reaction counts for scoring.
func (s *FeedService) loadReactionCounts(contentIDs []uuid.UUID) map[uuid.UUID]int {
	result := make(map[uuid.UUID]int, len(contentIDs))
	if len(contentIDs) == 0 {
		return result
	}

	type row struct {
		ContentID uuid.UUID `gorm:"column:content_id"`
		Total     int       `gorm:"column:total"`
	}

	var rows []row
	s.db.Raw(`
		SELECT content_id, COUNT(*) AS total
		FROM reactions
		WHERE content_id IN ?
		GROUP BY content_id
	`, contentIDs).Scan(&rows)

	for _, r := range rows {
		result[r.ContentID] = r.Total
	}
	return result
}

// affinityForCreator returns the affinity score for a content creator.
// Uses edge data if available, 0.5 default for followed users, 0.0 for unknown.
func affinityForCreator(creatorID uuid.UUID, affinityMap map[uuid.UUID]float64, followedSet map[uuid.UUID]bool) float64 {
	if score, ok := affinityMap[creatorID]; ok {
		return score
	}
	if followedSet[creatorID] {
		return 0.5
	}
	return 0.0
}

// scoreItem computes the composite feed score for a single content item.
// score = (w_affinity * affinity) + (w_recency * recency) + (w_quality * quality)
func (s *FeedService) scoreItem(affinity float64, createdAt time.Time, reactionCount int, now time.Time) float64 {
	hours := now.Sub(createdAt).Hours()
	recency := 1.0 / (1.0 + hours/24.0)
	quality := math.Min(math.Log1p(float64(reactionCount))/3.0, 1.0)

	return (s.cfg.FeedAffinityWeight * affinity) +
		(s.cfg.FeedRecencyWeight * recency) +
		(s.cfg.FeedQualityWeight * quality)
}

// scoreSlice scores a slice of content items and returns scored wrappers.
func (s *FeedService) scoreSlice(
	contents []model.Content,
	affinityMap map[uuid.UUID]float64,
	followedSet map[uuid.UUID]bool,
	reactionCounts map[uuid.UUID]int,
	now time.Time,
	tier int,
) []scoredContent {
	scored := make([]scoredContent, len(contents))
	for i, c := range contents {
		aff := affinityForCreator(c.CreatorID, affinityMap, followedSet)
		scored[i] = scoredContent{
			Content: c,
			score:   s.scoreItem(aff, c.CreatedAt, reactionCounts[c.ID], now),
			tier:    tier,
		}
	}
	return scored
}

// applyDiversity penalizes the 4th+ item from the same creator by 50%,
// then re-sorts by score.
func (s *FeedService) applyDiversity(items []scoredContent) []scoredContent {
	creatorCount := make(map[uuid.UUID]int)
	for i := range items {
		creatorCount[items[i].CreatorID]++
		if creatorCount[items[i].CreatorID] > 3 {
			items[i].score *= 0.5
		}
	}
	sortScored(items)
	return items
}

// ---------------------------------------------------------------------------
// Cursor encoding (page-based for scored feeds)
// ---------------------------------------------------------------------------

func encodeScoredCursor(page int) string {
	return fmt.Sprintf("p:%d", page)
}

func decodeScoredCursor(cursor string) (int, error) {
	if strings.HasPrefix(cursor, "p:") {
		page, err := strconv.Atoi(cursor[2:])
		if err != nil || page < 0 {
			return 0, fmt.Errorf("malformed scored cursor")
		}
		return page, nil
	}
	return 0, fmt.Errorf("malformed scored cursor")
}

// ---------------------------------------------------------------------------
// Enrichment (unchanged from original)
// ---------------------------------------------------------------------------

// enrichWithRoomContext finds active rooms matching feed items' tags and returns
// a map of content ID to room context.
func (s *FeedService) enrichWithRoomContext(contents []model.Content) map[uuid.UUID]*RoomContext {
	result := make(map[uuid.UUID]*RoomContext)
	if len(contents) == 0 || s.rooms == nil {
		return result
	}

	// Collect all unique tag IDs from feed items
	allTagIDs := make(map[uuid.UUID]bool)
	for _, c := range contents {
		for _, t := range c.Tags {
			allTagIDs[t.ID] = true
		}
	}
	if len(allTagIDs) == 0 {
		return result
	}

	tagIDSlice := make([]uuid.UUID, 0, len(allTagIDs))
	for id := range allTagIDs {
		tagIDSlice = append(tagIDSlice, id)
	}

	// Find active rooms that share any of these tags, with member counts and tag names
	type roomRow struct {
		RoomID      uuid.UUID `gorm:"column:room_id"`
		ClusterKey  string    `gorm:"column:cluster_key"`
		TagID       uuid.UUID `gorm:"column:tag_id"`
		TagName     string    `gorm:"column:tag_name"`
		MemberCount int64     `gorm:"column:member_count"`
	}

	var rows []roomRow
	err := s.db.Raw(`
		SELECT r.id AS room_id, r.cluster_key, rt.tag_id, t.name AS tag_name,
			(SELECT COUNT(*) FROM room_members rm WHERE rm.room_id = r.id) AS member_count
		FROM rooms r
		JOIN room_tags rt ON rt.room_id = r.id
		JOIN tags t ON t.id = rt.tag_id
		WHERE r.expires_at > ?
		AND rt.tag_id IN ?
	`, time.Now(), tagIDSlice).Scan(&rows).Error
	if err != nil {
		return result
	}

	// Group rows by room: cluster_key → room info
	type roomInfo struct {
		RoomID      uuid.UUID
		ClusterKey  string
		Tags        []string
		TagIDs      []uuid.UUID
		MemberCount int64
	}
	roomsByKey := make(map[string]*roomInfo)
	for _, row := range rows {
		ri, ok := roomsByKey[row.ClusterKey]
		if !ok {
			ri = &roomInfo{
				RoomID:      row.RoomID,
				ClusterKey:  row.ClusterKey,
				MemberCount: row.MemberCount,
			}
			roomsByKey[row.ClusterKey] = ri
		}
		// Deduplicate tags
		found := false
		for _, tid := range ri.TagIDs {
			if tid == row.TagID {
				found = true
				break
			}
		}
		if !found {
			ri.TagIDs = append(ri.TagIDs, row.TagID)
			ri.Tags = append(ri.Tags, row.TagName)
		}
	}

	// For each content item, compute its cluster key and match to a room
	for _, c := range contents {
		if len(c.Tags) == 0 {
			continue
		}
		tagIDs := make([]uuid.UUID, len(c.Tags))
		for i, t := range c.Tags {
			tagIDs[i] = t.ID
		}
		key := BuildClusterKey(tagIDs)
		if ri, ok := roomsByKey[key]; ok {
			result[c.ID] = &RoomContext{
				RoomID:      ri.RoomID,
				Tags:        ri.Tags,
				MemberCount: ri.MemberCount,
			}
		}
	}

	return result
}

func (s *FeedService) hydrateReactionCounts(contents []model.Content) error {
	if len(contents) == 0 {
		return nil
	}

	contentIDs := make([]uuid.UUID, len(contents))
	for i, c := range contents {
		contentIDs[i] = c.ID
	}

	type row struct {
		ContentID uuid.UUID `gorm:"column:content_id"`
		Kind      string    `gorm:"column:kind"`
		Count     int       `gorm:"column:count"`
	}

	var rows []row
	if err := s.db.Model(&model.Reaction{}).
		Select("content_id, kind, COUNT(*) as count").
		Where("content_id IN ?", contentIDs).
		Group("content_id, kind").
		Scan(&rows).Error; err != nil {
		return fmt.Errorf("failed to load reaction counts: %w", err)
	}

	reactionMap := make(map[uuid.UUID]map[string]int)
	for _, r := range rows {
		if reactionMap[r.ContentID] == nil {
			reactionMap[r.ContentID] = make(map[string]int)
		}
		reactionMap[r.ContentID][r.Kind] = r.Count
	}

	for i := range contents {
		contents[i].Reactions = reactionMap[contents[i].ID]
	}

	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func extractContentIDs(contents []model.Content) []uuid.UUID {
	ids := make([]uuid.UUID, len(contents))
	for i, c := range contents {
		ids[i] = c.ID
	}
	return ids
}

func collectContentIDs(slices ...[]model.Content) []uuid.UUID {
	var ids []uuid.UUID
	for _, sl := range slices {
		ids = append(ids, extractContentIDs(sl)...)
	}
	return ids
}

func collectCreatorIDs(slices ...[]model.Content) []uuid.UUID {
	seen := make(map[uuid.UUID]bool)
	for _, sl := range slices {
		for _, c := range sl {
			seen[c.CreatorID] = true
		}
	}
	ids := make([]uuid.UUID, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids
}

func sortScored(items []scoredContent) {
	sort.Slice(items, func(i, j int) bool { return items[i].score > items[j].score })
}

func takeN(items []scoredContent, n int) []scoredContent {
	if n >= len(items) {
		return items
	}
	return items[:n]
}
