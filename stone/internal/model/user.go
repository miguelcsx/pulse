package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Handle      string    `gorm:"uniqueIndex;size:30;not null" json:"handle"`
	Email       string    `gorm:"uniqueIndex;not null" json:"-"`
	Password    string    `gorm:"not null" json:"-"`
	DisplayName string    `gorm:"size:100" json:"display_name"`
	Bio         string    `gorm:"size:500" json:"bio"`
	AvatarURL   string    `json:"avatar_url"`
	Location    string    `gorm:"size:100" json:"location"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

type UserProfile struct {
	User
	FollowerCount  int           `json:"follower_count"`
	FollowingCount int           `json:"following_count"`
	ContentCount   int           `json:"content_count"`
	IsFollowing    bool          `json:"is_following,omitempty"`
	IsBlocked      bool          `json:"is_blocked,omitempty"`
	TrustProfile   *TrustProfile `json:"trust_profile,omitempty"`
}
