package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Event struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	Type       string     `gorm:"size:50;not null;index" json:"type"` // view, skip, follow, unfollow, upload, enter_room, etc.
	TargetType string     `gorm:"size:50" json:"target_type"`         // content, user, room, path
	TargetID   *uuid.UUID `gorm:"type:uuid" json:"target_id,omitempty"`
	Metadata   string     `gorm:"type:jsonb;default:'{}'" json:"metadata"`
	CreatedAt  time.Time  `json:"created_at"`
}

func (e *Event) BeforeCreate(tx *gorm.DB) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}
