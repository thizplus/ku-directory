package repositories

import (
	"context"

	"github.com/google/uuid"
	"gofiber-template/domain/models"
)

type ActivityLogRepository interface {
	// Create a new activity log
	Create(ctx context.Context, log *models.ActivityLog) error

	// Get logs by folder with pagination
	GetByFolder(ctx context.Context, folderID uuid.UUID, offset, limit int) ([]models.ActivityLog, int64, error)

	// Get logs by folder and type
	GetByFolderAndType(ctx context.Context, folderID uuid.UUID, activityType models.ActivityType, offset, limit int) ([]models.ActivityLog, int64, error)

	// Get recent logs across all folders (for admin)
	GetRecent(ctx context.Context, limit int) ([]models.ActivityLog, error)

	// Delete old logs (cleanup)
	DeleteOlderThan(ctx context.Context, days int) (int64, error)
}
