package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
)

type SyncJobRepositoryImpl struct {
	db *gorm.DB
}

func NewSyncJobRepository(db *gorm.DB) repositories.SyncJobRepository {
	return &SyncJobRepositoryImpl{db: db}
}

func (r *SyncJobRepositoryImpl) Create(ctx context.Context, job *models.SyncJob) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *SyncJobRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.SyncJob, error) {
	var job models.SyncJob
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *SyncJobRepositoryImpl) GetByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]models.SyncJob, int64, error) {
	var jobs []models.SyncJob
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.SyncJob{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&jobs).Error

	return jobs, total, err
}

func (r *SyncJobRepositoryImpl) GetLatestByUserAndType(ctx context.Context, userID uuid.UUID, jobType models.SyncJobType) (*models.SyncJob, error) {
	var job models.SyncJob
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND job_type = ?", userID, jobType).
		Order("created_at DESC").
		First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *SyncJobRepositoryImpl) GetPendingJobs(ctx context.Context, jobType models.SyncJobType, limit int) ([]models.SyncJob, error) {
	var jobs []models.SyncJob
	err := r.db.WithContext(ctx).
		Where("job_type = ? AND status = ?", jobType, models.SyncJobStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&jobs).Error

	return jobs, err
}

func (r *SyncJobRepositoryImpl) Update(ctx context.Context, id uuid.UUID, job *models.SyncJob) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Updates(job).Error
}

func (r *SyncJobRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status models.SyncJobStatus) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if status == models.SyncJobStatusRunning {
		now := time.Now()
		updates["started_at"] = &now
	} else if status == models.SyncJobStatusCompleted || status == models.SyncJobStatusFailed {
		now := time.Now()
		updates["completed_at"] = &now
	}

	return r.db.WithContext(ctx).Model(&models.SyncJob{}).Where("id = ?", id).Updates(updates).Error
}

func (r *SyncJobRepositoryImpl) UpdateProgress(ctx context.Context, id uuid.UUID, processed, failed int) error {
	updates := map[string]interface{}{
		"processed_items": processed,
		"failed_items":    failed,
		"updated_at":      time.Now(),
	}
	return r.db.WithContext(ctx).Model(&models.SyncJob{}).Where("id = ?", id).Updates(updates).Error
}

func (r *SyncJobRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.SyncJob{}).Error
}
