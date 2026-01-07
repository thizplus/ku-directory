package services

import (
	"context"

	"github.com/google/uuid"
	"gofiber-template/domain/models"
)

// DriveFolder represents a folder from Google Drive
type DriveFolder struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	ParentID string `json:"parentId"`
}

// DownloadProgress represents download progress for a file
type DownloadProgress struct {
	Current  int    `json:"current"`
	Total    int    `json:"total"`
	FileName string `json:"fileName"`
}

// DownloadProgressCallback is called for each file downloaded
type DownloadProgressCallback func(progress DownloadProgress)

// DriveService handles Google Drive operations
type DriveService interface {
	// OAuth
	GetAuthURL(state string) string
	HandleCallback(ctx context.Context, userID uuid.UUID, code string) error
	IsConnected(ctx context.Context, userID uuid.UUID) bool
	Disconnect(ctx context.Context, userID uuid.UUID) error

	// Folders
	ListFolders(ctx context.Context, userID uuid.UUID, parentID string) ([]DriveFolder, error)
	SetRootFolder(ctx context.Context, userID uuid.UUID, folderID string) error
	GetRootFolder(ctx context.Context, userID uuid.UUID) (string, error)
	GetRootFolderInfo(ctx context.Context, userID uuid.UUID) (*DriveFolder, error)

	// Sync
	StartSync(ctx context.Context, userID uuid.UUID) (*models.SyncJob, error)
	GetSyncStatus(ctx context.Context, userID uuid.UUID) (*models.SyncJob, error)

	// Photos
	GetPhotos(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Photo, int64, error)
	GetPhotosByFolder(ctx context.Context, userID uuid.UUID, folderPath string, page, limit int) ([]models.Photo, int64, error)
	GetPhotosByFolderId(ctx context.Context, userID uuid.UUID, folderId string, page, limit int) ([]models.Photo, int64, error)
	SearchPhotos(ctx context.Context, userID uuid.UUID, searchQuery string, page, limit int) ([]models.Photo, int64, error)
	GetPhotoThumbnail(ctx context.Context, userID uuid.UUID, driveFileID string, size int) ([]byte, string, error)
	DownloadPhotosAsZip(ctx context.Context, userID uuid.UUID, driveFileIDs []string, onProgress DownloadProgressCallback) ([]byte, error)

	// Webhook
	HandleWebhook(ctx context.Context, channelID, resourceID, resourceState, token string) error
}
