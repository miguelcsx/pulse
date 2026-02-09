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

// GetProfile returns a user profile with follower/following/content counts
// and whether the viewer is following or has blocked the user.
func (s *UserService) GetProfile(id uuid.UUID, viewerID uuid.UUID) (*model.UserProfile, error) {
	var user model.User
	if err := s.db.First(&user, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	profile := &model.UserProfile{User: user}

	// Follower count: how many users follow this user.
	var followerCount int64
	s.db.Model(&model.Follow{}).Where("followee_id = ?", id).Count(&followerCount)
	profile.FollowerCount = int(followerCount)

	// Following count: how many users this user follows.
	var followingCount int64
	s.db.Model(&model.Follow{}).Where("follower_id = ?", id).Count(&followingCount)
	profile.FollowingCount = int(followingCount)

	// Content count.
	var contentCount int64
	s.db.Model(&model.Content{}).Where("creator_id = ?", id).Count(&contentCount)
	profile.ContentCount = int(contentCount)

	// Check relationship from viewer's perspective.
	if viewerID != uuid.Nil && viewerID != id {
		var followExists int64
		s.db.Model(&model.Follow{}).
			Where("follower_id = ? AND followee_id = ?", viewerID, id).
			Count(&followExists)
		profile.IsFollowing = followExists > 0

		var blockExists int64
		s.db.Model(&model.Block{}).
			Where("blocker_id = ? AND blocked_id = ?", viewerID, id).
			Count(&blockExists)
		profile.IsBlocked = blockExists > 0
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
