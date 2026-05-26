package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Tag is a user-defined hashtag. Anyone can create tags by using them.
// Tags are normalized to lowercase. No fixed namespace — users define freely.
type Tag struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Name       string    `gorm:"uniqueIndex;size:100;not null" json:"name"`
	UsageCount int       `gorm:"default:0" json:"usage_count"`
	CreatedAt  time.Time `json:"created_at"`
}

func (t *Tag) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
