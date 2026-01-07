package repositories

import (
	"context"

	"github.com/google/uuid"

	"gofiber-template/domain/models"
)

type PersonRepository interface {
	Create(ctx context.Context, person *models.Person) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Person, error)
	GetByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]models.Person, int64, error)
	Update(ctx context.Context, id uuid.UUID, person *models.Person) error
	UpdateFaceCount(ctx context.Context, id uuid.UUID, count int) error
	Delete(ctx context.Context, id uuid.UUID) error
	Count(ctx context.Context, userID uuid.UUID) (int64, error)
}
