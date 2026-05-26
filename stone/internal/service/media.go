package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/pulse/stone/internal/model"
	"github.com/pulse/stone/internal/store"
)

var (
	ErrMediaAssetNotFound = errors.New("media asset not found")
	ErrMediaNotUploadable = errors.New("media asset is not uploadable")
	ErrMediaTooLarge      = errors.New("file is larger than declared size")
	ErrMediaNotReady      = errors.New("media asset is not ready")
	ErrMediaInvalidMIME   = errors.New("file content does not match declared MIME type")
)

// allowedMIMEPrefixes maps content types to the set of MIME prefixes
// that are acceptable based on magic-byte sniffing.
var allowedMIMEPrefixes = map[string][]string{
	model.ContentTypeImage:      {"image/"},
	model.ContentTypeVideo:      {"video/", "application/mp4", "application/octet-stream"},
	model.ContentTypeShortVideo: {"video/", "application/mp4", "application/octet-stream"},
}

const mediaTaskType = "media:process"

type mediaTaskPayload struct {
	AssetID string `json:"asset_id"`
}

// MediaService manages media asset lifecycle: ingest, processing, and ready-state delivery.
type MediaService struct {
	db      *gorm.DB
	storage store.Storage

	queue  *asynq.Client
	server *asynq.Server

	shutdownOnce sync.Once
	done         chan struct{}
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

// sniffingReader wraps a reader and captures the first 512 bytes for MIME detection,
// then transparently replays them on subsequent reads.
type sniffingReader struct {
	r       io.Reader
	buf     []byte
	sniffed bool
	replay  int
}

func newSniffingReader(r io.Reader) *sniffingReader {
	return &sniffingReader{r: r}
}

func (s *sniffingReader) Read(p []byte) (int, error) {
	// If we still have buffered bytes to replay, serve those first.
	if s.replay < len(s.buf) {
		n := copy(p, s.buf[s.replay:])
		s.replay += n
		return n, nil
	}
	return s.r.Read(p)
}

// sniff reads up to 512 bytes and returns the detected MIME type.
// Subsequent calls to Read will replay the sniffed bytes.
func (s *sniffingReader) sniff() (string, error) {
	if s.sniffed {
		return http.DetectContentType(s.buf), nil
	}
	buf := make([]byte, 512)
	n, err := io.ReadAtLeast(s.r, buf, 1)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", err
	}
	s.buf = buf[:n]
	s.sniffed = true
	return http.DetectContentType(s.buf), nil
}

func NewMediaService(db *gorm.DB, storage store.Storage) *MediaService {
	s := &MediaService{
		db:      db,
		storage: storage,
		done:    make(chan struct{}),
	}

	if err := s.initQueue(); err != nil {
		slog.Error("media queue init failed; falling back to inline processing", "error", err)
	}

	s.recoverOnStartup()
	return s
}

func (s *MediaService) initQueue() error {
	redisURL := strings.TrimSpace(os.Getenv("REDIS_URL"))
	if redisURL == "" {
		redisURL = "redis://localhost:6379/0"
	}

	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		return fmt.Errorf("failed to parse redis url: %w", err)
	}

	clientOpt := asynq.RedisClientOpt{
		Addr:     opts.Addr,
		Password: opts.Password,
		DB:       opts.DB,
	}

	s.queue = asynq.NewClient(clientOpt)
	s.server = asynq.NewServer(clientOpt, asynq.Config{
		Concurrency: 10,
		Queues: map[string]int{
			"media": 1,
		},
	})

	mux := asynq.NewServeMux()
	mux.HandleFunc(mediaTaskType, s.handleProcessMediaTask)

	go func() {
		defer close(s.done)
		if err := s.server.Run(mux); err != nil {
			slog.Error("media worker stopped", "error", err)
		}
	}()

	return nil
}

// Shutdown gracefully drains the processing queue and stops the processor goroutine.
func (s *MediaService) Shutdown() {
	s.shutdownOnce.Do(func() {
		if s.server != nil {
			s.server.Shutdown()
		}
		if s.queue != nil {
			_ = s.queue.Close()
		}
		if s.done != nil {
			<-s.done
		}
	})
}

