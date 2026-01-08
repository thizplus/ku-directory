package services

import (
	"context"

	"github.com/google/uuid"
	"gofiber-template/domain/models"
)

type SharedFolderService interface {
	// Folder management
	AddFolder(ctx context.Context, userID uuid.UUID, driveFolderID, resourceKey string, accessToken, refreshToken string) (*models.SharedFolder, error)
	GetUserFolders(ctx context.Context, userID uuid.UUID) ([]models.SharedFolder, error)
	GetFolderByID(ctx context.Context, userID uuid.UUID, folderID uuid.UUID) (*models.SharedFolder, error)
	RemoveUserAccess(ctx context.Context, userID uuid.UUID, folderID uuid.UUID) error

	// Sync operations
	TriggerSync(ctx context.Context, userID uuid.UUID, folderID uuid.UUID, forceFullSync bool) error
	GetSyncStatus(ctx context.Context, folderID uuid.UUID) (*models.SharedFolder, error)

	// Webhook handling
	HandleWebhook(ctx context.Context, channelID, resourceID, resourceState, token string) error

	// Register webhook for existing folder
	RegisterWebhook(ctx context.Context, userID uuid.UUID, folderID uuid.UUID) error

	// Webhook maintenance
	RenewExpiringWebhooks(ctx context.Context) (renewed int, failed int, err error)
}
