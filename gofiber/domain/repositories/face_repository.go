package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"gofiber-template/domain/models"
)

type FaceRepository interface {
	Create(ctx context.Context, face *models.Face) error
	CreateBatch(ctx context.Context, faces []*models.Face) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Face, error)
	GetByPhoto(ctx context.Context, photoID uuid.UUID) ([]models.Face, error)
	GetByPerson(ctx context.Context, personID uuid.UUID) ([]models.Face, error)
	GetByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]models.Face, int64, error)

	// SharedFolder-based queries
	GetBySharedFolders(ctx context.Context, folderIDs []uuid.UUID, offset, limit int) ([]models.Face, int64, error)
	CountBySharedFolders(ctx context.Context, folderIDs []uuid.UUID) (int64, error)

	// Vector search - find similar faces
	SearchSimilar(ctx context.Context, userID uuid.UUID, embedding pgvector.Vector, limit int, threshold float64) ([]FaceSearchResult, error)
	SearchSimilarByFolderPathPrefix(ctx context.Context, pathPrefix string, embedding pgvector.Vector, limit int, threshold float64) ([]FaceSearchResult, error)
	SearchSimilarBySharedFolders(ctx context.Context, folderIDs []uuid.UUID, embedding pgvector.Vector, limit int, threshold float64) ([]FaceSearchResult, error)

	Update(ctx context.Context, id uuid.UUID, face *models.Face) error
	UpdatePersonID(ctx context.Context, id uuid.UUID, personID *uuid.UUID) error
	Delete(ctx context.Context, id uuid.UUID) error
	DeleteByPhoto(ctx context.Context, photoID uuid.UUID) error
	Count(ctx context.Context, userID uuid.UUID) (int64, error)
}

// FaceSearchResult represents a face search result with similarity score
type FaceSearchResult struct {
	Face       models.Face
	Photo      models.Photo
	Similarity float64 // Cosine similarity (0-1, higher is more similar)
}
