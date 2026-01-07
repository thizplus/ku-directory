package postgres

import (
	"context"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
)

type FileRepositoryImpl struct {
	db *gorm.DB
}

func NewFileRepository(db *gorm.DB) repositories.FileRepository {
	return &FileRepositoryImpl{db: db}
}

func (r *FileRepositoryImpl) Create(ctx context.Context, file *models.File) error {
	return r.db.WithContext(ctx).Create(file).Error
}

func (r *FileRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.File, error) {
	var file models.File
	err := r.db.WithContext(ctx).Preload("User").Where("id = ?", id).First(&file).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

func (r *FileRepositoryImpl) GetByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*models.File, error) {
	var files []*models.File
	err := r.db.WithContext(ctx).Preload("User").Where("user_id = ?", userID).Offset(offset).Limit(limit).Find(&files).Error
	return files, err
}

func (r *FileRepositoryImpl) Update(ctx context.Context, id uuid.UUID, file *models.File) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Updates(file).Error
}

func (r *FileRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.File{}).Error
}

func (r *FileRepositoryImpl) List(ctx context.Context, offset, limit int) ([]*models.File, error) {
	var files []*models.File
	err := r.db.WithContext(ctx).Preload("User").Offset(offset).Limit(limit).Find(&files).Error
	return files, err
}

func (r *FileRepositoryImpl) Count(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.File{}).Count(&count).Error
	return count, err
}

func (r *FileRepositoryImpl) CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.File{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}