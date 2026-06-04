package service

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/model"
)

var ErrAdviceNotFound = errors.New("advice entity not found")

type AskInput struct {
	Question        string `json:"question"`
	Topic           string `json:"topic"`
	Urgency         string `json:"urgency"`
	DesiredHelpType string `json:"desired_help_type"`
	Visibility      string `json:"visibility"`
}

type TrustProfileInput struct {
	Topics          string `json:"topics"`
	LivedExperience string `json:"lived_experience"`
	Availability    string `json:"availability"`
}

type BridgeActionInput struct {
	Message string `json:"message"`
}

type HelpSignalInput struct {
	Kind string `json:"kind"`
}

type TodayResult struct {
	LatestAsk          *model.Ask          `json:"latest_ask,omitempty"`
	Bridges            []model.Bridge      `json:"bridges"`
	IncomingBridges    []model.Bridge      `json:"incoming_bridges"`
	PerspectiveInbox   []model.Bridge      `json:"perspective_inbox"`
	ResponseReceipts   []ResponseReceipt   `json:"response_receipts"`
	HelpSessions       []model.HelpSession `json:"help_sessions"`
	SharedRooms        []model.HelpSession `json:"shared_rooms"`
	RelationshipTrails []model.Path        `json:"relationship_trails"`
	TrustProfile       *model.TrustProfile `json:"trust_profile,omitempty"`
	StarterPrompts     []string            `json:"starter_prompts"`
}

type ResponseReceipt struct {
	BridgeID uuid.UUID  `json:"bridge_id"`
	User     model.User `json:"user"`
	Question string     `json:"question"`
	Signals  []string   `json:"signals"`
	LastAt   time.Time  `json:"last_at"`
}

type AdviceService struct {
	db       *gorm.DB
	embedder Embedder
}

func NewAdviceService(db *gorm.DB, embedder Embedder) *AdviceService {
	return &AdviceService{db: db, embedder: embedder}
}

func (s *AdviceService) CreateAsk(userID uuid.UUID, input AskInput) (*model.Ask, []model.Bridge, error) {
	question := strings.TrimSpace(input.Question)
	if len(question) < 12 {
		return nil, nil, fmt.Errorf("question must be at least 12 characters")
	}

	topic := normalizeTopic(input.Topic)
	if topic == "" {
		topic = inferTopic(question)
	}

	ask := &model.Ask{
		UserID:          userID,
		Question:        question,
		TriageSummary:   buildTriageSummary(question, topic, input.DesiredHelpType),
		Topic:           topic,
		Urgency:         normalizeChoice(input.Urgency, "soon", map[string]bool{"now": true, "soon": true, "this_week": true, "exploring": true}),
		DesiredHelpType: normalizeChoice(input.DesiredHelpType, model.HelpTypeAdvice, map[string]bool{model.HelpTypeAdvice: true, model.HelpTypePeer: true, model.HelpTypeMentor: true, model.HelpTypeFeedback: true}),
		Visibility:      normalizeChoice(input.Visibility, "community", map[string]bool{"private": true, "community": true, "public": true}),
		Embedding:       s.embeddingVector(question + " " + topic),
	}

	if err := s.db.Create(ask).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to create ask: %w", err)
	}

	bridges, err := s.GenerateBridges(ask.ID, userID, 5)
	if err != nil {
		return ask, []model.Bridge{}, err
	}

	// Implicit routing: the ask automatically reaches the strongest matches —
	// the user does not have to manually "request" each person. Reaching the
	// rest stays optional (POST /bridges/:id/ask).
	bridges = s.autoRouteBridges(bridges, 3)
	return ask, bridges, nil
}

// autoRouteBridges promotes the top-N suggested bridges (by confidence) to
// "asked" so their recommended people see the ask as a real, directed request.
func (s *AdviceService) autoRouteBridges(bridges []model.Bridge, n int) []model.Bridge {
	routed := 0
	for i := range bridges {
		if routed >= n {
			break
		}
		if bridges[i].Status != model.BridgeStatusSuggested {
			continue
		}
		if err := s.db.Model(&model.Bridge{}).
			Where("id = ?", bridges[i].ID).
			Update("status", model.BridgeStatusAsked).Error; err != nil {
			continue
		}
		bridges[i].Status = model.BridgeStatusAsked
		routed++
	}
	return bridges
}

func (s *AdviceService) GetAskBridges(askID, userID uuid.UUID) ([]model.Bridge, error) {
	var ask model.Ask
	if err := s.db.Where("id = ? AND user_id = ?", askID, userID).First(&ask).Error; err != nil {
		return nil, ErrAdviceNotFound
	}
	return s.GenerateBridges(askID, userID, 5)
}

func (s *AdviceService) GenerateBridges(askID, userID uuid.UUID, limit int) ([]model.Bridge, error) {
	if limit <= 0 {
		limit = 5
	}
	if limit > 10 {
		limit = 10
	}

	var ask model.Ask
	if err := s.db.First(&ask, "id = ? AND user_id = ?", askID, userID).Error; err != nil {
		return nil, ErrAdviceNotFound
	}

	candidates, err := s.rankBridgeCandidates(ask, userID, limit)
	if err != nil {
		return nil, err
	}

	bridges := make([]model.Bridge, 0, len(candidates))
	for _, candidate := range candidates {
		bridge := model.Bridge{
			AskID:             ask.ID,
			RequesterID:       userID,
			RecommendedUserID: candidate.user.ID,
			Reason:            candidate.reason,
			BridgeType:        candidate.bridgeType,
			Confidence:        candidate.confidence,
			Status:            model.BridgeStatusSuggested,
		}
		err := s.db.Where("ask_id = ? AND recommended_user_id = ?", ask.ID, candidate.user.ID).
			Assign(map[string]any{
				"reason":      bridge.Reason,
				"bridge_type": bridge.BridgeType,
				"confidence":  bridge.Confidence,
			}).
			FirstOrCreate(&bridge).Error
		if err != nil {
			return nil, fmt.Errorf("failed to persist bridge: %w", err)
		}
		loaded, err := s.loadBridge(bridge.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to load bridge: %w", err)
		}
		bridges = append(bridges, *loaded)
	}

	sort.Slice(bridges, func(i, j int) bool {
		return bridges[i].Confidence > bridges[j].Confidence
	})
	return bridges, nil
}

