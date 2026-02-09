package model

import (
	"time"

	"github.com/google/uuid"
)

// Embedding stores vector embeddings for users/content/tags.
type Embedding struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	EntityType string    `gorm:"size:20;not null;index" json:"entity_type"` // user, content, tag
	EntityID   uuid.UUID `gorm:"type:uuid;not null;index" json:"entity_id"`
	Vector     string    `gorm:"type:vector(256)" json:"-"` // pgvector column
	CreatedAt  time.Time `json:"created_at"`
}
