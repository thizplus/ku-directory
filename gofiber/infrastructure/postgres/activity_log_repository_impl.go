package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
)

type ActivityLogRepositoryImpl struct {
	db *gorm.DB
}

func NewActivityLogRepository(db *gorm.DB) repositories.ActivityLogRepository {
	return &ActivityLogRepositoryImpl{db: db}
}

func (r *ActivityLogRepositoryImpl) Create(ctx context.Context, log *models.ActivityLog) error {
	if log.ID == uuid.Nil {
		log.ID = uuid.New()
	}
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}
	return r.db.WithContext(ctx).Create(log).Error
}

func (r *ActivityLogRepositoryImpl) GetByFolder(ctx context.Context, folderID uuid.UUID, offset, limit int) ([]models.ActivityLog, int64, error) {
	var logs []models.ActivityLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ActivityLog{}).Where("shared_folder_id = ?", folderID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error

	return logs, total, err
}

func (r *ActivityLogRepositoryImpl) GetByFolderAndType(ctx context.Context, folderID uuid.UUID, activityType models.ActivityType, offset, limit int) ([]models.ActivityLog, int64, error) {
	var logs []models.ActivityLog
	var total int64

	query := r.db.WithContext(ctx).Model(&models.ActivityLog{}).
		Where("shared_folder_id = ?", folderID).
		Where("activity_type = ?", activityType)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&logs).Error

	return logs, total, err
}

func (r *ActivityLogRepositoryImpl) GetRecent(ctx context.Context, limit int) ([]models.ActivityLog, error) {
	var logs []models.ActivityLog
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error

	return logs, err
}

func (r *ActivityLogRepositoryImpl) DeleteOlderThan(ctx context.Context, days int) (int64, error) {
	threshold := time.Now().AddDate(0, 0, -days)
	result := r.db.WithContext(ctx).
		Where("created_at < ?", threshold).
		Delete(&models.ActivityLog{})

	return result.RowsAffected, result.Error
}
