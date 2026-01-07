package services

import (
	"context"
	"gofiber-template/domain/dto"
	"gofiber-template/domain/models"
	"github.com/google/uuid"
)

type JobService interface {
	CreateJob(ctx context.Context, req *dto.CreateJobRequest) (*models.Job, error)
	GetJob(ctx context.Context, jobID uuid.UUID) (*models.Job, error)
	UpdateJob(ctx context.Context, jobID uuid.UUID, req *dto.UpdateJobRequest) (*models.Job, error)
	DeleteJob(ctx context.Context, jobID uuid.UUID) error
	ListJobs(ctx context.Context, offset, limit int) ([]*models.Job, int64, error)
	StartJob(ctx context.Context, jobID uuid.UUID) error
	StopJob(ctx context.Context, jobID uuid.UUID) error
	ExecuteJob(ctx context.Context, job *models.Job) error
}