package service

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/store"
)

// MediaService manages media asset lifecycle: ingest, processing, and ready-state delivery.
type MediaService struct {
	db      *gorm.DB
	storage store.Storage
	queue   chan uuid.UUID
}

type countingReader struct {
	r io.Reader
	n int64
}

func (r *countingReader) Read(p []byte) (int, error) {
	n, err := r.r.Read(p)
	r.n += int64(n)
	return n, err
}

func NewMediaService(db *gorm.DB, storage store.Storage) *MediaService {
	s := &MediaService{
		db:      db,
		storage: storage,
		queue:   make(chan uuid.UUID, 1024),
	}
	go s.runProcessor()
	return s
}

func (s *MediaService) InitUpload(ownerID uuid.UUID, contentType, filename, mimeType string, sizeBytes int64) (*model.MediaAsset, error) {
	switch contentType {
	case model.ContentTypeImage, model.ContentTypeVideo, model.ContentTypeShortVideo:
	default:
		return nil, fmt.Errorf("invalid media content type: %s", contentType)
	}

	asset := &model.MediaAsset{
		OwnerID:     ownerID,
		ContentType: contentType,
		Filename:    strings.TrimSpace(filename),
		MimeType:    strings.TrimSpace(mimeType),
		SizeBytes:   sizeBytes,
		Status:      model.MediaAssetStatusInitiated,
	}
	if err := s.db.Create(asset).Error; err != nil {
		return nil, fmt.Errorf("failed to create media asset: %w", err)
	}

	s.enrichURLs(asset)
	return asset, nil
}

func (s *MediaService) UploadBinary(assetID, ownerID uuid.UUID, reader io.Reader, fileSize int64) (*model.MediaAsset, error) {
	var asset model.MediaAsset
	if err := s.db.First(&asset, "id = ? AND owner_id = ?", assetID, ownerID).Error; err != nil {
		return nil, fmt.Errorf("failed to find media asset: %w", err)
	}

	if asset.Status != model.MediaAssetStatusInitiated && asset.Status != model.MediaAssetStatusFailed {
		return nil, fmt.Errorf("media asset is not uploadable in status %s", asset.Status)
	}

	if asset.SizeBytes > 0 && fileSize > 0 && fileSize > asset.SizeBytes {
		return nil, fmt.Errorf("file is larger than declared size")
	}

	ext := strings.ToLower(filepath.Ext(asset.Filename))
	if ext == "" {
		switch asset.ContentType {
		case model.ContentTypeImage:
			ext = ".jpg"
		case model.ContentTypeVideo, model.ContentTypeShortVideo:
			ext = ".mp4"
		}
	}

	originalPath := fmt.Sprintf("media/originals/%s%s", asset.ID.String(), ext)
	counter := &countingReader{r: reader}
	if err := s.storage.SaveAs(originalPath, counter); err != nil {
		return nil, fmt.Errorf("failed to save media binary: %w", err)
	}
	actualSize := counter.n
	if asset.SizeBytes > 0 && actualSize > asset.SizeBytes {
		_ = s.storage.Delete(originalPath)
		return nil, fmt.Errorf("file is larger than declared size")
	}

	now := time.Now()
	asset.OriginalPath = originalPath
	asset.Status = model.MediaAssetStatusUploaded
	asset.ErrorMessage = ""
	asset.UpdatedAt = now
	if actualSize > 0 {
		asset.SizeBytes = actualSize
	} else if fileSize > 0 {
		asset.SizeBytes = fileSize
	}

	if err := s.db.Model(&asset).Updates(map[string]any{
		"original_path": asset.OriginalPath,
		"status":        asset.Status,
		"error_message": "",
		"size_bytes":    asset.SizeBytes,
		"updated_at":    asset.UpdatedAt,
	}).Error; err != nil {
		return nil, fmt.Errorf("failed to persist uploaded media: %w", err)
	}

	s.enqueue(asset.ID)
	s.enrichURLs(&asset)
	return &asset, nil
}

func (s *MediaService) GetForOwner(assetID, ownerID uuid.UUID) (*model.MediaAsset, error) {
	var asset model.MediaAsset
	if err := s.db.First(&asset, "id = ? AND owner_id = ?", assetID, ownerID).Error; err != nil {
		return nil, fmt.Errorf("failed to find media asset: %w", err)
	}
	s.enrichURLs(&asset)
	return &asset, nil
}

func (s *MediaService) GetReadyForOwner(assetID, ownerID uuid.UUID) (*model.MediaAsset, error) {
	asset, err := s.GetForOwner(assetID, ownerID)
	if err != nil {
		return nil, err
	}
	if asset.Status != model.MediaAssetStatusReady {
		return nil, fmt.Errorf("media asset %s is not ready", asset.ID)
	}
	return asset, nil
}

func (s *MediaService) enqueue(assetID uuid.UUID) {
	select {
	case s.queue <- assetID:
	default:
		go func() { s.queue <- assetID }()
	}
}

func (s *MediaService) runProcessor() {
	for assetID := range s.queue {
		_ = s.processAsset(assetID)
	}
}

func (s *MediaService) processAsset(assetID uuid.UUID) error {
	var asset model.MediaAsset
	if err := s.db.First(&asset, "id = ?", assetID).Error; err != nil {
		return err
	}

	if asset.Status != model.MediaAssetStatusUploaded {
		return nil
	}

	now := time.Now()
	if err := s.db.Model(&asset).Updates(map[string]any{
		"status":     model.MediaAssetStatusProcessing,
		"updated_at": now,
	}).Error; err != nil {
		return err
	}

	if strings.TrimSpace(asset.OriginalPath) == "" {
		return s.failAsset(asset.ID, "original media path missing")
	}

	// Placeholder processing pipeline: pass-through playback path.
	playbackPath := asset.OriginalPath
	readyAt := time.Now()
	if err := s.db.Model(&asset).Updates(map[string]any{
		"playback_path": playbackPath,
		"status":        model.MediaAssetStatusReady,
		"ready_at":      readyAt,
		"updated_at":    readyAt,
		"error_message": "",
	}).Error; err != nil {
		return err
	}

	return nil
}

func (s *MediaService) failAsset(assetID uuid.UUID, reason string) error {
	return s.db.Model(&model.MediaAsset{}).
		Where("id = ?", assetID).
		Updates(map[string]any{
			"status":        model.MediaAssetStatusFailed,
			"error_message": reason,
			"updated_at":    time.Now(),
		}).Error
}

func (s *MediaService) enrichURLs(asset *model.MediaAsset) {
	if asset == nil {
		return
	}
	if asset.OriginalPath != "" {
		asset.OriginalURL = s.storage.URL(asset.OriginalPath)
	}
	if asset.PlaybackPath != "" {
		asset.PlaybackURL = s.storage.URL(asset.PlaybackPath)
	}
}
