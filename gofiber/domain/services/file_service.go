package services

import (
	"context"
	"gofiber-template/domain/dto"
	"gofiber-template/domain/models"
	"mime/multipart"

	"github.com/google/uuid"
)

type FileService interface {
	UploadFile(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader, options *dto.UploadFileRequest) (*models.File, error)
	GetFile(ctx context.Context, fileID uuid.UUID) (*models.File, error)
	GetUserFiles(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*models.File, int64, error)
	DeleteFile(ctx context.Context, fileID uuid.UUID) error
	ListFiles(ctx context.Context, offset, limit int) ([]*models.File, int64, error)
}
