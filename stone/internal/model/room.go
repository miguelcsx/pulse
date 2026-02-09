package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Room struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ClusterKey  string       `gorm:"not null;index" json:"cluster_key"`
	Tags        []Tag        `gorm:"many2many:room_tags" json:"tags,omitempty"`
	Members     []RoomMember `gorm:"foreignKey:RoomID" json:"members,omitempty"`
	MemberCount int64        `gorm:"-" json:"member_count"`
	ExpiresAt   time.Time    `gorm:"index" json:"expires_at"`
	CreatedAt   time.Time    `json:"created_at"`
}

func (r *Room) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

type RoomMember struct {
	RoomID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"room_id"`
	UserID   uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	User     User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	JoinedAt time.Time `json:"joined_at"`
}

type RoomTag struct {
	RoomID uuid.UUID `gorm:"type:uuid;primaryKey"`
	TagID  uuid.UUID `gorm:"type:uuid;primaryKey"`
}
