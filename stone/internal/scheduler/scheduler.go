package scheduler

import (
	"context"
	"log/slog"
	"time"

	"gorm.io/gorm"

	"github.com/pulse/stone/internal/config"
	"github.com/pulse/stone/internal/model"
)

// Scheduler runs periodic background cleanup tasks.
type Scheduler struct {
	db  *gorm.DB
	cfg *config.Config

	// mediaEnqueue is called to re-enqueue stuck media assets for processing.
	// Injected by the caller so the scheduler doesn't depend on MediaService directly.
	mediaEnqueue func(assetIDs []string)
}

// New creates a new Scheduler. The mediaEnqueue callback is optional (pass nil
// to skip media recovery).
func New(db *gorm.DB, cfg *config.Config, mediaEnqueue func(assetIDs []string)) *Scheduler {
	return &Scheduler{
		db:           db,
		cfg:          cfg,
		mediaEnqueue: mediaEnqueue,
	}
}

// Start begins the periodic task loop. It blocks until ctx is cancelled.
// Callers should launch this in a goroutine.
func (s *Scheduler) Start(ctx context.Context) {
	interval := s.cfg.SchedulerInterval
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	slog.Info("scheduler started", "interval", interval)

	// Run once immediately on startup.
	s.runAll()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("scheduler stopped")
			return
		case <-ticker.C:
			s.runAll()
		}
	}
}

func (s *Scheduler) runAll() {
	s.cleanupExpiredRooms()
	s.cleanupExpiredTokens()
	s.recoverStuckMedia()
	s.cleanupStaleEvents()
}

// cleanupExpiredRooms deletes rooms that have passed their expiration time.
// Room members are removed via ON DELETE CASCADE.
func (s *Scheduler) cleanupExpiredRooms() {
	result := s.db.Where("expires_at <= ?", time.Now()).Delete(&model.Room{})
	if result.Error != nil {
		slog.Error("scheduler: failed to cleanup expired rooms", "error", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		slog.Info("scheduler: cleaned up expired rooms", "count", result.RowsAffected)
	}
}

// cleanupExpiredTokens removes refresh tokens that are both expired and revoked,
// or expired beyond a grace period (7 days past expiry).
func (s *Scheduler) cleanupExpiredTokens() {
	gracePeriod := time.Now().Add(-7 * 24 * time.Hour)

	// Delete tokens that expired more than 7 days ago (whether revoked or not).
	result := s.db.
		Where("expires_at < ?", gracePeriod).
		Delete(&model.RefreshToken{})
	if result.Error != nil {
		slog.Error("scheduler: failed to cleanup expired tokens", "error", result.Error)
		return
	}

	// Also delete revoked tokens that expired.
	revokedResult := s.db.
		Where("revoked_at IS NOT NULL AND expires_at < ?", time.Now()).
		Delete(&model.RefreshToken{})
	if revokedResult.Error != nil {
		slog.Error("scheduler: failed to cleanup revoked tokens", "error", revokedResult.Error)
		return
	}

	total := result.RowsAffected + revokedResult.RowsAffected
	if total > 0 {
		slog.Info("scheduler: cleaned up expired/revoked tokens", "count", total)
	}
}

// recoverStuckMedia finds media assets stuck in "uploaded" or "processing" status
// longer than the configured threshold and re-enqueues them.
func (s *Scheduler) recoverStuckMedia() {
	if s.mediaEnqueue == nil {
		return
	}

	threshold := time.Now().Add(-s.cfg.MediaRecoveryThreshold)

	var stuckIDs []string
	err := s.db.Model(&model.MediaAsset{}).
		Where("status IN ? AND updated_at < ?",
			[]string{model.MediaAssetStatusUploaded, model.MediaAssetStatusProcessing},
			threshold,
		).
		Pluck("id", &stuckIDs).Error
	if err != nil {
		slog.Error("scheduler: failed to query stuck media assets", "error", err)
		return
	}

	if len(stuckIDs) == 0 {
		return
	}

	// Reset processing assets back to uploaded so they can be re-processed.
	s.db.Model(&model.MediaAsset{}).
		Where("status = ? AND updated_at < ?", model.MediaAssetStatusProcessing, threshold).
		Updates(map[string]any{
			"status":     model.MediaAssetStatusUploaded,
			"updated_at": time.Now(),
		})

	slog.Info("scheduler: recovering stuck media assets", "count", len(stuckIDs))
	s.mediaEnqueue(stuckIDs)
}

// cleanupStaleEvents deletes events older than 90 days to prevent unbounded
// table growth. In production, events should be archived before deletion.
func (s *Scheduler) cleanupStaleEvents() {
	cutoff := time.Now().Add(-90 * 24 * time.Hour)

	result := s.db.Where("created_at < ?", cutoff).Delete(&model.Event{})
	if result.Error != nil {
		slog.Error("scheduler: failed to cleanup stale events", "error", result.Error)
		return
	}
	if result.RowsAffected > 0 {
		slog.Info("scheduler: cleaned up stale events", "count", result.RowsAffected)
	}
}
