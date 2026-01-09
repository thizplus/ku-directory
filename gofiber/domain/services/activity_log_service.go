package services

import (
	"context"

	"github.com/google/uuid"
	"gofiber-template/domain/models"
)

type ActivityLogService interface {
	// GetByFolder returns activity logs for a folder with pagination
	GetByFolder(ctx context.Context, folderID uuid.UUID, page, limit int) ([]models.ActivityLog, int64, error)

	// GetByFolderAndType returns activity logs filtered by type
	GetByFolderAndType(ctx context.Context, folderID uuid.UUID, activityType models.ActivityType, page, limit int) ([]models.ActivityLog, int64, error)

	// GetRecent returns recent activity logs across all folders
	GetRecent(ctx context.Context, limit int) ([]models.ActivityLog, error)

	// Cleanup deletes old activity logs
	Cleanup(ctx context.Context, days int) (int64, error)
}