func (s *AdviceService) AskBridge(bridgeID, userID uuid.UUID, input BridgeActionInput) (*model.Bridge, error) {
	return s.updateBridgeStatus(bridgeID, userID, model.BridgeStatusAsked)
}

func (s *AdviceService) RespondBridge(bridgeID, userID uuid.UUID, input BridgeActionInput) (*model.Bridge, error) {
	var bridge model.Bridge
	if err := s.db.Where("id = ? AND recommended_user_id = ?", bridgeID, userID).First(&bridge).Error; err != nil {
		return nil, ErrAdviceNotFound
	}
	body := strings.TrimSpace(input.Message)
	if body == "" {
		body = "I can share perspective on this."
	}
	if len(body) > 1200 {
		return nil, fmt.Errorf("response must be 1200 characters or fewer")
	}

	err := s.db.Transaction(func(tx *gorm.DB) error {
		response := model.BridgeResponse{
			BridgeID:    bridge.ID,
			ResponderID: userID,
			Body:        body,
		}
		if err := tx.Where("bridge_id = ? AND responder_id = ?", bridge.ID, userID).
			Assign(map[string]any{
				"body":       body,
				"updated_at": time.Now(),
			}).
			FirstOrCreate(&response).Error; err != nil {
			return err
		}
		return tx.Model(&bridge).Updates(map[string]any{"status": model.BridgeStatusResponded}).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to record bridge response: %w", err)
	}
	return s.loadBridge(bridge.ID)
}

// AddPerspective lets anyone who has lived a public Commons ask offer their
// perspective — the non-Twitter alternative to a reply. It creates (or reuses)
// a bridge from the ask to this person and records their answer.
func (s *AdviceService) AddPerspective(askID, userID uuid.UUID, message string) (*model.Bridge, error) {
	body := strings.TrimSpace(message)
	if len(body) < 8 {
		return nil, fmt.Errorf("perspective must be at least 8 characters")
	}
	if len(body) > 1200 {
		return nil, fmt.Errorf("response must be 1200 characters or fewer")
	}

	var ask model.Ask
	if err := s.db.First(&ask, "id = ?", askID).Error; err != nil {
		return nil, ErrAdviceNotFound
	}
	if ask.Visibility != "public" {
		return nil, fmt.Errorf("this ask is not open to the Commons")
	}
	if ask.UserID == userID {
		return nil, fmt.Errorf("you cannot answer your own ask")
	}

	bridge := model.Bridge{
		AskID:             ask.ID,
		RequesterID:       ask.UserID,
		RecommendedUserID: userID,
		Reason:            "Offered perspective from lived experience in the Commons.",
		BridgeType:        model.BridgeTypeAdjacentPerspective,
		Status:            model.BridgeStatusResponded,
	}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("ask_id = ? AND recommended_user_id = ?", ask.ID, userID).
			Assign(map[string]any{
				"requester_id": ask.UserID,
				"reason":       bridge.Reason,
				"bridge_type":  bridge.BridgeType,
				"status":       model.BridgeStatusResponded,
			}).
			FirstOrCreate(&bridge).Error; err != nil {
			return err
		}
		response := model.BridgeResponse{BridgeID: bridge.ID, ResponderID: userID, Body: body}
		return tx.Where("bridge_id = ? AND responder_id = ?", bridge.ID, userID).
			Assign(map[string]any{"body": body, "updated_at": time.Now()}).
			FirstOrCreate(&response).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add perspective: %w", err)
	}
	return s.loadBridge(bridge.ID)
}

func (s *AdviceService) RecordSignal(bridgeID, userID uuid.UUID, input HelpSignalInput) (*model.HelpSignal, error) {
	kind := normalizeChoice(input.Kind, "", map[string]bool{
		"useful": true, "clarifying": true, "motivating": true, "practical": true, "not_relevant": true,
	})
	if kind == "" {
		return nil, fmt.Errorf("unsupported signal kind")
	}

	var bridge model.Bridge
	if err := s.db.Where("id = ? AND (requester_id = ? OR recommended_user_id = ?)", bridgeID, userID, userID).First(&bridge).Error; err != nil {
		return nil, ErrAdviceNotFound
	}

	signal := &model.HelpSignal{BridgeID: bridge.ID, UserID: userID, Kind: kind}
	err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("bridge_id = ? AND user_id = ? AND kind = ?", bridge.ID, userID, kind).
			FirstOrCreate(signal).Error; err != nil {
			return err
		}

		if kind == "not_relevant" {
			return tx.Model(&bridge).Update("status", model.BridgeStatusDismissed).Error
		}

		return tx.Exec(`
			INSERT INTO trust_profiles (user_id, helped_count, response_quality, created_at, updated_at)
			VALUES (?, 1, 1, now(), now())
			ON CONFLICT (user_id) DO UPDATE SET
				helped_count = trust_profiles.helped_count + 1,
				response_quality = LEAST(5, trust_profiles.response_quality + 0.2),
				updated_at = now()
		`, bridge.RecommendedUserID).Error
	})
	if err != nil {
		return nil, fmt.Errorf("failed to record help signal: %w", err)
	}
	return signal, nil
}

