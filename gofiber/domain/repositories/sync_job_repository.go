package repositories

import (
	"context"

	"github.com/google/uuid"
	"gofiber-template/domain/models"
)

type SyncJobRepository interface {
	Create(ctx context.Context, job *models.SyncJob) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.SyncJob, error)
	GetByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]models.SyncJob, int64, error)
	GetLatestByUserAndType(ctx context.Context, userID uuid.UUID, jobType models.SyncJobType) (*models.SyncJob, error)
	GetPendingJobs(ctx context.Context, jobType models.SyncJobType, limit int) ([]models.SyncJob, error)
	HasPendingOrRunningJobForFolder(ctx context.Context, folderID uuid.UUID) (bool, error)
	Update(ctx context.Context, id uuid.UUID, job *models.SyncJob) error
	UpdateStatus(ctx context.Context, id uuid.UUID, status models.SyncJobStatus) error
	UpdateProgress(ctx context.Context, id uuid.UUID, processed, failed int) error
	Delete(ctx context.Context, id uuid.UUID) error
}
