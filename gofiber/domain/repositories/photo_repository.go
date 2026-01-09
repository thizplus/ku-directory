package repositories

import (
	"context"

	"github.com/google/uuid"
	"gofiber-template/domain/models"
)

type PhotoRepository interface {
	// CRUD
	Create(ctx context.Context, photo *models.Photo) error
	CreateBatch(ctx context.Context, photos []*models.Photo) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Photo, error)
	GetByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Photo, error)
	GetByDriveFileID(ctx context.Context, driveFileID string) (*models.Photo, error)
	Update(ctx context.Context, id uuid.UUID, photo *models.Photo) error
	UpdateFaceStatus(ctx context.Context, id uuid.UUID, status models.FaceProcessingStatus, faceCount int) error
	UpdateFolderPath(ctx context.Context, driveFolderID string, newPath string) (int64, error)
	Delete(ctx context.Context, id uuid.UUID) error

	// SharedFolder-based queries
	GetBySharedFolder(ctx context.Context, folderID uuid.UUID, offset, limit int) ([]models.Photo, int64, error)
	GetBySharedFolderAndPath(ctx context.Context, folderID uuid.UUID, folderPath string, offset, limit int) ([]models.Photo, int64, error)
	GetBySharedFolderAndDriveFolderID(ctx context.Context, folderID uuid.UUID, driveFolderID string, offset, limit int) ([]models.Photo, int64, error)
	SearchByFolderPathInSharedFolder(ctx context.Context, folderID uuid.UUID, searchQuery string, offset, limit int) ([]models.Photo, int64, error)
	GetFolderPathsInSharedFolder(ctx context.Context, folderID uuid.UUID) ([]string, error)
	CountBySharedFolder(ctx context.Context, folderID uuid.UUID) (int64, error)
	CountBySharedFolderAndFaceStatus(ctx context.Context, folderID uuid.UUID, status models.FaceProcessingStatus) (int64, error)

	// Multi-folder queries (for users with access to multiple folders)
	GetBySharedFolders(ctx context.Context, folderIDs []uuid.UUID, offset, limit int) ([]models.Photo, int64, error)
	GetBySharedFoldersAndPath(ctx context.Context, folderIDs []uuid.UUID, folderPath string, offset, limit int) ([]models.Photo, int64, error)

	// Face processing
	GetPendingFaceProcessing(ctx context.Context, folderID uuid.UUID, limit int) ([]models.Photo, error)
	GetByFaceStatus(ctx context.Context, status models.FaceProcessingStatus, limit int) ([]models.Photo, error)
	GetPendingBySharedFolders(ctx context.Context, folderIDs []uuid.UUID, limit int) ([]models.Photo, error)
	ResetFailedToPending(ctx context.Context, folderID *uuid.UUID) (int64, error)                      // Reset failed photos to pending, optionally by folder
	ResetStuckProcessingToPending(ctx context.Context, stuckThresholdMinutes int) (int64, error)     // Reset photos stuck in processing for too long

	// Soft delete (trash) operations
	SetTrashedByDriveFileID(ctx context.Context, driveFileID string, isTrashed bool) error
	SetTrashedByDriveFolderID(ctx context.Context, driveFolderID string, isTrashed bool) (int64, error)

	// Delete operations (hard delete)
	DeleteByDriveFileID(ctx context.Context, driveFileID string) error
	DeleteByDriveFolderID(ctx context.Context, driveFolderID string) (int64, error)
	DeleteNotInDriveIDsForFolder(ctx context.Context, folderID uuid.UUID, driveFileIDs []string) (int64, error)

	// Legacy methods (for migration/compatibility - can be removed later)
	GetByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]models.Photo, int64, error)
	GetByUserAndFolder(ctx context.Context, userID uuid.UUID, folderPath string, offset, limit int) ([]models.Photo, int64, error)
	GetByUserAndFolderId(ctx context.Context, userID uuid.UUID, folderId string, offset, limit int) ([]models.Photo, int64, error)
	SearchByFolderPath(ctx context.Context, userID uuid.UUID, searchQuery string, offset, limit int) ([]models.Photo, int64, error)
	DeleteByFolderID(ctx context.Context, userID uuid.UUID, folderID string) (int64, error)
	DeleteNotInDriveIDs(ctx context.Context, userID uuid.UUID, driveFileIDs []string) (int64, error)
	Count(ctx context.Context, userID uuid.UUID) (int64, error)
	CountByFaceStatus(ctx context.Context, userID uuid.UUID, status models.FaceProcessingStatus) (int64, error)
	GetFolderPaths(ctx context.Context, userID uuid.UUID) ([]string, error)

	// Shared folder methods (query by folder path prefix) - Legacy
	GetByFolderPathPrefix(ctx context.Context, pathPrefix string, offset, limit int) ([]models.Photo, int64, error)
	CountByFolderPathPrefix(ctx context.Context, pathPrefix string) (int64, error)
	CountByFolderPathPrefixAndFaceStatus(ctx context.Context, pathPrefix string, status models.FaceProcessingStatus) (int64, error)
}