func (s *AdviceService) UpdateTrustProfile(userID uuid.UUID, input TrustProfileInput) (*model.TrustProfile, error) {
	topics := strings.TrimSpace(input.Topics)
	experience := strings.TrimSpace(input.LivedExperience)
	availability := normalizeChoice(input.Availability, "async", map[string]bool{"async": true, "live_now": true, "bookable_10m": true})
	vec := s.embeddingVector(topics + " " + experience)

	profile := &model.TrustProfile{
		UserID:          userID,
		Topics:          topics,
		LivedExperience: experience,
		Availability:    availability,
		ExpertiseVector: vec,
	}
	if err := s.db.Where("user_id = ?", userID).
		Assign(map[string]any{
			"topics":           topics,
			"lived_experience": experience,
			"availability":     availability,
			"expertise_vector": vec,
			"updated_at":       time.Now(),
		}).
		FirstOrCreate(profile).Error; err != nil {
		return nil, fmt.Errorf("failed to update trust profile: %w", err)
	}
	return s.GetTrustProfile(userID)
}

func (s *AdviceService) GetTrustProfile(userID uuid.UUID) (*model.TrustProfile, error) {
	var profile model.TrustProfile
	if err := s.db.Preload("User").First(&profile, "user_id = ?", userID).Error; err != nil {
		return nil, err
	}
	return &profile, nil
}

func (s *AdviceService) GetToday(userID uuid.UUID) (*TodayResult, error) {
	result := &TodayResult{
		Bridges:            []model.Bridge{},
		IncomingBridges:    []model.Bridge{},
		PerspectiveInbox:   []model.Bridge{},
		ResponseReceipts:   []ResponseReceipt{},
		HelpSessions:       []model.HelpSession{},
		SharedRooms:        []model.HelpSession{},
		RelationshipTrails: []model.Path{},
		StarterPrompts: []string{
			"I'm stuck getting my first 10 users.",
			"I need a peer to review my launch plan.",
			"I want advice from someone a few steps ahead.",
		},
	}

	var ask model.Ask
	if err := s.db.Where("user_id = ?", userID).Order("created_at DESC").First(&ask).Error; err == nil {
		result.LatestAsk = &ask
		bridges, _ := s.GetAskBridges(ask.ID, userID)
		result.Bridges = bridges
	}

	incoming, _ := s.GetIncomingBridges(userID, 8)
	result.IncomingBridges = incoming
	result.PerspectiveInbox = unansweredBridges(incoming)
	result.ResponseReceipts = s.responseReceipts(userID, 6)
	if sessions, err := s.ListHelpSessions(userID); err == nil {
		result.HelpSessions = sessions
		result.SharedRooms = takeHelpSessions(sessions, 4)
	}
	result.RelationshipTrails = s.relationshipTrails(userID, 4)

	if profile, err := s.GetTrustProfile(userID); err == nil {
		result.TrustProfile = profile
	}

	return result, nil
}

func (s *AdviceService) GetIncomingBridges(userID uuid.UUID, limit int) ([]model.Bridge, error) {
	if limit <= 0 {
		limit = 8
	}
	if limit > 20 {
		limit = 20
	}

	// Only asks actually routed to this person ("asked") or already handled
	// ("responded") — never raw "suggested" candidates nobody sent them.
	var bridges []model.Bridge
	err := s.db.Preload("Ask.User").
		Preload("RecommendedUser").
		Preload("Responses.Responder").
		Where("recommended_user_id = ? AND status IN ?", userID,
			[]string{model.BridgeStatusAsked, model.BridgeStatusResponded}).
		Order("created_at DESC").
		Limit(limit).
		Find(&bridges).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load incoming bridges: %w", err)
	}
	return bridges, nil
}

// CommonsEntry is one answered ask published to the Commons: the question plus
// the perspectives people offered. When the ask is anonymous the asker identity
// is stripped before it leaves the service.
type CommonsEntry struct {
	Ask       model.Ask              `json:"ask"`
	Responses []model.BridgeResponse `json:"responses"`
}

// ListCommons returns public asks newest first. Answered asks become durable
// lived perspective; unanswered asks remain visible so Commons never depends on
// a private manual publish loop before the community can help.
func (s *AdviceService) ListCommons(userID uuid.UUID, cursor string, limit int) ([]CommonsEntry, string, bool, error) {
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
	offset := page * limit

	var asks []model.Ask
	if err := s.db.Preload("User").
		Where("visibility = ?", "public").
		Order("created_at DESC").
		Limit(limit + 1).
		Offset(offset).
		Find(&asks).Error; err != nil {
		return nil, "", false, fmt.Errorf("failed to load commons: %w", err)
	}

	hasMore := len(asks) > limit
	if hasMore {
		asks = asks[:limit]
	}
	if len(asks) == 0 {
		return []CommonsEntry{}, "", false, nil
	}

	askIDs := make([]uuid.UUID, len(asks))
	for i, a := range asks {
		askIDs[i] = a.ID
	}

	var bridges []model.Bridge
	if err := s.db.Preload("Responses.Responder").
		Where("ask_id IN ?", askIDs).
		Find(&bridges).Error; err != nil {
		return nil, "", false, fmt.Errorf("failed to load commons responses: %w", err)
	}
	responsesByAsk := make(map[uuid.UUID][]model.BridgeResponse)
	for _, b := range bridges {
		responsesByAsk[b.AskID] = append(responsesByAsk[b.AskID], b.Responses...)
	}

	entries := make([]CommonsEntry, 0, len(asks))
	for _, ask := range asks {
		responses := responsesByAsk[ask.ID]
		if ask.Anonymous {
			// Strip the asker's identity before it leaves the service.
			ask.User = model.User{}
			ask.UserID = uuid.Nil
		}
		ask.Embedding = nil
		entries = append(entries, CommonsEntry{Ask: ask, Responses: responses})
	}

	var nextCursor string
	if hasMore {
		nextCursor = encodeScoredCursor(page + 1)
	}
	return entries, nextCursor, hasMore, nil
}

