package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Path struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	CreatorID       uuid.UUID  `gorm:"type:uuid;not null;index" json:"creator_id"`
	Creator         User       `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	Title           string     `gorm:"size:200;not null" json:"title"`
	Description     string     `gorm:"size:1000" json:"description"`
	SystemGenerated bool       `gorm:"default:false" json:"system_generated"`
	Items           []PathItem `gorm:"foreignKey:PathID;constraint:OnDelete:CASCADE" json:"items,omitempty"`
	FollowerCount   int        `gorm:"default:0" json:"follower_count"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (p *Path) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

type PathItem struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PathID    uuid.UUID `gorm:"type:uuid;not null;index" json:"path_id"`
	ContentID uuid.UUID `gorm:"type:uuid;not null" json:"content_id"`
	Content   Content   `gorm:"foreignKey:ContentID" json:"content,omitempty"`
	Position  int       `gorm:"not null" json:"position"`
	Note      string    `gorm:"size:500" json:"note"`
}

func (pi *PathItem) BeforeCreate(tx *gorm.DB) error {
	if pi.ID == uuid.Nil {
		pi.ID = uuid.New()
	}
	return nil
}

type PathFollow struct {
	PathID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"path_id"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}
