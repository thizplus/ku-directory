package models

import (
	"time"

	"github.com/google/uuid"
)

type SyncJobType string

const (
	SyncJobTypeDriveSync   SyncJobType = "drive_sync"    // Sync files from Google Drive
	SyncJobTypeFaceProcess SyncJobType = "face_process"  // Process faces in photos
)

type SyncJobStatus string

const (
	SyncJobStatusPending    SyncJobStatus = "pending"
	SyncJobStatusRunning    SyncJobStatus = "running"
	SyncJobStatusCompleted  SyncJobStatus = "completed"
	SyncJobStatusFailed     SyncJobStatus = "failed"
	SyncJobStatusCancelled  SyncJobStatus = "cancelled"
)

type SyncJob struct {
	ID     uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()" json:"id"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`

	// Job info
	JobType SyncJobType   `gorm:"not null;index" json:"job_type"`
	Status  SyncJobStatus `gorm:"default:'pending';index" json:"status"`

	// Progress tracking
	TotalItems     int `gorm:"default:0" json:"total_files"`     // Total items to process
	ProcessedItems int `gorm:"default:0" json:"processed_files"` // Items processed so far
	FailedItems    int `gorm:"default:0" json:"failed_files"`    // Items that failed

	// Timing
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`

	// Error info
	LastError string `gorm:"type:text" json:"last_error,omitempty"`

	// Metadata (JSON for additional job-specific data)
	Metadata string `gorm:"type:jsonb" json:"-"` // e.g., {"folder_id": "xxx", "page_token": "yyy"}

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// Relations
	User User `gorm:"foreignKey:UserID" json:"-"`
}

func (SyncJob) TableName() string {
	return "sync_jobs"
}

// DriveWebhookLog stores incoming webhook events from Google Drive
type DriveWebhookLog struct {
	ID     uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	// Webhook data
	ChannelID  string `gorm:"index"`
	ResourceID string
	EventType  string // "sync", "update", "trash", etc.

	// Processing
	Processed   bool `gorm:"default:false;index"`
	ProcessedAt *time.Time

	// Raw payload
	Payload string `gorm:"type:jsonb"`

	CreatedAt time.Time
}

func (DriveWebhookLog) TableName() string {
	return "drive_webhook_logs"
}
