package model

import (
	"time"

	"github.com/google/uuid"
)

type Follow struct {
	FollowerID uuid.UUID `gorm:"type:uuid;primaryKey" json:"follower_id"`
	FolloweeID uuid.UUID `gorm:"type:uuid;primaryKey" json:"followee_id"`
	Follower   User      `gorm:"foreignKey:FollowerID" json:"follower,omitempty"`
	Followee   User      `gorm:"foreignKey:FolloweeID" json:"followee,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type Block struct {
	BlockerID uuid.UUID `gorm:"type:uuid;primaryKey" json:"blocker_id"`
	BlockedID uuid.UUID `gorm:"type:uuid;primaryKey" json:"blocked_id"`
	CreatedAt time.Time `json:"created_at"`
}
