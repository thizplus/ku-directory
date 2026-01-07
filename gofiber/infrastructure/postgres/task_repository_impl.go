package postgres

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
)

type TaskRepositoryImpl struct {
	db *gorm.DB
}

func NewTaskRepository(db *gorm.DB) repositories.TaskRepository {
	return &TaskRepositoryImpl{db: db}
}

func (r *TaskRepositoryImpl) Create(ctx context.Context, task *models.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *TaskRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.Task, error) {
	var task models.Task
	err := r.db.WithContext(ctx).Preload("User").Where("id = ?", id).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *TaskRepositoryImpl) GetByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*models.Task, error) {
	var tasks []*models.Task
	err := r.db.WithContext(ctx).Preload("User").Where("user_id = ?", userID).Offset(offset).Limit(limit).Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepositoryImpl) Update(ctx context.Context, id uuid.UUID, task *models.Task) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Updates(task).Error
}

func (r *TaskRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Task{}).Error
}

func (r *TaskRepositoryImpl) List(ctx context.Context, offset, limit int) ([]*models.Task, error) {
	var tasks []*models.Task
	err := r.db.WithContext(ctx).Preload("User").Offset(offset).Limit(limit).Find(&tasks).Error
	return tasks, err
}

func (r *TaskRepositoryImpl) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Task{}).Count(&count).Error
	return count, err
}

func (r *TaskRepositoryImpl) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Task{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}