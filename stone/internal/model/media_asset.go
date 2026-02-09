package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	MediaAssetStatusInitiated  = "initiated"
	MediaAssetStatusUploaded   = "uploaded"
	MediaAssetStatusProcessing = "processing"
	MediaAssetStatusReady      = "ready"
	MediaAssetStatusFailed     = "failed"
)

// MediaAsset tracks uploaded multimedia files and asynchronous processing state.
type MediaAsset struct {
	ID           uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	OwnerID      uuid.UUID  `gorm:"type:uuid;not null;index" json:"owner_id"`
	Owner        User       `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	ContentType  string     `gorm:"size:20;not null" json:"content_type"`
	OriginalPath string     `gorm:"type:text;not null;default:''" json:"original_path"`
	PlaybackPath string     `gorm:"type:text;not null;default:''" json:"playback_path"`
	Filename     string     `gorm:"size:255;not null;default:''" json:"filename"`
	MimeType     string     `gorm:"size:120;not null;default:''" json:"mime_type"`
	SizeBytes    int64      `gorm:"not null;default:0" json:"size_bytes"`
	Status       string     `gorm:"size:20;not null;index" json:"status"`
	ErrorMessage string     `gorm:"size:1000;not null;default:''" json:"error_message"`
	ReadyAt      *time.Time `json:"ready_at,omitempty"`
	OriginalURL  string     `gorm:"-" json:"original_url,omitempty"`
	PlaybackURL  string     `gorm:"-" json:"playback_url,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

func (a *MediaAsset) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	if a.Status == "" {
		a.Status = MediaAssetStatusInitiated
	}
	return nil
}