type AskVisibilityInput struct {
	Visibility string `json:"visibility"`
	Anonymous  bool   `json:"anonymous"`
}

// UpdateAskVisibility lets the asker change who can see an ask — in particular
// publishing an answered ask to the Commons, optionally anonymously.
func (s *AdviceService) UpdateAskVisibility(askID, userID uuid.UUID, input AskVisibilityInput) (*model.Ask, error) {
	visibility := normalizeChoice(input.Visibility, "community", map[string]bool{"private": true, "community": true, "public": true})

	var ask model.Ask
	if err := s.db.Where("id = ? AND user_id = ?", askID, userID).First(&ask).Error; err != nil {
		return nil, ErrAdviceNotFound
	}
	if err := s.db.Model(&ask).Updates(map[string]any{
		"visibility": visibility,
		"anonymous":  input.Anonymous,
		"updated_at": time.Now(),
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to update ask visibility: %w", err)
	}
	ask.Visibility = visibility
	ask.Anonymous = input.Anonymous
	ask.Embedding = nil
	return &ask, nil
}

// NetworkConnection is one person you've actually exchanged perspective with —
// either they answered your ask, or you answered theirs.
type NetworkConnection struct {
	User        model.User   `json:"user"`
	Direction   string       `json:"direction"` // "you_asked" | "you_answered" | "connected" | "nearby"
	Topic       string       `json:"topic"`
	Question    string       `json:"question"`
	Where       string       `json:"where"`
	ContextTags []string     `json:"context_tags"`
	Affinity    float64      `json:"affinity"`
	ActiveRoom  *RoomContext `json:"active_room,omitempty"`
	SharedPath  *model.Path  `json:"shared_path,omitempty"`
	LastAt      time.Time    `json:"last_at"`
}

// GetNetwork returns the people you've connected with through real exchanges,
// explicit follows, and implicit affinity. No vanity metrics — only context for
// who is close and why they are easy to re-enter.
func (s *AdviceService) GetNetwork(userID uuid.UUID) ([]NetworkConnection, error) {
	connections := make([]NetworkConnection, 0)
	seen := make(map[uuid.UUID]bool)

	// People who answered your asks.
	var answered []model.Bridge
	if err := s.db.Preload("RecommendedUser").Preload("Ask").
		Where("requester_id = ? AND status = ?", userID, model.BridgeStatusResponded).
		Order("updated_at DESC").
		Find(&answered).Error; err != nil {
		return nil, fmt.Errorf("failed to load answered bridges: %w", err)
	}
	for _, b := range answered {
		if b.RecommendedUserID == userID || seen[b.RecommendedUserID] {
			continue
		}
		seen[b.RecommendedUserID] = true
		where, tags, room, path, affinity := s.networkContext(userID, b.RecommendedUserID)
		connections = append(connections, NetworkConnection{
			User:        b.RecommendedUser,
			Direction:   "you_asked",
			Topic:       b.Ask.Topic,
			Question:    b.Ask.Question,
			Where:       where,
			ContextTags: tags,
			Affinity:    affinity,
			ActiveRoom:  room,
			SharedPath:  path,
			LastAt:      b.UpdatedAt,
		})
	}

	// People whose asks you answered.
	var answeredByMe []model.Bridge
	if err := s.db.Preload("Ask.User").
		Where("id IN (SELECT bridge_id FROM bridge_responses WHERE responder_id = ?)", userID).
		Order("updated_at DESC").
		Find(&answeredByMe).Error; err != nil {
		return nil, fmt.Errorf("failed to load given responses: %w", err)
	}
	for _, b := range answeredByMe {
		asker := b.Ask.User
		if asker.ID == uuid.Nil || asker.ID == userID || seen[asker.ID] {
			continue
		}
		seen[asker.ID] = true
		where, tags, room, path, affinity := s.networkContext(userID, asker.ID)
		connections = append(connections, NetworkConnection{
			User:        asker,
			Direction:   "you_answered",
			Topic:       b.Ask.Topic,
			Question:    b.Ask.Question,
			Where:       where,
			ContextTags: tags,
			Affinity:    affinity,
			ActiveRoom:  room,
			SharedPath:  path,
			LastAt:      b.UpdatedAt,
		})
	}

	// People you intentionally connected with. They should not disappear just
	// because they have not answered an ask yet.
	var follows []model.Follow
	if err := s.db.Preload("Followee").
		Where("follower_id = ?", userID).
		Order("created_at DESC").
		Limit(20).
		Find(&follows).Error; err != nil {
		return nil, fmt.Errorf("failed to load followed people: %w", err)
	}
	for _, follow := range follows {
		if follow.FolloweeID == userID || seen[follow.FolloweeID] {
			continue
		}
		seen[follow.FolloweeID] = true
		where, tags, room, path, affinity := s.networkContext(userID, follow.FolloweeID)
		connections = append(connections, NetworkConnection{
			User:        follow.Followee,
			Direction:   "connected",
			Topic:       "connected",
			Question:    where,
			Where:       where,
			ContextTags: tags,
			Affinity:    affinity,
			ActiveRoom:  room,
			SharedPath:  path,
			LastAt:      follow.CreatedAt,
		})
	}

	// High-affinity people you may not have followed or helped yet. This is the
	// implicit graph surfacing itself as a path, not a follower count.
	var edges []model.UserAffinityEdge
	if err := s.db.Where("user_id = ? AND other_user_id <> ?", userID, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocker_id = ? AND blocked_id = user_affinity_edges.other_user_id
		)`, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocked_id = ? AND blocker_id = user_affinity_edges.other_user_id
		)`, userID).
		Order("score_7d DESC, last_signal_at DESC").
		Limit(12).
		Find(&edges).Error; err != nil {
		return nil, fmt.Errorf("failed to load affinity edges: %w", err)
	}
	for _, edge := range edges {
		if edge.OtherUserID == userID || seen[edge.OtherUserID] {
			continue
		}
		var user model.User
		if err := s.db.First(&user, "id = ?", edge.OtherUserID).Error; err != nil {
			continue
		}
		seen[edge.OtherUserID] = true
		where, tags, room, path, affinity := s.networkContext(userID, edge.OtherUserID)
		if affinity == 0 {
			affinity = edge.Score7D
		}
		connections = append(connections, NetworkConnection{
			User:        user,
			Direction:   "nearby",
			Topic:       "nearby",
			Question:    where,
			Where:       where,
			ContextTags: tags,
			Affinity:    affinity,
			ActiveRoom:  room,
			SharedPath:  path,
			LastAt:      edge.LastSignalAt,
		})
	}

	sort.Slice(connections, func(i, j int) bool {
		return connections[i].LastAt.After(connections[j].LastAt)
	})
	return connections, nil
}

