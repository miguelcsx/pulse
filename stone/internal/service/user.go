package service

import (
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/model"
)

// UserService handles user profile operations.
type UserService struct {
	db *gorm.DB
}

// NewUserService creates a new UserService.
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// GetByID returns a user by their ID.
func (s *UserService) GetByID(id uuid.UUID) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

// profileRow holds the result of the single combined profile query.
type profileRow struct {
	model.User
	FollowerCount  int  `gorm:"column:follower_count"`
	FollowingCount int  `gorm:"column:following_count"`
	ContentCount   int  `gorm:"column:content_count"`
	IsFollowing    bool `gorm:"column:is_following"`
	IsBlocked      bool `gorm:"column:is_blocked"`
}

// GetProfile returns a user profile with follower/following/content counts
// and whether the viewer is following or has blocked the user.
// All data is fetched in a single query using subselects instead of 5+
// separate COUNT queries.
func (s *UserService) GetProfile(id uuid.UUID, viewerID uuid.UUID) (*model.UserProfile, error) {
	var row profileRow

	query := s.db.Table("users").
		Select(`
			users.*,
			(SELECT COUNT(*) FROM follows WHERE followee_id = users.id) AS follower_count,
			(SELECT COUNT(*) FROM follows WHERE follower_id = users.id) AS following_count,
			(SELECT COUNT(*) FROM contents WHERE creator_id = users.id) AS content_count,
			CASE WHEN ? != '00000000-0000-0000-0000-000000000000' AND ? != users.id
				THEN (SELECT COUNT(*) > 0 FROM follows WHERE follower_id = ? AND followee_id = users.id)
				ELSE false
			END AS is_following,
			CASE WHEN ? != '00000000-0000-0000-0000-000000000000' AND ? != users.id
				THEN (SELECT COUNT(*) > 0 FROM blocks WHERE blocker_id = ? AND blocked_id = users.id)
				ELSE false
			END AS is_blocked
		`, viewerID, viewerID, viewerID, viewerID, viewerID, viewerID).
		Where("users.id = ?", id)

	if err := query.Take(&row).Error; err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	profile := &model.UserProfile{
		User:           row.User,
		FollowerCount:  row.FollowerCount,
		FollowingCount: row.FollowingCount,
		ContentCount:   row.ContentCount,
		IsFollowing:    row.IsFollowing,
		IsBlocked:      row.IsBlocked,
	}

	var trust model.TrustProfile
	if err := s.db.First(&trust, "user_id = ?", id).Error; err == nil {
		profile.TrustProfile = &trust
	}

	return profile, nil
}

// Update updates the user's profile fields. Only non-nil fields are updated.
func (s *UserService) Update(id uuid.UUID, displayName, bio, location, avatarURL *string) (*model.User, error) {
	var user model.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	updates := make(map[string]interface{})
	if displayName != nil {
		updates["display_name"] = *displayName
	}
	if bio != nil {
		updates["bio"] = *bio
	}
	if location != nil {
		updates["location"] = *location
	}
	if avatarURL != nil {
		updates["avatar_url"] = *avatarURL
	}

	if len(updates) > 0 {
		if err := s.db.Model(&user).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("failed to update user: %w", err)
		}
	}

	return &user, nil
}
