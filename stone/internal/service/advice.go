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
	LatestAsk       *model.Ask          `json:"latest_ask,omitempty"`
	Bridges         []model.Bridge      `json:"bridges"`
	IncomingBridges []model.Bridge      `json:"incoming_bridges"`
	HelpSessions    []model.HelpSession `json:"help_sessions"`
	TrustProfile    *model.TrustProfile `json:"trust_profile,omitempty"`
	StarterPrompts  []string            `json:"starter_prompts"`
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
	return ask, bridges, nil
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
		bridge.RecommendedUser = candidate.user
		bridges = append(bridges, bridge)
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
	if err := s.db.Model(&bridge).Updates(map[string]any{"status": model.BridgeStatusResponded}).Error; err != nil {
		return nil, fmt.Errorf("failed to mark bridge responded: %w", err)
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
		Bridges:         []model.Bridge{},
		IncomingBridges: []model.Bridge{},
		HelpSessions:    []model.HelpSession{},
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

	sessions, _ := s.ListHelpSessions(userID)
	result.HelpSessions = sessions

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

	var bridges []model.Bridge
	err := s.db.Preload("Ask.User").
		Preload("RecommendedUser").
		Where("recommended_user_id = ? AND status <> ?", userID, model.BridgeStatusDismissed).
		Order("created_at DESC").
		Limit(limit).
		Find(&bridges).Error
	if err != nil {
		return nil, fmt.Errorf("failed to load incoming bridges: %w", err)
	}
	return bridges, nil
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
	user       model.User
	reason     string
	bridgeType string
	confidence float64
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

	askVector := parseVector(ask.Embedding)
	candidates := make([]bridgeCandidate, 0, len(profiles))
	for _, profile := range profiles {
		textScore := overlapScore(ask.Topic+" "+ask.Question, profile.Topics+" "+profile.LivedExperience)
		vectorScore := cosine(askVector, parseVector(profile.ExpertiseVector))
		quality := math.Min((float64(profile.HelpedCount)*0.04)+(profile.ResponseQuality*0.12), 0.35)
		availability := availabilityBoost(profile.Availability)
		score := clampFloat(0.45*textScore+0.35*vectorScore+quality+availability, 0, 1)
		if score <= 0 {
			score = 0.12 + quality + availability
		}
		bridgeType := bridgeTypeForAsk(ask.DesiredHelpType, score, profile)
		candidates = append(candidates, bridgeCandidate{
			user:       profile.User,
			reason:     buildAdviceBridgeReason(ask, profile, bridgeType),
			bridgeType: bridgeType,
			confidence: roundConfidence(score),
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
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
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
	if err := s.db.Preload("Ask.User").Preload("RecommendedUser").First(&bridge, "id = ?", id).Error; err != nil {
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
