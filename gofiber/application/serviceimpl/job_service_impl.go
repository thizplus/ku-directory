package serviceimpl

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gofiber-template/domain/dto"
	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"gofiber-template/pkg/logger"
	"gofiber-template/pkg/scheduler"
)

type JobServiceImpl struct {
	jobRepo   repositories.JobRepository
	scheduler scheduler.EventScheduler
}

func NewJobService(jobRepo repositories.JobRepository, scheduler scheduler.EventScheduler) services.JobService {
	return &JobServiceImpl{
		jobRepo:   jobRepo,
		scheduler: scheduler,
	}
}

func (s *JobServiceImpl) CreateJob(ctx context.Context, req *dto.CreateJobRequest) (*models.Job, error) {
	if err := scheduler.ValidateCronExpression(req.CronExpr); err != nil {
		return nil, fmt.Errorf("invalid cron expression: %v", err)
	}

	existingJob, _ := s.jobRepo.GetByName(ctx, req.Name)
	if existingJob != nil {
		return nil, errors.New("job with this name already exists")
	}

	nextRun, err := scheduler.GetNextRunTime(req.CronExpr)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate next run time: %v", err)
	}

	job := &models.Job{
		ID:        uuid.New(),
		Name:      req.Name,
		CronExpr:  req.CronExpr,
		Payload:   req.Payload,
		Status:    "active",
		NextRun:   nextRun,
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.jobRepo.Create(ctx, job)
	if err != nil {
		return nil, err
	}

	err = s.scheduler.AddJob(job.ID.String(), req.CronExpr, func() {
		s.ExecuteJob(context.Background(), job)
	})
	if err != nil {
		s.jobRepo.Delete(ctx, job.ID)
		return nil, fmt.Errorf("failed to schedule job: %v", err)
	}

	return job, nil
}

func (s *JobServiceImpl) GetJob(ctx context.Context, jobID uuid.UUID) (*models.Job, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, errors.New("job not found")
	}
	return job, nil
}

func (s *JobServiceImpl) UpdateJob(ctx context.Context, jobID uuid.UUID, req *dto.UpdateJobRequest) (*models.Job, error) {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return nil, errors.New("job not found")
	}

	needsReschedule := false

	if req.Name != "" {
		job.Name = req.Name
	}
	if req.CronExpr != "" {
		if err := scheduler.ValidateCronExpression(req.CronExpr); err != nil {
			return nil, fmt.Errorf("invalid cron expression: %v", err)
		}
		job.CronExpr = req.CronExpr
		needsReschedule = true
	}
	if req.Payload != "" {
		job.Payload = req.Payload
	}
	if req.IsActive != job.IsActive {
		job.IsActive = req.IsActive
		needsReschedule = true
	}

	if needsReschedule {
		s.scheduler.RemoveJob(jobID.String())
		if job.IsActive {
			nextRun, err := scheduler.GetNextRunTime(job.CronExpr)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate next run time: %v", err)
			}
			job.NextRun = nextRun

			err = s.scheduler.AddJob(jobID.String(), job.CronExpr, func() {
				s.ExecuteJob(context.Background(), job)
			})
			if err != nil {
				return nil, fmt.Errorf("failed to reschedule job: %v", err)
			}
		}
	}

	job.UpdatedAt = time.Now()

	err = s.jobRepo.Update(ctx, jobID, job)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func (s *JobServiceImpl) DeleteJob(ctx context.Context, jobID uuid.UUID) error {
	_, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return errors.New("job not found")
	}

	s.scheduler.RemoveJob(jobID.String())

	return s.jobRepo.Delete(ctx, jobID)
}

func (s *JobServiceImpl) ListJobs(ctx context.Context, offset, limit int) ([]*models.Job, int64, error) {
	jobs, err := s.jobRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.jobRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return jobs, count, nil
}

func (s *JobServiceImpl) StartJob(ctx context.Context, jobID uuid.UUID) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return errors.New("job not found")
	}

	if job.IsActive {
		return errors.New("job is already active")
	}

	job.IsActive = true
	job.UpdatedAt = time.Now()

	nextRun, err := scheduler.GetNextRunTime(job.CronExpr)
	if err != nil {
		return fmt.Errorf("failed to calculate next run time: %v", err)
	}
	job.NextRun = nextRun

	err = s.scheduler.AddJob(jobID.String(), job.CronExpr, func() {
		s.ExecuteJob(context.Background(), job)
	})
	if err != nil {
		return fmt.Errorf("failed to start job: %v", err)
	}

	return s.jobRepo.Update(ctx, jobID, job)
}

func (s *JobServiceImpl) StopJob(ctx context.Context, jobID uuid.UUID) error {
	job, err := s.jobRepo.GetByID(ctx, jobID)
	if err != nil {
		return errors.New("job not found")
	}

	if !job.IsActive {
		return errors.New("job is already inactive")
	}

	job.IsActive = false
	job.UpdatedAt = time.Now()

	s.scheduler.RemoveJob(jobID.String())

	return s.jobRepo.Update(ctx, jobID, job)
}

func (s *JobServiceImpl) ExecuteJob(ctx context.Context, job *models.Job) error {
	now := time.Now()

	logger.Sync("job_execute", "Executing job", map[string]interface{}{
		"job_id":   job.ID.String(),
		"job_name": job.Name,
		"time":     now.Format(time.RFC3339),
	})

	job.LastRun = &now
	job.Status = "running"

	nextRun, err := scheduler.GetNextRunTime(job.CronExpr)
	if err == nil {
		job.NextRun = nextRun
	}

	s.jobRepo.Update(ctx, job.ID, job)

	job.Status = "completed"
	job.UpdatedAt = time.Now()

	return s.jobRepo.Update(ctx, job.ID, job)
}
