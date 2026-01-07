package repositories

import (
	"context"
	"time"
	"gofiber-template/domain/models"
	"github.com/google/uuid"
)

type JobRepository interface {
	Create(ctx context.Context, job *models.Job) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Job, error)
	GetByName(ctx context.Context, name string) (*models.Job, error)
	GetActiveJobs(ctx context.Context) ([]*models.Job, error)
	Update(ctx context.Context, id uuid.UUID, job *models.Job) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*models.Job, error)
	Count(ctx context.Context) (int64, error)
	UpdateLastRun(ctx context.Context, id uuid.UUID, lastRun *time.Time) error
	UpdateNextRun(ctx context.Context, id uuid.UUID, nextRun *time.Time) error
}