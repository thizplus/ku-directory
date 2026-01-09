package models

import (
	"time"

	"github.com/google/uuid"
)

type ActivityType string

const (
	// Sync activities
	ActivitySyncStarted   ActivityType = "sync_started"
	ActivitySyncCompleted ActivityType = "sync_completed"
	ActivitySyncFailed    ActivityType = "sync_failed"

	// Photo activities
	ActivityPhotosAdded    ActivityType = "photos_added"
	ActivityPhotosTrashed  ActivityType = "photos_trashed"
	ActivityPhotosRestored ActivityType = "photos_restored"
	ActivityPhotosDeleted  ActivityType = "photos_deleted"  // Permanent delete
	ActivityPhotoRenamed   ActivityType = "photo_renamed"   // Name changed
	ActivityPhotoMoved     ActivityType = "photo_moved"     // Moved to different folder
	ActivityPhotoUpdated   ActivityType = "photo_updated"   // Content/metadata updated

	// Folder activities
	ActivityFolderTrashed  ActivityType = "folder_trashed"
	ActivityFolderRestored ActivityType = "folder_restored"
	ActivityFolderRenamed  ActivityType = "folder_renamed" // Name changed, same location
	ActivityFolderMoved    ActivityType = "folder_moved"   // Location changed
	ActivityFolderDeleted  ActivityType = "folder_deleted" // Permanent delete

	// Webhook activities
	ActivityWebhookReceived ActivityType = "webhook_received"
	ActivityWebhookRenewed  ActivityType = "webhook_renewed"
	ActivityWebhookExpired  ActivityType = "webhook_expired"

	// Error activities
	ActivityTokenExpired ActivityType = "token_expired"
	ActivitySyncError    ActivityType = "sync_error"
)

// ActivityLog stores all sync activities for audit and debugging
type ActivityLog struct {
	ID             uuid.UUID    `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SharedFolderID uuid.UUID    `gorm:"type:uuid;not null;index"`
	ActivityType   ActivityType `gorm:"type:varchar(50);not null;index"`
	Message        string       `gorm:"type:text"`  // Human-readable message (Thai)
	Details        string       `gorm:"type:jsonb"` // Structured details as JSON string
	RawData        string       `gorm:"type:jsonb"` // Raw data from Google Drive as JSON string
	CreatedAt      time.Time    `gorm:"index"`

	// Relations
	SharedFolder SharedFolder `gorm:"foreignKey:SharedFolderID"`
}

func (ActivityLog) TableName() string {
	return "activity_logs"
}

// ActivityDetails is a helper struct for common activity details
type ActivityDetails struct {
	// Counts
	Count         int `json:"count,omitempty"`
	TotalNew      int `json:"total_new,omitempty"`
	TotalUpdated  int `json:"total_updated,omitempty"`
	TotalDeleted  int `json:"total_deleted,omitempty"`
	TotalFailed   int `json:"total_failed,omitempty"`
	TotalTrashed  int `json:"total_trashed,omitempty"`
	TotalRestored int `json:"total_restored,omitempty"`

	// File/Folder info
	FileNames      []string `json:"file_names,omitempty"`
	FolderName     string   `json:"folder_name,omitempty"`
	FolderPath     string   `json:"folder_path,omitempty"`
	OldFolderName  string   `json:"old_folder_name,omitempty"`
	NewFolderName  string   `json:"new_folder_name,omitempty"`
	DriveFileID    string   `json:"drive_file_id,omitempty"`
	DriveFolderID  string   `json:"drive_folder_id,omitempty"`

	// Sync info
	JobID         string `json:"job_id,omitempty"`
	IsIncremental bool   `json:"is_incremental,omitempty"`
	DurationMs    int64  `json:"duration_ms,omitempty"`

	// Error info
	ErrorMessage string `json:"error_message,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`

	// Webhook info
	ChannelID     string `json:"channel_id,omitempty"`
	ResourceState string `json:"resource_state,omitempty"`
}
