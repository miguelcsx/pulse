package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	HelpTypeAdvice   = "advice"
	HelpTypePeer     = "peer"
	HelpTypeMentor   = "mentor"
	HelpTypeFeedback = "feedback"

	BridgeTypeMentor              = "mentor"
	BridgeTypePeer                = "peer"
	BridgeTypeAdjacentPerspective = "adjacent_perspective"

	BridgeStatusSuggested = "suggested"
	BridgeStatusAsked     = "asked"
	BridgeStatusResponded = "responded"
	BridgeStatusDismissed = "dismissed"
)

// Ask is a human-help request. It replaces the feed as the primary product
// intent: users state what they need human perspective on.
type Ask struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID          uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	User            User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Question        string    `gorm:"type:text;not null" json:"question"`
	TriageSummary   string    `gorm:"type:text;not null;default:''" json:"triage_summary"`
	Topic           string    `gorm:"size:120;not null;default:''" json:"topic"`
	Urgency         string    `gorm:"size:40;not null;default:'soon'" json:"urgency"`
	DesiredHelpType string    `gorm:"size:40;not null;default:'advice'" json:"desired_help_type"`
	Visibility      string    `gorm:"size:40;not null;default:'community'" json:"visibility"`
	// Anonymous hides the asker's identity when an answered ask is published
	// to the Commons — vulnerability stays protected, the answerer gets credit.
	Anonymous bool    `gorm:"not null;default:false" json:"anonymous"`
	Embedding *string `gorm:"type:vector(1024)" json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (a *Ask) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// Bridge is an explainable recommendation from one ask to one useful human.
type Bridge struct {
	ID                uuid.UUID        `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	AskID             uuid.UUID        `gorm:"type:uuid;not null;index" json:"ask_id"`
	Ask               Ask              `gorm:"foreignKey:AskID" json:"ask,omitempty"`
	RequesterID       uuid.UUID        `gorm:"type:uuid;not null;index" json:"requester_id"`
	RecommendedUserID uuid.UUID        `gorm:"type:uuid;not null;index" json:"recommended_user_id"`
	RecommendedUser   User             `gorm:"foreignKey:RecommendedUserID" json:"recommended_user"`
	Reason            string           `gorm:"type:text;not null" json:"reason"`
	BridgeType        string           `gorm:"size:40;not null" json:"bridge_type"`
	Confidence        float64          `gorm:"not null;default:0" json:"confidence"`
	Status            string           `gorm:"size:40;not null;default:'suggested'" json:"status"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
	Responses         []BridgeResponse `gorm:"foreignKey:BridgeID" json:"responses,omitempty"`
}

func (b *Bridge) BeforeCreate(tx *gorm.DB) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return nil
}

// BridgeResponse is the actual perspective offered by the recommended person.
type BridgeResponse struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	BridgeID    uuid.UUID `gorm:"type:uuid;not null;index" json:"bridge_id"`
	ResponderID uuid.UUID `gorm:"type:uuid;not null;index" json:"responder_id"`
	Responder   User      `gorm:"foreignKey:ResponderID" json:"responder,omitempty"`
	Body        string    `gorm:"type:text;not null" json:"body"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (r *BridgeResponse) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}

// HelpSignal records post-interaction quality feedback for the matching loop.
type HelpSignal struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	BridgeID  uuid.UUID `gorm:"type:uuid;not null;index" json:"bridge_id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	Kind      string    `gorm:"size:40;not null" json:"kind"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *HelpSignal) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

// TrustProfile is the mentor/peer side of the graph: proof, topics, and availability.
type TrustProfile struct {
	UserID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	User            User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Topics          string    `gorm:"type:text;not null;default:''" json:"topics"`
	LivedExperience string    `gorm:"type:text;not null;default:''" json:"lived_experience"`
	Availability    string    `gorm:"size:40;not null;default:'async'" json:"availability"`
	HelpedCount     int       `gorm:"not null;default:0" json:"helped_count"`
	ResponseQuality float64   `gorm:"not null;default:0" json:"response_quality"`
	ExpertiseVector *string   `gorm:"type:vector(1024)" json:"-"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// HelpSession is a live micro-room centered on an advice intent, not a hashtag feed.
type HelpSession struct {
	ID          uuid.UUID           `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	Title       string              `gorm:"size:160;not null" json:"title"`
	Intent      string              `gorm:"size:120;not null;default:''" json:"intent"`
	Description string              `gorm:"type:text;not null;default:''" json:"description"`
	MemberCount int64               `gorm:"-" json:"member_count"`
	ExpiresAt   time.Time           `gorm:"index" json:"expires_at"`
	CreatedAt   time.Time           `json:"created_at"`
	Members     []HelpSessionMember `gorm:"foreignKey:SessionID" json:"members,omitempty"`
}

func (s *HelpSession) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

type HelpSessionMember struct {
	SessionID uuid.UUID `gorm:"type:uuid;primaryKey" json:"session_id"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey" json:"user_id"`
	User      User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	JoinedAt  time.Time `json:"joined_at"`
}