// RecoverAssets re-enqueues media assets by their string IDs. This is the callback
// used by the scheduler for periodic media recovery sweeps.
func (s *MediaService) RecoverAssets(assetIDs []string) {
	for _, rawID := range assetIDs {
		parsed, err := uuid.Parse(rawID)
		if err != nil {
			slog.Warn("media recovery: skipping invalid asset ID", "id", rawID, "error", err)
			continue
		}
		s.enqueue(parsed)
	}
}

// recoverOnStartup queries for assets stuck in "uploaded" status and re-enqueues them.
func (s *MediaService) recoverOnStartup() {
	var stuckIDs []uuid.UUID
	err := s.db.Model(&model.MediaAsset{}).
		Where("status = ?", model.MediaAssetStatusUploaded).
		Pluck("id", &stuckIDs).Error
	if err != nil {
		slog.Error("media recovery: failed to query uploaded assets on startup", "error", err)
		return
	}
	if len(stuckIDs) == 0 {
		return
	}
	slog.Info("media recovery: re-enqueuing uploaded assets from previous run", "count", len(stuckIDs))
	for _, id := range stuckIDs {
		s.enqueue(id)
	}
}

func (s *MediaService) InitUpload(ownerID uuid.UUID, contentType, filename, mimeType string, sizeBytes int64) (*model.MediaAsset, error) {
	if !model.IsMediaContentType(contentType) {
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
		return nil, ErrMediaAssetNotFound
	}

	if asset.Status != model.MediaAssetStatusInitiated && asset.Status != model.MediaAssetStatusFailed {
		return nil, ErrMediaNotUploadable
	}

	if asset.SizeBytes > 0 && fileSize > 0 && fileSize > asset.SizeBytes {
		return nil, ErrMediaTooLarge
	}

	// Validate MIME type via magic-byte sniffing.
	sniffer := newSniffingReader(reader)
	detectedMIME, err := sniffer.sniff()
	if err != nil {
		return nil, fmt.Errorf("failed to read file header for MIME detection: %w", err)
	}
	if !isAllowedMIME(asset.ContentType, detectedMIME) {
		return nil, fmt.Errorf("%w: expected %s content, detected %s", ErrMediaInvalidMIME, asset.ContentType, detectedMIME)
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
	counter := &countingReader{r: sniffer}
	if err := s.storage.SaveAs(originalPath, counter); err != nil {
		return nil, fmt.Errorf("failed to save media binary: %w", err)
	}
	actualSize := counter.n
	if asset.SizeBytes > 0 && actualSize > asset.SizeBytes {
		_ = s.storage.Delete(originalPath)
		return nil, ErrMediaTooLarge
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
		return nil, ErrMediaAssetNotFound
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
		return nil, ErrMediaNotReady
	}
	return asset, nil
}

func (s *MediaService) enqueue(assetID uuid.UUID) {
	if s.queue == nil {
		go func() {
			if err := s.processAsset(assetID); err != nil {
				slog.Error("media processing failed (inline)", "asset_id", assetID, "error", err)
			}
		}()
		return
	}

	payload, _ := json.Marshal(mediaTaskPayload{AssetID: assetID.String()})
	task := asynq.NewTask(mediaTaskType, payload)
	if _, err := s.queue.Enqueue(task, asynq.Queue("media")); err != nil {
		slog.Error("failed to enqueue media task", "asset_id", assetID, "error", err)
	}
}

func (s *MediaService) handleProcessMediaTask(_ context.Context, t *asynq.Task) error {
	var payload mediaTaskPayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return err
	}

	assetID, err := uuid.Parse(payload.AssetID)
	if err != nil {
		return err
	}

	return s.processAsset(assetID)
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

	// Processing pipeline: currently pass-through.
	// Future: image resizing, video transcoding, thumbnail generation.
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

// isAllowedMIME checks whether the detected MIME type is acceptable for the
// given content type.
func isAllowedMIME(contentType, detectedMIME string) bool {
	prefixes, ok := allowedMIMEPrefixes[contentType]
	if !ok {
		// Unknown content type (e.g. text) — no MIME check needed.
		return true
	}
	lower := strings.ToLower(detectedMIME)
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	return false
}

// ContextShutdown returns a function suitable for use in graceful shutdown
// sequences. It is equivalent to calling Shutdown().
func (s *MediaService) ContextShutdown(_ context.Context) error {
	s.Shutdown()
	return nil
}
