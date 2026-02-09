package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// ContentType enumerates the kinds of content users can post.
const (
	ContentTypeImage      = "image"
	ContentTypeVideo      = "video"
	ContentTypeShortVideo = "short_video"
	ContentTypeText       = "text"
)

// Content represents a piece of user-created content (image, video, short video, or text post).
type Content struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatorID    uuid.UUID      `gorm:"type:uuid;not null;index" json:"creator_id"`
	Creator      User           `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	ContentType  string         `gorm:"size:20;not null;default:'image'" json:"content_type"` // image, video, short_video, text
	MediaAssetID *uuid.UUID     `gorm:"type:uuid;index" json:"media_asset_id,omitempty"`
	MediaAsset   *MediaAsset    `gorm:"foreignKey:MediaAssetID" json:"media_asset,omitempty"`
	MediaURL     string         `json:"media_url"`             // empty for text posts
	Body         string         `gorm:"type:text" json:"body"` // text content or caption
	Tags         []Tag          `gorm:"many2many:content_tags" json:"tags,omitempty"`
	Reactions    map[string]int `gorm:"-" json:"reactions,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

func (c *Content) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

// ContentTag is the join table for content<->tag with optional weight.
type ContentTag struct {
	ContentID uuid.UUID `gorm:"type:uuid;primaryKey" json:"content_id"`
	TagID     uuid.UUID `gorm:"type:uuid;primaryKey" json:"tag_id"`
}

// Reaction replaces likes. Semantic, non-binary reactions that map to mood vectors.
// Examples: "gave_me_energy", "calmed_me", "on_repeat", "surprised_me", "my_aesthetic"
type Reaction struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	ContentID uuid.UUID `gorm:"type:uuid;not null;index" json:"content_id"`
	Kind      string    `gorm:"size:30;not null" json:"kind"` // semantic reaction type
	CreatedAt time.Time `json:"created_at"`
}

func (r *Reaction) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
