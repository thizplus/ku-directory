package postgres

import (
	"context"
	"time"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
)

type JobRepositoryImpl struct {
	db *gorm.DB
}

func NewJobRepository(db *gorm.DB) repositories.JobRepository {
	return &JobRepositoryImpl{db: db}
}

func (r *JobRepositoryImpl) Create(ctx context.Context, job *models.Job) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *JobRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	var job models.Job
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *JobRepositoryImpl) GetByName(ctx context.Context, name string) (*models.Job, error) {
	var job models.Job
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *JobRepositoryImpl) GetActiveJobs(ctx context.Context) ([]*models.Job, error) {
	var jobs []*models.Job
	err := r.db.WithContext(ctx).Where("is_active = ?", true).Find(&jobs).Error
	return jobs, err
}

func (r *JobRepositoryImpl) Update(ctx context.Context, id uuid.UUID, job *models.Job) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Updates(job).Error
}

func (r *JobRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Job{}).Error
}

func (r *JobRepositoryImpl) List(ctx context.Context, offset, limit int) ([]*models.Job, error) {
	var jobs []*models.Job
	err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&jobs).Error
	return jobs, err
}

func (r *JobRepositoryImpl) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Job{}).Count(&count).Error
	return count, err
}

func (r *JobRepositoryImpl) UpdateLastRun(ctx context.Context, id uuid.UUID, lastRun *time.Time) error {
	return r.db.WithContext(ctx).Model(&models.Job{}).Where("id = ?", id).Update("last_run", lastRun).Error
}

func (r *JobRepositoryImpl) UpdateNextRun(ctx context.Context, id uuid.UUID, nextRun *time.Time) error {
	return r.db.WithContext(ctx).Model(&models.Job{}).Where("id = ?", id).Update("next_run", nextRun).Error
}