package dto

import (
	"time"

	"github.com/google/uuid"
	"gofiber-template/domain/models"
)

// SubFolderInfo represents a sub-folder within a shared folder
type SubFolderInfo struct {
	Path       string `json:"path"`
	Name       string `json:"name"`
	PhotoCount int64  `json:"photo_count"`
}

// SharedFolderResponse is the DTO for shared folder API responses
type SharedFolderResponse struct {
	ID              uuid.UUID       `json:"id"`
	DriveFolderID   string          `json:"drive_folder_id"`
	DriveFolderName string          `json:"drive_folder_name"`
	DriveFolderPath string          `json:"drive_folder_path"`
	Description     string          `json:"description,omitempty"`
	SyncStatus      string          `json:"sync_status"`
	LastSyncAt      *time.Time      `json:"last_sync_at,omitempty"`
	LastSyncError   string          `json:"last_sync_error,omitempty"`
	PhotoCount      int64           `json:"photo_count"`
	UserCount       int64           `json:"user_count"`
	Children        []SubFolderInfo `json:"children,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`

	// Webhook status
	WebhookStatus string     `json:"webhook_status"`           // "active", "expiring", "expired", "inactive"
	WebhookExpiry *time.Time `json:"webhook_expiry,omitempty"` // When webhook expires
}

// AddFolderRequest is the request for adding a new folder
type AddFolderRequest struct {
	DriveFolderID    string `json:"drive_folder_id" validate:"required"`
	DriveResourceKey string `json:"drive_resource_key,omitempty"` // For older shared folders (pre-2021)
}

// SharedFolderListResponse is the response for listing folders
type SharedFolderListResponse struct {
	Folders []SharedFolderResponse `json:"folders"`
}

// SharedFolderToResponse converts a SharedFolder model to response DTO
func SharedFolderToResponse(folder *models.SharedFolder, photoCount, userCount int64) *SharedFolderResponse {
	if folder == nil {
		return nil
	}

	// Calculate webhook status
	webhookStatus := calculateWebhookStatus(folder)

	return &SharedFolderResponse{
		ID:              folder.ID,
		DriveFolderID:   folder.DriveFolderID,
		DriveFolderName: folder.DriveFolderName,
		DriveFolderPath: folder.DriveFolderPath,
		Description:     folder.Description,
		SyncStatus:      string(folder.SyncStatus),
		LastSyncAt:      folder.LastSyncedAt,
		LastSyncError:   folder.LastError,
		PhotoCount:      photoCount,
		UserCount:       userCount,
		CreatedAt:       folder.CreatedAt,
		WebhookStatus:   webhookStatus,
		WebhookExpiry:   folder.WebhookExpiry,
	}
}

// calculateWebhookStatus determines the webhook status based on expiry time
func calculateWebhookStatus(folder *models.SharedFolder) string {
	// No webhook registered
	if folder.WebhookChannelID == "" || folder.WebhookExpiry == nil {
		return "inactive"
	}

	now := time.Now()
	expiry := *folder.WebhookExpiry

	// Already expired
	if expiry.Before(now) {
		return "expired"
	}

	// Expiring soon (within 48 hours)
	if expiry.Before(now.Add(48 * time.Hour)) {
		return "expiring"
	}

	// Active and healthy
	return "active"
}
