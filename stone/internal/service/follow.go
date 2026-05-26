package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/neo4j/neo4j-go-driver/v5/neo4j"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/store"
)

// FollowService handles follow/unfollow/block/unblock operations.
type FollowService struct {
	db    *gorm.DB
	graph *store.GraphStore
}

var (
	ErrFollowBlocked = errors.New("cannot follow due to block relationship")
)

// NewFollowService creates a new FollowService.
func NewFollowService(db *gorm.DB, graph *store.GraphStore) *FollowService {
	return &FollowService{db: db, graph: graph}
}

// Follow creates a follow relationship between follower and followee.
func (s *FollowService) Follow(followerID, followeeID uuid.UUID) error {
	if followerID == followeeID {
		return errors.New("cannot follow yourself")
	}

	blocked, err := s.isBlockedEitherDirection(followerID, followeeID)
	if err != nil {
		return err
	}
	if blocked {
		return ErrFollowBlocked
	}

	follow := model.Follow{
		FollowerID: followerID,
		FolloweeID: followeeID,
		CreatedAt:  time.Now(),
	}

	if err := s.db.Create(&follow).Error; err != nil {
		return fmt.Errorf("failed to create follow: %w", err)
	}
	if err := s.syncFollowToGraph(followerID, followeeID); err != nil {
		return err
	}
	return nil
}

// Unfollow removes the follow relationship between follower and followee.
func (s *FollowService) Unfollow(followerID, followeeID uuid.UUID) error {
	result := s.db.Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Delete(&model.Follow{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete follow: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("follow relationship not found")
	}
	if err := s.removeFollowFromGraph(followerID, followeeID); err != nil {
		return err
	}
	return nil
}

// Block creates a block relationship and removes any existing follow relationships
// in both directions between the two users.
func (s *FollowService) Block(blockerID, blockedID uuid.UUID) error {
	if blockerID == blockedID {
		return errors.New("cannot block yourself")
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		// Remove follows in both directions.
		tx.Where("follower_id = ? AND followee_id = ?", blockerID, blockedID).
			Delete(&model.Follow{})
		tx.Where("follower_id = ? AND followee_id = ?", blockedID, blockerID).
			Delete(&model.Follow{})

		block := model.Block{
			BlockerID: blockerID,
			BlockedID: blockedID,
			CreatedAt: time.Now(),
		}

		if err := tx.Create(&block).Error; err != nil {
			return fmt.Errorf("failed to create block: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}
	return s.syncBlockToGraph(blockerID, blockedID)
}

// Unblock removes the block relationship.
func (s *FollowService) Unblock(blockerID, blockedID uuid.UUID) error {
	result := s.db.Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Delete(&model.Block{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete block: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("block relationship not found")
	}
	if err := s.removeBlockFromGraph(blockerID, blockedID); err != nil {
		return err
	}
	return nil
}

// IsFollowing checks whether followerID is following followeeID.
func (s *FollowService) IsFollowing(followerID, followeeID uuid.UUID) (bool, error) {
	var count int64
	err := s.db.Model(&model.Follow{}).
		Where("follower_id = ? AND followee_id = ?", followerID, followeeID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check follow: %w", err)
	}
	return count > 0, nil
}

// IsBlocked checks whether blockerID has blocked blockedID.
func (s *FollowService) IsBlocked(blockerID, blockedID uuid.UUID) (bool, error) {
	var count int64
	err := s.db.Model(&model.Block{}).
		Where("blocker_id = ? AND blocked_id = ?", blockerID, blockedID).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check block: %w", err)
	}
	return count > 0, nil
}

func (s *FollowService) isBlockedEitherDirection(userA, userB uuid.UUID) (bool, error) {
	var count int64
	err := s.db.Model(&model.Block{}).
		Where("(blocker_id = ? AND blocked_id = ?) OR (blocker_id = ? AND blocked_id = ?)", userA, userB, userB, userA).
		Count(&count).Error
	if err != nil {
		return false, fmt.Errorf("failed to check block relationship: %w", err)
	}
	return count > 0, nil
}

func (s *FollowService) syncFollowToGraph(followerID, followeeID uuid.UUID) error {
	if s.graph == nil {
		return nil
	}

	ctx := context.Background()
	query := `
		MERGE (a:User {id: $followerID})
		MERGE (b:User {id: $followeeID})
		MERGE (a)-[:FOLLOWS]->(b)
	`
	params := map[string]any{
		"followerID": followerID.String(),
		"followeeID": followeeID.String(),
	}

	_, err := s.graph.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})
	return err
}

func (s *FollowService) removeFollowFromGraph(followerID, followeeID uuid.UUID) error {
	if s.graph == nil {
		return nil
	}

	ctx := context.Background()
	query := `
		MATCH (a:User {id: $followerID})-[r:FOLLOWS]->(b:User {id: $followeeID})
		DELETE r
	`
	params := map[string]any{
		"followerID": followerID.String(),
		"followeeID": followeeID.String(),
	}

	_, err := s.graph.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})
	return err
}

func (s *FollowService) syncBlockToGraph(blockerID, blockedID uuid.UUID) error {
	if s.graph == nil {
		return nil
	}

	ctx := context.Background()
	query := `
		MERGE (a:User {id: $blockerID})
		MERGE (b:User {id: $blockedID})
		WITH a, b
		OPTIONAL MATCH (a)-[f1:FOLLOWS]->(b)
		DELETE f1
		WITH a, b
		OPTIONAL MATCH (b)-[f2:FOLLOWS]->(a)
		DELETE f2
		MERGE (a)-[:BLOCKS]->(b)
	`
	params := map[string]any{
		"blockerID": blockerID.String(),
		"blockedID": blockedID.String(),
	}

	_, err := s.graph.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})
	return err
}

func (s *FollowService) removeBlockFromGraph(blockerID, blockedID uuid.UUID) error {
	if s.graph == nil {
		return nil
	}

	ctx := context.Background()
	query := `
		MATCH (a:User {id: $blockerID})-[r:BLOCKS]->(b:User {id: $blockedID})
		DELETE r
	`
	params := map[string]any{
		"blockerID": blockerID.String(),
		"blockedID": blockedID.String(),
	}

	_, err := s.graph.ExecuteWrite(ctx, func(tx neo4j.ManagedTransaction) (any, error) {
		_, err := tx.Run(ctx, query, params)
		return nil, err
	})
	return err
}
