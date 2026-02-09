package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RefreshToken represents a long-lived session credential that can be rotated and revoked.
type RefreshToken struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	User        User       `gorm:"foreignKey:UserID" json:"user,omitempty"`
	ExpiresAt   time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt   *time.Time `gorm:"index" json:"revoked_at,omitempty"`
	ReplacedBy  *uuid.UUID `gorm:"type:uuid" json:"replaced_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (r *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
