package repositories

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gofiber-template/domain/models"
)

type SharedFolderRepository interface {
	// SharedFolder CRUD
	Create(ctx context.Context, folder *models.SharedFolder) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.SharedFolder, error)
	GetByDriveFolderID(ctx context.Context, driveFolderID string) (*models.SharedFolder, error)
	GetByWebhookToken(ctx context.Context, token string) (*models.SharedFolder, error)
	GetAll(ctx context.Context) ([]models.SharedFolder, error)
	GetAllNeedingSync(ctx context.Context) ([]models.SharedFolder, error)
	Update(ctx context.Context, id uuid.UUID, folder *models.SharedFolder) error
	UpdateSyncStatus(ctx context.Context, id uuid.UUID, status models.SyncStatus, lastError string) error
	UpdateTokens(ctx context.Context, id uuid.UUID, accessToken, refreshToken string, expiry *time.Time, ownerID uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error

	// User folder access
	AddUserAccess(ctx context.Context, access *models.UserFolderAccess) error
	RemoveUserAccess(ctx context.Context, userID, folderID uuid.UUID) error
	GetUserAccess(ctx context.Context, userID, folderID uuid.UUID) (*models.UserFolderAccess, error)
	GetUsersByFolder(ctx context.Context, folderID uuid.UUID) ([]models.User, error)
	GetFoldersByUser(ctx context.Context, userID uuid.UUID) ([]models.SharedFolder, error)
	HasUserAccess(ctx context.Context, userID, folderID uuid.UUID) (bool, error)

	// Count
	CountUsers(ctx context.Context, folderID uuid.UUID) (int64, error)
}
