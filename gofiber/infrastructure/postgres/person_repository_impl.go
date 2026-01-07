package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
)

type PersonRepositoryImpl struct {
	db *gorm.DB
}

func NewPersonRepository(db *gorm.DB) repositories.PersonRepository {
	return &PersonRepositoryImpl{db: db}
}

func (r *PersonRepositoryImpl) Create(ctx context.Context, person *models.Person) error {
	return r.db.WithContext(ctx).Create(person).Error
}

func (r *PersonRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.Person, error) {
	var person models.Person
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&person).Error
	if err != nil {
		return nil, err
	}
	return &person, nil
}

func (r *PersonRepositoryImpl) GetByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]models.Person, int64, error) {
	var persons []models.Person
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Person{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("name ASC").
		Offset(offset).
		Limit(limit).
		Find(&persons).Error

	return persons, total, err
}

func (r *PersonRepositoryImpl) Update(ctx context.Context, id uuid.UUID, person *models.Person) error {
	person.UpdatedAt = time.Now()
	return r.db.WithContext(ctx).Where("id = ?", id).Updates(person).Error
}

func (r *PersonRepositoryImpl) UpdateFaceCount(ctx context.Context, id uuid.UUID, count int) error {
	return r.db.WithContext(ctx).
		Model(&models.Person{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"face_count": count,
			"updated_at": time.Now(),
		}).Error
}

func (r *PersonRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Person{}).Error
}

func (r *PersonRepositoryImpl) Count(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Person{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}