func (s *AdviceService) latestUserContext(userID uuid.UUID) string {
	var content model.Content
	if err := s.db.Preload("Tags").
		Where("creator_id = ?", userID).
		Order("created_at DESC").
		First(&content).Error; err != nil {
		return "Connected through your graph"
	}
	tagNames := make([]string, 0, len(content.Tags))
	for _, tag := range content.Tags {
		tagNames = append(tagNames, "#"+tag.Name)
	}
	if len(tagNames) > 0 {
		return "Recently around " + strings.Join(tagNames, " ")
	}
	if strings.TrimSpace(content.Body) != "" {
		return content.Body
	}
	return "Recent moment in your graph"
}

func unansweredBridges(bridges []model.Bridge) []model.Bridge {
	out := make([]model.Bridge, 0, len(bridges))
	for _, bridge := range bridges {
		if len(bridge.Responses) == 0 {
			out = append(out, bridge)
		}
	}
	return out
}

func takeHelpSessions(sessions []model.HelpSession, limit int) []model.HelpSession {
	if limit <= 0 || len(sessions) <= limit {
		return sessions
	}
	return sessions[:limit]
}

func (s *AdviceService) responseReceipts(userID uuid.UUID, limit int) []ResponseReceipt {
	if limit <= 0 {
		limit = 6
	}

	type row struct {
		BridgeID uuid.UUID `gorm:"column:bridge_id"`
		Signal   string    `gorm:"column:signal"`
		LastAt   time.Time `gorm:"column:last_at"`
	}
	var rows []row
	if err := s.db.Raw(`
		SELECT hs.bridge_id, hs.kind AS signal, hs.created_at AS last_at
		FROM help_signals hs
		JOIN bridges b ON b.id = hs.bridge_id
		WHERE b.recommended_user_id = ? AND hs.kind <> 'not_relevant'
		ORDER BY hs.created_at DESC
		LIMIT ?
	`, userID, limit*3).Scan(&rows).Error; err != nil || len(rows) == 0 {
		return []ResponseReceipt{}
	}

	bridgeIDs := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		bridgeIDs = append(bridgeIDs, row.BridgeID)
	}

	var bridges []model.Bridge
	if err := s.db.Preload("Ask.User").
		Where("id IN ?", bridgeIDs).
		Find(&bridges).Error; err != nil {
		return []ResponseReceipt{}
	}

	byID := make(map[uuid.UUID]model.Bridge, len(bridges))
	for _, bridge := range bridges {
		byID[bridge.ID] = bridge
	}

	out := make([]ResponseReceipt, 0, limit)
	seen := map[uuid.UUID]int{}
	for _, row := range rows {
		bridge, ok := byID[row.BridgeID]
		if !ok {
			continue
		}
		if idx, exists := seen[row.BridgeID]; exists {
			out[idx].Signals = appendUniqueString(out[idx].Signals, row.Signal)
			if row.LastAt.After(out[idx].LastAt) {
				out[idx].LastAt = row.LastAt
			}
			continue
		}
		seen[row.BridgeID] = len(out)
		out = append(out, ResponseReceipt{
			BridgeID: bridge.ID,
			User:     bridge.Ask.User,
			Question: bridge.Ask.Question,
			Signals:  []string{row.Signal},
			LastAt:   row.LastAt,
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func (s *AdviceService) relationshipTrails(userID uuid.UUID, limit int) []model.Path {
	if limit <= 0 {
		limit = 4
	}
	var paths []model.Path
	err := s.db.Preload("Creator").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Preload("Items.Content.Creator").
		Preload("Items.Content.Tags").
		Where(`
			id IN (
				SELECT path_id FROM path_follows WHERE user_id = ?
				UNION
				SELECT pf.path_id FROM path_follows pf
				JOIN follows f ON f.followee_id = pf.user_id
				WHERE f.follower_id = ?
				UNION
				SELECT id FROM paths
				WHERE creator_id IN (SELECT followee_id FROM follows WHERE follower_id = ?)
			)
		`, userID, userID, userID).
		Order("created_at DESC").
		Limit(limit).
		Find(&paths).Error
	if err != nil {
		return []model.Path{}
	}
	return paths
}

func (s *AdviceService) networkContext(userID, otherID uuid.UUID) (string, []string, *RoomContext, *model.Path, float64) {
	tags := s.latestUserTags(otherID, 4)
	where := s.latestUserContext(otherID)
	room := s.activeRoomForUser(otherID)
	path := s.sharedPathForUsers(userID, otherID)
	affinity := s.affinityScore(userID, otherID)

	if room != nil && len(room.Tags) > 0 {
		where = "Live in a shared context around " + formatHashTags(room.Tags)
	} else if path != nil {
		where = "On the relationship trail " + path.Title
	}
	return where, tags, room, path, affinity
}

func (s *AdviceService) latestUserTags(userID uuid.UUID, limit int) []string {
	if limit <= 0 {
		limit = 4
	}
	type row struct {
		Name string `gorm:"column:name"`
	}
	var rows []row
	if err := s.db.Raw(`
		SELECT t.name
		FROM tags t
		JOIN content_tags ct ON ct.tag_id = t.id
		JOIN contents c ON c.id = ct.content_id
		WHERE c.creator_id = ?
		GROUP BY t.id, t.name
		ORDER BY MAX(c.created_at) DESC
		LIMIT ?
	`, userID, limit).Scan(&rows).Error; err != nil {
		return []string{}
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Name != "" {
			out = append(out, row.Name)
		}
	}
	return out
}

func (s *AdviceService) activeRoomForUser(userID uuid.UUID) *RoomContext {
	var room model.Room
	if err := s.db.Preload("Tags").
		Where("expires_at > ?", time.Now()).
		Where("id IN (SELECT room_id FROM room_members WHERE user_id = ?)", userID).
		Order("created_at DESC").
		First(&room).Error; err != nil {
		return nil
	}

	var memberCount int64
	s.db.Model(&model.RoomMember{}).Where("room_id = ?", room.ID).Count(&memberCount)
	tags := make([]string, 0, len(room.Tags))
	for _, tag := range room.Tags {
		tags = append(tags, tag.Name)
	}
	return &RoomContext{RoomID: room.ID, Tags: tags, MemberCount: memberCount}
}

func (s *AdviceService) sharedPathForUsers(userID, otherID uuid.UUID) *model.Path {
	var path model.Path
	err := s.db.Preload("Creator").
		Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("position ASC") }).
		Preload("Items.Content.Creator").
		Preload("Items.Content.Tags").
		Where(`
			id IN (
				SELECT mine.path_id
				FROM path_follows mine
				JOIN path_follows theirs ON theirs.path_id = mine.path_id
				WHERE mine.user_id = ? AND theirs.user_id = ?
				UNION
				SELECT id FROM paths
				WHERE creator_id = ?
				AND id IN (SELECT path_id FROM path_follows WHERE user_id = ?)
			)
		`, userID, otherID, otherID, userID).
		Order("created_at DESC").
		First(&path).Error
	if err != nil {
		return nil
	}
	return &path
}

func (s *AdviceService) affinityScore(userID, otherID uuid.UUID) float64 {
	var edge model.UserAffinityEdge
	if err := s.db.First(&edge, "user_id = ? AND other_user_id = ?", userID, otherID).Error; err != nil {
		return 0
	}
	return edge.Score7D
}

func appendUniqueString(items []string, value string) []string {
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func (s *AdviceService) ListHelpSessions(userID uuid.UUID) ([]model.HelpSession, error) {
	if err := s.ensureDefaultHelpSessions(); err != nil {
		return nil, err
	}

	type row struct {
		model.HelpSession
		MemberCount int64 `gorm:"column:member_count"`
	}
	var rows []row
	if err := s.db.Table("help_sessions").
		Select("help_sessions.*, (SELECT COUNT(*) FROM help_session_members WHERE session_id = help_sessions.id) AS member_count").
		Where("expires_at > ?", time.Now()).
		Order("created_at DESC").
		Limit(12).
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to list help sessions: %w", err)
	}

	out := make([]model.HelpSession, 0, len(rows))
	for _, row := range rows {
		session := row.HelpSession
		session.MemberCount = row.MemberCount
		out = append(out, session)
	}
	return out, nil
}

func (s *AdviceService) JoinHelpSession(sessionID, userID uuid.UUID) (*model.HelpSession, error) {
	var session model.HelpSession
	if err := s.db.Where("id = ? AND expires_at > ?", sessionID, time.Now()).First(&session).Error; err != nil {
		return nil, ErrAdviceNotFound
	}
	member := model.HelpSessionMember{SessionID: session.ID, UserID: userID, JoinedAt: time.Now()}
	if err := s.db.Where("session_id = ? AND user_id = ?", session.ID, userID).FirstOrCreate(&member).Error; err != nil {
		return nil, fmt.Errorf("failed to join help session: %w", err)
	}
	sessions, err := s.ListHelpSessions(userID)
	if err != nil {
		return nil, err
	}
	for _, item := range sessions {
		if item.ID == session.ID {
			return &item, nil
		}
	}
	return &session, nil
}

type bridgeCandidate struct {
	user           model.User
	reason         string
	bridgeType     string
	confidence     float64
	relevance      float64
	recentExposure int64
	repeatedPair   bool
}

type bridgeExposure struct {
	recentCount    int64
	requesterCount int64
}

func (s *AdviceService) rankBridgeCandidates(ask model.Ask, userID uuid.UUID, limit int) ([]bridgeCandidate, error) {
	var profiles []model.TrustProfile
	err := s.db.Preload("User").
		Where("user_id <> ?", userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocker_id = ? AND blocked_id = trust_profiles.user_id
		)`, userID).
		Where(`NOT EXISTS (
			SELECT 1 FROM blocks WHERE blocked_id = ? AND blocker_id = trust_profiles.user_id
		)`, userID).
		Find(&profiles).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load trust profiles: %w", err)
	}

	profileIDs := make([]uuid.UUID, 0, len(profiles))
	for _, profile := range profiles {
		profileIDs = append(profileIDs, profile.UserID)
	}
	exposure := s.loadBridgeExposure(userID, profileIDs)

	askVector := parseVector(ask.Embedding)
	candidates := make([]bridgeCandidate, 0, len(profiles))
	for _, profile := range profiles {
		textScore := overlapScore(ask.Topic+" "+ask.Question, profile.Topics+" "+profile.LivedExperience)
		vectorScore := cosine(askVector, parseVector(profile.ExpertiseVector))
		quality := math.Min((float64(profile.HelpedCount)*0.04)+(profile.ResponseQuality*0.12), 0.35)
		availability := availabilityBoost(profile.Availability)
		relevance := clampFloat(0.45*textScore+0.35*vectorScore+quality+availability, 0, 1)
		if relevance <= 0 {
			relevance = 0.12 + quality + availability
		}
		stats := exposure[profile.UserID]
		exposurePenalty := math.Min(float64(stats.recentCount)*0.08, 0.32)
		if stats.requesterCount > 0 {
			exposurePenalty += 0.16
		}
		score := clampFloat(relevance-exposurePenalty, 0.08, 1)
		bridgeType := bridgeTypeForAsk(ask.DesiredHelpType, relevance, profile)
		candidates = append(candidates, bridgeCandidate{
			user:           profile.User,
			reason:         buildAdviceBridgeReason(ask, profile, bridgeType),
			bridgeType:     bridgeType,
			confidence:     roundConfidence(score),
			relevance:      roundConfidence(relevance),
			recentExposure: stats.recentCount,
			repeatedPair:   stats.requesterCount > 0,
		})
	}

	if len(candidates) == 0 {
		cold, err := s.coldStartCandidates(ask, userID, limit)
		if err != nil {
			return nil, err
		}
		return cold, nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].confidence > candidates[j].confidence
	})
	return diversifyBridgeCandidates(candidates, limit), nil
}

func (s *AdviceService) loadBridgeExposure(requesterID uuid.UUID, userIDs []uuid.UUID) map[uuid.UUID]bridgeExposure {
	result := make(map[uuid.UUID]bridgeExposure, len(userIDs))
	if len(userIDs) == 0 {
		return result
	}

	type row struct {
		RecommendedUserID uuid.UUID `gorm:"column:recommended_user_id"`
		RecentCount       int64     `gorm:"column:recent_count"`
		RequesterCount    int64     `gorm:"column:requester_count"`
	}
	var rows []row
	err := s.db.Raw(`
		SELECT
			recommended_user_id,
			COUNT(*) FILTER (WHERE created_at > now() - interval '48 hours') AS recent_count,
			COUNT(*) FILTER (WHERE requester_id = ?) AS requester_count
		FROM bridges
		WHERE recommended_user_id IN ?
		GROUP BY recommended_user_id
	`, requesterID, userIDs).Scan(&rows).Error
	if err != nil {
		return result
	}
	for _, row := range rows {
		result[row.RecommendedUserID] = bridgeExposure{
			recentCount:    row.RecentCount,
			requesterCount: row.RequesterCount,
		}
	}
	return result
}

func diversifyBridgeCandidates(candidates []bridgeCandidate, limit int) []bridgeCandidate {
	if limit <= 0 || len(candidates) <= limit {
		return candidates
	}

	selected := make([]bridgeCandidate, 0, limit)
	used := make(map[uuid.UUID]bool, limit)
	anchorCount := limit - 2
	if anchorCount < 1 {
		anchorCount = 1
	}
	if anchorCount > limit {
		anchorCount = limit
	}

	for _, candidate := range candidates {
		if len(selected) >= anchorCount {
			break
		}
		selected = append(selected, candidate)
		used[candidate.user.ID] = true
	}

	exploration := make([]bridgeCandidate, 0, len(candidates))
	for _, candidate := range candidates {
		if !used[candidate.user.ID] && candidate.relevance >= 0.18 {
			exploration = append(exploration, candidate)
		}
	}
	sort.Slice(exploration, func(i, j int) bool {
		if exploration[i].repeatedPair != exploration[j].repeatedPair {
			return !exploration[i].repeatedPair
		}
		if exploration[i].recentExposure != exploration[j].recentExposure {
			return exploration[i].recentExposure < exploration[j].recentExposure
		}
		return exploration[i].relevance > exploration[j].relevance
	})

	for _, candidate := range exploration {
		if len(selected) >= limit {
			break
		}
		selected = append(selected, candidate)
		used[candidate.user.ID] = true
	}
	for _, candidate := range candidates {
		if len(selected) >= limit {
			break
		}
		if !used[candidate.user.ID] {
			selected = append(selected, candidate)
		}
	}
	return selected
}

func (s *AdviceService) coldStartCandidates(ask model.Ask, userID uuid.UUID, limit int) ([]bridgeCandidate, error) {
	var users []model.User
	if err := s.db.Where("id <> ?", userID).Order("created_at DESC").Limit(limit).Find(&users).Error; err != nil {
		return nil, err
	}
	out := make([]bridgeCandidate, 0, len(users))
	for i, user := range users {
		bridgeType := model.BridgeTypePeer
		if i == 0 && ask.DesiredHelpType == model.HelpTypeMentor {
			bridgeType = model.BridgeTypeMentor
		}
		out = append(out, bridgeCandidate{
			user:       user,
			reason:     "They are active in the community and can help calibrate your question while Pulse learns your advice graph.",
			bridgeType: bridgeType,
			confidence: 0.2,
		})
	}
	return out, nil
}

func (s *AdviceService) updateBridgeStatus(bridgeID, userID uuid.UUID, status string) (*model.Bridge, error) {
	var bridge model.Bridge
	if err := s.db.Where("id = ? AND requester_id = ?", bridgeID, userID).First(&bridge).Error; err != nil {
		return nil, ErrAdviceNotFound
	}
	if err := s.db.Model(&bridge).Update("status", status).Error; err != nil {
		return nil, fmt.Errorf("failed to update bridge: %w", err)
	}
	return s.loadBridge(bridge.ID)
}

func (s *AdviceService) loadBridge(id uuid.UUID) (*model.Bridge, error) {
	var bridge model.Bridge
	if err := s.db.Preload("Ask.User").
		Preload("RecommendedUser").
		Preload("Responses.Responder").
		First(&bridge, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &bridge, nil
}

func (s *AdviceService) ensureDefaultHelpSessions() error {
	var count int64
	if err := s.db.Model(&model.HelpSession{}).Where("expires_at > ?", time.Now()).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	sessions := []model.HelpSession{
		{Title: "First 10 customers", Intent: "go_to_market", Description: "Fast feedback for builders trying to get early users.", ExpiresAt: time.Now().Add(6 * time.Hour)},
		{Title: "Portfolio review", Intent: "creative_feedback", Description: "Low-pressure critique from peers and mentors.", ExpiresAt: time.Now().Add(6 * time.Hour)},
		{Title: "Builder accountability", Intent: "accountability", Description: "Find someone shipping through the same messy middle.", ExpiresAt: time.Now().Add(6 * time.Hour)},
	}
	return s.db.Create(&sessions).Error
}

func (s *AdviceService) embeddingVector(text string) *string {
	if s.embedder == nil || strings.TrimSpace(text) == "" {
		return nil
	}
	vec, err := s.embedder.EmbedText(text)
	if err != nil {
		return nil
	}
	formatted := formatVector(vec)
	if formatted == "" {
		return nil
	}
	return &formatted
}

func buildTriageSummary(question, topic, helpType string) string {
	kind := strings.TrimSpace(helpType)
	if kind == "" {
		kind = model.HelpTypeAdvice
	}
	if topic == "" {
		return "Looking for " + kind + " from someone with relevant lived context."
	}
	return fmt.Sprintf("Looking for %s around %s from someone with relevant lived context.", kind, topic)
}

func buildAdviceBridgeReason(ask model.Ask, profile model.TrustProfile, bridgeType string) string {
	topic := ask.Topic
	if topic == "" {
		topic = "this problem"
	}
	switch bridgeType {
	case model.BridgeTypeMentor:
		if strings.TrimSpace(profile.LivedExperience) != "" {
			return fmt.Sprintf("They have lived experience related to %s and can give you practical perspective.", topic)
		}
		return fmt.Sprintf("They look a few steps ahead on %s and are available for advice.", topic)
	case model.BridgeTypePeer:
		return fmt.Sprintf("They are close enough to your context on %s to compare notes without a heavy mentor dynamic.", topic)
	default:
		return fmt.Sprintf("They bring an adjacent perspective on %s that can break your current frame.", topic)
	}
}

func bridgeTypeForAsk(helpType string, score float64, profile model.TrustProfile) string {
	switch helpType {
	case model.HelpTypeMentor:
		return model.BridgeTypeMentor
	case model.HelpTypePeer:
		return model.BridgeTypePeer
	case model.HelpTypeFeedback:
		return model.BridgeTypeAdjacentPerspective
	}
	if profile.HelpedCount > 2 || score > 0.72 {
		return model.BridgeTypeMentor
	}
	if score > 0.38 {
		return model.BridgeTypePeer
	}
	return model.BridgeTypeAdjacentPerspective
}

func availabilityBoost(availability string) float64 {
	switch availability {
	case "live_now":
		return 0.12
	case "bookable_10m":
		return 0.08
	default:
		return 0.03
	}
}

func normalizeChoice(raw, fallback string, allowed map[string]bool) string {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if allowed[normalized] {
		return normalized
	}
	return fallback
}

func normalizeTopic(raw string) string {
	topic := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(raw, "#")))
	return strings.Join(strings.Fields(topic), " ")
}

func inferTopic(question string) string {
	words := strings.Fields(strings.ToLower(question))
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:\"'()[]{}")
		if strings.HasPrefix(word, "#") && len(word) > 2 {
			return strings.TrimPrefix(word, "#")
		}
	}
	keywords := []string{"customers", "launch", "fundraising", "design", "portfolio", "pricing", "career", "lonely", "users", "mentor"}
	for _, keyword := range keywords {
		if strings.Contains(strings.ToLower(question), keyword) {
			return keyword
		}
	}
	for _, word := range words {
		word = strings.Trim(word, ".,!?;:\"'()[]{}")
		if len(word) > 5 {
			return word
		}
	}
	return "general"
}

func overlapScore(a, b string) float64 {
	left := tokenSet(a)
	right := tokenSet(b)
	if len(left) == 0 || len(right) == 0 {
		return 0
	}
	var shared float64
	for token := range left {
		if right[token] {
			shared++
		}
	}
	return math.Min(shared/math.Sqrt(float64(len(left)*len(right))), 1)
}

func tokenSet(text string) map[string]bool {
	out := map[string]bool{}
	for _, token := range strings.Fields(strings.ToLower(text)) {
		token = strings.Trim(token, ".,!?;:\"'()[]{}#")
		if len(token) > 2 {
			out[token] = true
		}
	}
	return out
}

func formatVector(vec []float64) string {
	if len(vec) == 0 {
		return ""
	}
	parts := make([]string, len(vec))
	for i, v := range vec {
		parts[i] = fmt.Sprintf("%.6f", v)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func parseVector(raw *string) []float64 {
	if raw == nil {
		return nil
	}
	trimmed := strings.TrimSpace(strings.Trim(*raw, "[]"))
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	out := make([]float64, 0, len(parts))
	for _, part := range parts {
		var value float64
		if _, err := fmt.Sscanf(strings.TrimSpace(part), "%f", &value); err == nil {
			out = append(out, value)
		}
	}
	return out
}

func cosine(a, b []float64) float64 {
	if len(a) == 0 || len(b) == 0 || len(a) != len(b) {
		return 0
	}
	var dot, an, bn float64
	for i := range a {
		dot += a[i] * b[i]
		an += a[i] * a[i]
		bn += b[i] * b[i]
	}
	if an == 0 || bn == 0 {
		return 0
	}
	return clampFloat((dot/math.Sqrt(an*bn)+1)/2, 0, 1)
}

func roundConfidence(v float64) float64 {
	return math.Round(clampFloat(v, 0, 1)*100) / 100
}

func clampFloat(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}
