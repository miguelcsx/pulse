package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/model"
)

// EventInput represents input for recording an analytics event.
type EventInput struct {
	Type       string `json:"type"`
	TargetType string `json:"target_type"`
	TargetID   string `json:"target_id"`
	Metadata   string `json:"metadata"`
}

// EventService handles analytics event recording.
type EventService struct {
	db *gorm.DB
}

var ErrInvalidEventInput = errors.New("invalid event input")

type affinitySignal struct {
	targetType string
	targetID   *uuid.UUID
	delta      float64
}

const (
	affinityHalfLife7DDays  = 3.0
	affinityHalfLife30DDays = 14.0
)

var allowedEventTypes = map[string]struct{}{
	"view":          {},
	"dwell":         {},
	"skip":          {},
	"replay":        {},
	"save":          {},
	"reaction":      {},
	"path_follow":   {},
	"path_followed": {},
	"follow_path":   {},
	"tag_explore":   {},
	"tag_open":      {},
	"enter_room":    {},
	"room_entered":  {},
	"follow":        {},
}

var allowedTargetTypes = map[string]struct{}{
	"":        {},
	"user":    {},
	"content": {},
	"path":    {},
	"room":    {},
	"tag":     {},
}

// NewEventService creates a new EventService.
func NewEventService(db *gorm.DB) *EventService {
	return &EventService{db: db}
}

// RecordBatch inserts a batch of analytics events for a given user.
func (s *EventService) RecordBatch(userID uuid.UUID, events []EventInput) error {
	if len(events) == 0 {
		return nil
	}

	models := make([]model.Event, 0, len(events))
	signals := make([]affinitySignal, 0, len(events))
	for _, e := range events {
		eventType, targetType, err := normalizeEventContract(e.Type, e.TargetType)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidEventInput, err)
		}

		var targetID *uuid.UUID
		rawTargetID := strings.TrimSpace(e.TargetID)
		if rawTargetID != "" {
			parsedTargetID, err := uuid.Parse(rawTargetID)
			if err != nil {
				return fmt.Errorf("%w: invalid target_id %q", ErrInvalidEventInput, e.TargetID)
			}
			targetID = &parsedTargetID
		}
		if targetType == "" && targetID != nil {
			return fmt.Errorf("%w: target_type is required when target_id is provided", ErrInvalidEventInput)
		}
		if targetType != "" && targetID == nil {
			return fmt.Errorf("%w: target_id is required for target_type %q", ErrInvalidEventInput, targetType)
		}

		metadata, err := normalizeMetadata(e.Metadata)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidEventInput, err)
		}

		models = append(models, model.Event{
			UserID:     userID,
			Type:       eventType,
			TargetType: targetType,
			TargetID:   targetID,
			Metadata:   metadata,
		})

		delta := implicitSignalFromEvent(eventType, metadata)
		if delta != 0 && targetID != nil {
			signals = append(signals, affinitySignal{
				targetType: targetType,
				targetID:   targetID,
				delta:      delta,
			})
		}
	}

	now := time.Now()
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&models).Error; err != nil {
			return fmt.Errorf("failed to record events: %w", err)
		}
		if err := s.applyAffinitySignals(tx, userID, signals, now); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *EventService) applyAffinitySignals(tx *gorm.DB, userID uuid.UUID, signals []affinitySignal, now time.Time) error {
	if len(signals) == 0 {
		return nil
	}

	aggregated := make(map[uuid.UUID]float64)
	for _, signal := range signals {
		targetUserID, ok, err := s.resolveSignalTargetUser(tx, userID, signal.targetType, signal.targetID)
		if err != nil {
			return err
		}
		if !ok {
			continue
		}
		aggregated[targetUserID] += signal.delta
	}

	for targetUserID, delta := range aggregated {
		if delta == 0 {
			continue
		}
		if err := s.applyAffinityDelta(tx, userID, targetUserID, delta, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *EventService) resolveSignalTargetUser(tx *gorm.DB, userID uuid.UUID, targetType string, targetID *uuid.UUID) (uuid.UUID, bool, error) {
	if targetID == nil {
		return uuid.Nil, false, nil
	}

	switch strings.ToLower(strings.TrimSpace(targetType)) {
	case "user":
		if *targetID == userID {
			return uuid.Nil, false, nil
		}
		return *targetID, true, nil

	case "content":
		var row struct {
			CreatorID uuid.UUID `gorm:"column:creator_id"`
		}
		if err := tx.Model(&model.Content{}).
			Select("creator_id").
			Where("id = ?", *targetID).
			Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return uuid.Nil, false, nil
			}
			return uuid.Nil, false, fmt.Errorf("failed to resolve content creator: %w", err)
		}
		creatorID := row.CreatorID
		if creatorID == userID {
			return uuid.Nil, false, nil
		}
		return creatorID, true, nil

	case "path":
		var row struct {
			CreatorID uuid.UUID `gorm:"column:creator_id"`
		}
		if err := tx.Model(&model.Path{}).
			Select("creator_id").
			Where("id = ?", *targetID).
			Take(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return uuid.Nil, false, nil
			}
			return uuid.Nil, false, fmt.Errorf("failed to resolve path creator: %w", err)
		}
		creatorID := row.CreatorID
		if creatorID == userID {
			return uuid.Nil, false, nil
		}
		return creatorID, true, nil

	default:
		return uuid.Nil, false, nil
	}
}

