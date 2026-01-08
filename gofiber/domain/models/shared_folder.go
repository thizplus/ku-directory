package models

import (
	"time"

	"github.com/google/uuid"
)

type SyncStatus string

const (
	SyncStatusIdle     SyncStatus = "idle"
	SyncStatusSyncing  SyncStatus = "syncing"
	SyncStatusError    SyncStatus = "error"
)

// SharedFolder represents a Google Drive folder that is synced by the server
type SharedFolder struct {
	ID               uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	DriveFolderID    string    `gorm:"uniqueIndex;not null"` // Google Drive folder ID
	DriveFolderName  string    `gorm:"not null"`             // Folder name
	DriveFolderPath  string    `gorm:"not null"`             // Full path (for display)
	DriveResourceKey string    // Resource key for older shared folders (pre-2021)
	Description      string    // Folder description from Google Drive

	// Webhook info
	WebhookChannelID  string     // Channel ID from Google
	WebhookResourceID string     // Resource ID from Google
	WebhookToken      string     `gorm:"uniqueIndex"` // Token for webhook verification
	WebhookExpiry     *time.Time // When webhook expires

	// Sync info
	PageToken    string     // Page token for incremental sync
	LastSyncedAt *time.Time // Last successful sync time
	SyncStatus   SyncStatus `gorm:"default:'idle'"` // Current sync status
	LastError    string     // Last error message (if any)

	// OAuth tokens (from user who added this folder)
	DriveAccessToken  string     // Google Drive access token
	DriveRefreshToken string     // Google Drive refresh token
	DriveTokenExpiry  *time.Time // Token expiry time
	TokenOwnerID      uuid.UUID  `gorm:"type:uuid"` // User who provided the tokens

	CreatedAt time.Time
	UpdatedAt time.Time

	// Relations
	Photos       []Photo            `gorm:"foreignKey:SharedFolderID"`
	UserAccesses []UserFolderAccess `gorm:"foreignKey:SharedFolderID"`
}

func (SharedFolder) TableName() string {
	return "shared_folders"
}

// UserFolderAccess represents a user's access to a shared folder
type UserFolderAccess struct {
	ID             uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index"`
	SharedFolderID uuid.UUID `gorm:"type:uuid;not null;index"`

	// User's root path within this folder (for sub-folder access)
	// Empty means access to entire folder
	RootPath string

	CreatedAt time.Time

	// Relations
	User         User         `gorm:"foreignKey:UserID"`
	SharedFolder SharedFolder `gorm:"foreignKey:SharedFolderID"`
}

func (UserFolderAccess) TableName() string {
	return "user_folder_access"
}

// Unique constraint: one access record per user per folder
func (UserFolderAccess) TableConstraints() string {
	return "UNIQUE(user_id, shared_folder_id)"
}
