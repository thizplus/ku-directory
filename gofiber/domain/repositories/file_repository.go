package repositories

import (
	"context"
	"gofiber-template/domain/models"
	"github.com/google/uuid"
)

type FileRepository interface {
	Create(ctx context.Context, file *models.File) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.File, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*models.File, error)
	Update(ctx context.Context, id uuid.UUID, file *models.File) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]*models.File, error)
	Count(ctx context.Context) (int64, error)
	CountByUserID(ctx context.Context, userID uuid.UUID) (int64, error)
}