func (s *EventService) applyAffinityDelta(tx *gorm.DB, userID, otherUserID uuid.UUID, delta float64, now time.Time) error {
	if userID == otherUserID {
		return nil
	}

	lambda7 := math.Ln2 / (affinityHalfLife7DDays * 24 * 60 * 60)
	lambda30 := math.Ln2 / (affinityHalfLife30DDays * 24 * 60 * 60)

	if delta > 0 {
		if err := tx.Exec(`
			INSERT INTO user_affinity_edges (
				user_id, other_user_id, score_7d, score_30d, last_signal_at, created_at, updated_at
			)
			VALUES (?, ?, GREATEST(?, 0), GREATEST(?, 0), ?, ?, ?)
			ON CONFLICT (user_id, other_user_id) DO UPDATE SET
				score_7d = GREATEST(
					0,
					user_affinity_edges.score_7d * EXP(-? * GREATEST(EXTRACT(EPOCH FROM (? - user_affinity_edges.last_signal_at)), 0)) + ?
				),
				score_30d = GREATEST(
					0,
					user_affinity_edges.score_30d * EXP(-? * GREATEST(EXTRACT(EPOCH FROM (? - user_affinity_edges.last_signal_at)), 0)) + ?
				),
				last_signal_at = ?,
				updated_at = ?
		`,
			userID, otherUserID, delta, delta, now, now, now,
			lambda7, now, delta,
			lambda30, now, delta,
			now, now,
		).Error; err != nil {
			return fmt.Errorf("failed to upsert user affinity edge: %w", err)
		}
		return nil
	}

	if err := tx.Exec(`
		UPDATE user_affinity_edges
		SET
			score_7d = GREATEST(
				0,
				user_affinity_edges.score_7d * EXP(-? * GREATEST(EXTRACT(EPOCH FROM (? - user_affinity_edges.last_signal_at)), 0)) + ?
			),
			score_30d = GREATEST(
				0,
				user_affinity_edges.score_30d * EXP(-? * GREATEST(EXTRACT(EPOCH FROM (? - user_affinity_edges.last_signal_at)), 0)) + ?
			),
			last_signal_at = ?,
			updated_at = ?
		WHERE user_id = ? AND other_user_id = ?
	`,
		lambda7, now, delta,
		lambda30, now, delta,
		now, now,
		userID, otherUserID,
	).Error; err != nil {
		return fmt.Errorf("failed to decay user affinity edge: %w", err)
	}

	return nil
}

func implicitSignalFromEvent(eventType string, metadata string) float64 {
	typ := strings.ToLower(strings.TrimSpace(eventType))
	switch typ {
	case "view", "dwell":
		dwellMS := metadataNumber(metadata, "dwell_ms")
		score := math.Log1p(dwellMS / 1000.0)
		return clamp(score, 0.2, 3.0)
	case "skip":
		atMS := metadataNumber(metadata, "at_ms")
		penalty := 1.0 - (atMS / 2000.0)
		return -clamp(penalty, 0, 1.0)
	case "replay":
		return 1.0
	case "save":
		return 1.5
	case "reaction":
		return 1.2
	case "path_follow", "path_followed", "follow_path":
		return 2.0
	case "tag_explore", "tag_open":
		return 0.4
	case "enter_room", "room_entered":
		return 0.2
	case "follow":
		return 1.8
	default:
		return 0
	}
}

func metadataNumber(metadata, key string) float64 {
	if strings.TrimSpace(metadata) == "" {
		return 0
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(metadata), &raw); err != nil {
		return 0
	}
	val, ok := raw[key]
	if !ok {
		return 0
	}
	switch n := val.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	default:
		return 0
	}
}

func clamp(v, minV, maxV float64) float64 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func normalizeEventContract(eventType, targetType string) (string, string, error) {
	normalizedType := strings.ToLower(strings.TrimSpace(eventType))
	if _, ok := allowedEventTypes[normalizedType]; !ok {
		return "", "", fmt.Errorf("unsupported event type %q", eventType)
	}

	normalizedTarget := strings.ToLower(strings.TrimSpace(targetType))
	if _, ok := allowedTargetTypes[normalizedTarget]; !ok {
		return "", "", fmt.Errorf("unsupported target type %q", targetType)
	}

	return normalizedType, normalizedTarget, nil
}

func normalizeMetadata(metadata string) (string, error) {
	trimmed := strings.TrimSpace(metadata)
	if trimmed == "" || strings.EqualFold(trimmed, "null") {
		return "{}", nil
	}

	var decoded any
	if err := json.Unmarshal([]byte(trimmed), &decoded); err != nil {
		return "", fmt.Errorf("invalid metadata JSON: %w", err)
	}
	if _, ok := decoded.(map[string]any); !ok {
		return "", fmt.Errorf("metadata must be a JSON object")
	}

	return trimmed, nil
}
