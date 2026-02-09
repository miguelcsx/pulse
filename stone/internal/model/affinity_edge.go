package model

import (
	"time"

	"github.com/google/uuid"
)

// UserAffinityEdge stores directional affinity from one user to another.
// Scores are decayed over time and updated from behavioral events.
type UserAffinityEdge struct {
	UserID       uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	OtherUserID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"other_user_id"`
	Score7D      float64   `gorm:"not null;default:0" json:"score_7d"`
	Score30D     float64   `gorm:"not null;default:0" json:"score_30d"`
	LastSignalAt time.Time `gorm:"not null;index" json:"last_signal_at"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
