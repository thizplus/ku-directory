package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
)

type PhotoRepositoryImpl struct {
	db *gorm.DB
}

func NewPhotoRepository(db *gorm.DB) repositories.PhotoRepository {
	return &PhotoRepositoryImpl{db: db}
}

func (r *PhotoRepositoryImpl) Create(ctx context.Context, photo *models.Photo) error {
	return r.db.WithContext(ctx).Create(photo).Error
}

func (r *PhotoRepositoryImpl) CreateBatch(ctx context.Context, photos []*models.Photo) error {
	if len(photos) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(photos, 100).Error
}

func (r *PhotoRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.Photo, error) {
	var photo models.Photo
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&photo).Error
	if err != nil {
		return nil, err
	}
	return &photo, nil
}

func (r *PhotoRepositoryImpl) GetByIDs(ctx context.Context, ids []uuid.UUID) ([]models.Photo, error) {
	var photos []models.Photo
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&photos).Error
	if err != nil {
		return nil, err
	}
	return photos, nil
}

func (r *PhotoRepositoryImpl) GetByDriveFileID(ctx context.Context, driveFileID string) (*models.Photo, error) {
	var photo models.Photo
	err := r.db.WithContext(ctx).Where("drive_file_id = ?", driveFileID).First(&photo).Error
	if err != nil {
		return nil, err
	}
	return &photo, nil
}

func (r *PhotoRepositoryImpl) GetByUser(ctx context.Context, userID uuid.UUID, offset, limit int) ([]models.Photo, int64, error) {
	var photos []models.Photo
	var total int64

	// Get total count
	if err := r.db.WithContext(ctx).Model(&models.Photo{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("drive_created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&photos).Error

	return photos, total, err
}

func (r *PhotoRepositoryImpl) GetByUserAndFolder(ctx context.Context, userID uuid.UUID, folderPath string, offset, limit int) ([]models.Photo, int64, error) {
	var photos []models.Photo
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Photo{}).Where("user_id = ?", userID)
	if folderPath != "" {
		query = query.Where("drive_folder_path = ?", folderPath)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.
		Order("drive_created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&photos).Error

	return photos, total, err
}

func (r *PhotoRepositoryImpl) GetByUserAndFolderId(ctx context.Context, userID uuid.UUID, folderId string, offset, limit int) ([]models.Photo, int64, error) {
	var photos []models.Photo
	var total int64

	// Shared model: query by folder only, not by user_id
	// This allows all users with access to the folder to see shared photos
	query := r.db.WithContext(ctx).Model(&models.Photo{})
	if folderId != "" {
		query = query.Where("drive_folder_id = ?", folderId)
	} else {
		// Fallback: if no folder specified, filter by user's own photos
		query = query.Where("user_id = ?", userID)
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.
		Order("drive_created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&photos).Error

	return photos, total, err
}

func (r *PhotoRepositoryImpl) SearchByFolderPath(ctx context.Context, userID uuid.UUID, searchQuery string, offset, limit int) ([]models.Photo, int64, error) {
	var photos []models.Photo
	var total int64

	// Shared model: search by folder path without user_id filter
	query := r.db.WithContext(ctx).Model(&models.Photo{})
	if searchQuery != "" {
		// Case-insensitive search in folder path
		query = query.Where("LOWER(drive_folder_path) LIKE LOWER(?)", "%"+searchQuery+"%")
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.
		Order("drive_created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&photos).Error

	return photos, total, err
}

func (r *PhotoRepositoryImpl) GetPendingFaceProcessing(ctx context.Context, userID uuid.UUID, limit int) ([]models.Photo, error) {
	var photos []models.Photo
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND face_status = ?", userID, models.FaceStatusPending).
		Where("is_trashed = ?", false).
		Order("created_at ASC").
		Limit(limit).
		Find(&photos).Error

	return photos, err
}

func (r *PhotoRepositoryImpl) GetByFaceStatus(ctx context.Context, status models.FaceProcessingStatus, limit int) ([]models.Photo, error) {
	var photos []models.Photo
	err := r.db.WithContext(ctx).
		Where("face_status = ?", status).
		Where("is_trashed = ?", false).
		Order("created_at ASC").
		Limit(limit).
		Find(&photos).Error

	return photos, err
}

func (r *PhotoRepositoryImpl) GetPendingBySharedFolders(ctx context.Context, folderIDs []uuid.UUID, limit int) ([]models.Photo, error) {
	var photos []models.Photo
	err := r.db.WithContext(ctx).
		Where("shared_folder_id IN ?", folderIDs).
		Where("face_status = ?", models.FaceStatusPending).
		Where("is_trashed = ?", false).
		Order("created_at ASC").
		Limit(limit).
		Find(&photos).Error

	return photos, err
}

func (r *PhotoRepositoryImpl) Update(ctx context.Context, id uuid.UUID, photo *models.Photo) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Updates(photo).Error
}

func (r *PhotoRepositoryImpl) UpdateFaceStatus(ctx context.Context, id uuid.UUID, status models.FaceProcessingStatus, faceCount int) error {
	updates := map[string]interface{}{
		"face_status":       status,
		"face_count":        faceCount,
		"face_processed_at": time.Now(),
		"updated_at":        time.Now(),
	}
	return r.db.WithContext(ctx).Model(&models.Photo{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateFolderPath updates the folder path for all photos with the given drive_folder_id
func (r *PhotoRepositoryImpl) UpdateFolderPath(ctx context.Context, driveFolderID string, newPath string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Photo{}).
		Where("drive_folder_id = ?", driveFolderID).
		Updates(map[string]interface{}{
			"drive_folder_path": newPath,
			"updated_at":        time.Now(),
		})
	return result.RowsAffected, result.Error
}

func (r *PhotoRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.Photo{}).Error
}

// SetTrashedByDriveFileID sets the trashed status for a photo by its Drive file ID
func (r *PhotoRepositoryImpl) SetTrashedByDriveFileID(ctx context.Context, driveFileID string, isTrashed bool) error {
	updates := map[string]interface{}{
		"is_trashed": isTrashed,
		"updated_at": time.Now(),
	}
	if isTrashed {
		now := time.Now()
		updates["trashed_at"] = &now
	} else {
		updates["trashed_at"] = nil
	}
	return r.db.WithContext(ctx).Model(&models.Photo{}).
		Where("drive_file_id = ?", driveFileID).
		Updates(updates).Error
}

// SetTrashedByDriveFolderID sets the trashed status for all photos in a folder
// Only updates photos that actually need to change state (where is_trashed != target state)
func (r *PhotoRepositoryImpl) SetTrashedByDriveFolderID(ctx context.Context, driveFolderID string, isTrashed bool) (int64, error) {
	updates := map[string]interface{}{
		"is_trashed": isTrashed,
		"updated_at": time.Now(),
	}
	if isTrashed {
		now := time.Now()
		updates["trashed_at"] = &now
	} else {
		updates["trashed_at"] = nil
	}
	result := r.db.WithContext(ctx).Model(&models.Photo{}).
		Where("drive_folder_id = ?", driveFolderID).
		Where("is_trashed = ?", !isTrashed). // Only update photos that need state change
		Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *PhotoRepositoryImpl) DeleteByDriveFileID(ctx context.Context, driveFileID string) error {
	return r.db.WithContext(ctx).Where("drive_file_id = ?", driveFileID).Delete(&models.Photo{}).Error
}

func (r *PhotoRepositoryImpl) DeleteByFolderID(ctx context.Context, userID uuid.UUID, folderID string) (int64, error) {
	// Use transaction to handle cascading deletes
	var totalDeleted int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First, get the photo IDs in this folder
		var photoIDs []uuid.UUID
		if err := tx.Model(&models.Photo{}).Where("user_id = ? AND drive_folder_id = ?", userID, folderID).Pluck("id", &photoIDs).Error; err != nil {
			return err
		}

		if len(photoIDs) == 0 {
			return nil // Nothing to delete
		}

		// Delete faces associated with these photos first
		if err := tx.Where("photo_id IN ?", photoIDs).Delete(&models.Face{}).Error; err != nil {
			return err
		}

		// Now delete the photos
		result := tx.Where("user_id = ? AND drive_folder_id = ?", userID, folderID).Delete(&models.Photo{})
		if result.Error != nil {
			return result.Error
		}

		totalDeleted = result.RowsAffected
		return nil
	})

	return totalDeleted, err
}

func (r *PhotoRepositoryImpl) DeleteNotInDriveIDs(ctx context.Context, userID uuid.UUID, driveFileIDs []string) (int64, error) {
	// Use transaction to handle cascading deletes
	var totalDeleted int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// First, get the photo IDs that will be deleted
		var photoIDs []uuid.UUID
		var query *gorm.DB

		if len(driveFileIDs) == 0 {
			query = tx.Model(&models.Photo{}).Where("user_id = ?", userID).Pluck("id", &photoIDs)
		} else {
			query = tx.Model(&models.Photo{}).Where("user_id = ? AND drive_file_id NOT IN ?", userID, driveFileIDs).Pluck("id", &photoIDs)
		}

		if query.Error != nil {
			return query.Error
		}

		if len(photoIDs) == 0 {
			return nil // Nothing to delete
		}

		// Delete faces associated with these photos first
		if err := tx.Where("photo_id IN ?", photoIDs).Delete(&models.Face{}).Error; err != nil {
			return err
		}

		// Now delete the photos
		var result *gorm.DB
		if len(driveFileIDs) == 0 {
			result = tx.Where("user_id = ?", userID).Delete(&models.Photo{})
		} else {
			result = tx.Where("user_id = ? AND drive_file_id NOT IN ?", userID, driveFileIDs).Delete(&models.Photo{})
		}

		if result.Error != nil {
			return result.Error
		}

		totalDeleted = result.RowsAffected
		return nil
	})

	return totalDeleted, err
}

func (r *PhotoRepositoryImpl) Count(ctx context.Context, userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Photo{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func (r *PhotoRepositoryImpl) CountByFaceStatus(ctx context.Context, userID uuid.UUID, status models.FaceProcessingStatus) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Photo{}).
		Where("user_id = ? AND face_status = ?", userID, status).
		Count(&count).Error
	return count, err
}

func (r *PhotoRepositoryImpl) GetFolderPaths(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var paths []string
	err := r.db.WithContext(ctx).
		Model(&models.Photo{}).
		Where("user_id = ?", userID).
		Distinct("drive_folder_path").
		Pluck("drive_folder_path", &paths).Error

	return paths, err
}

// Shared folder methods

func (r *PhotoRepositoryImpl) GetByFolderPathPrefix(ctx context.Context, pathPrefix string, offset, limit int) ([]models.Photo, int64, error) {
	var photos []models.Photo
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Photo{})
	if pathPrefix != "" {
		query = query.Where("drive_folder_path LIKE ?", pathPrefix+"%")
	}

	// Get total count
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// Get paginated results
	err := query.
		Order("drive_created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&photos).Error

	return photos, total, err
}

func (r *PhotoRepositoryImpl) CountByFolderPathPrefix(ctx context.Context, pathPrefix string) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.Photo{})
	if pathPrefix != "" {
		query = query.Where("drive_folder_path LIKE ?", pathPrefix+"%")
	}
	err := query.Count(&count).Error
	return count, err
}

func (r *PhotoRepositoryImpl) CountByFolderPathPrefixAndFaceStatus(ctx context.Context, pathPrefix string, status models.FaceProcessingStatus) (int64, error) {
	var count int64
	query := r.db.WithContext(ctx).Model(&models.Photo{}).Where("face_status = ?", status)
	if pathPrefix != "" {
		query = query.Where("drive_folder_path LIKE ?", pathPrefix+"%")
	}
	err := query.Count(&count).Error
	return count, err
}

// ============================================
// SharedFolder-based methods (new)
// ============================================

func (r *PhotoRepositoryImpl) GetBySharedFolder(ctx context.Context, folderID uuid.UUID, offset, limit int) ([]models.Photo, int64, error) {
	var photos []models.Photo
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Photo{}).
		Where("shared_folder_id = ?", folderID).
		Where("is_trashed = ?", false)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("drive_created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&photos).Error

	return photos, total, err
}

func (r *PhotoRepositoryImpl) GetBySharedFolderAndPath(ctx context.Context, folderID uuid.UUID, folderPath string, offset, limit int) ([]models.Photo, int64, error) {
	var photos []models.Photo
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Photo{}).
		Where("shared_folder_id = ?", folderID).
		Where("is_trashed = ?", false)
	if folderPath != "" {
		query = query.Where("drive_folder_path = ?", folderPath)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("drive_created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&photos).Error

	return photos, total, err
}

func (r *PhotoRepositoryImpl) GetBySharedFolderAndDriveFolderID(ctx context.Context, folderID uuid.UUID, driveFolderID string, offset, limit int) ([]models.Photo, int64, error) {
	var photos []models.Photo
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Photo{}).
		Where("shared_folder_id = ?", folderID).
		Where("is_trashed = ?", false)
	if driveFolderID != "" {
		query = query.Where("drive_folder_id = ?", driveFolderID)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("drive_created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&photos).Error

	return photos, total, err
}

func (r *PhotoRepositoryImpl) SearchByFolderPathInSharedFolder(ctx context.Context, folderID uuid.UUID, searchQuery string, offset, limit int) ([]models.Photo, int64, error) {
	var photos []models.Photo
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Photo{}).
		Where("shared_folder_id = ?", folderID).
		Where("is_trashed = ?", false)
	if searchQuery != "" {
		query = query.Where("LOWER(drive_folder_path) LIKE LOWER(?)", "%"+searchQuery+"%")
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("drive_created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&photos).Error

	return photos, total, err
}

func (r *PhotoRepositoryImpl) GetFolderPathsInSharedFolder(ctx context.Context, folderID uuid.UUID) ([]string, error) {
	var paths []string
	err := r.db.WithContext(ctx).
		Model(&models.Photo{}).
		Where("shared_folder_id = ?", folderID).
		Where("is_trashed = ?", false).
		Distinct("drive_folder_path").
		Pluck("drive_folder_path", &paths).Error

	return paths, err
}

func (r *PhotoRepositoryImpl) CountBySharedFolder(ctx context.Context, folderID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Photo{}).
		Where("shared_folder_id = ?", folderID).
		Where("is_trashed = ?", false).
		Count(&count).Error
	return count, err
}

func (r *PhotoRepositoryImpl) CountBySharedFolderAndFaceStatus(ctx context.Context, folderID uuid.UUID, status models.FaceProcessingStatus) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Photo{}).
		Where("shared_folder_id = ? AND face_status = ?", folderID, status).
		Where("is_trashed = ?", false).
		Count(&count).Error
	return count, err
}

// Multi-folder queries

func (r *PhotoRepositoryImpl) GetBySharedFolders(ctx context.Context, folderIDs []uuid.UUID, offset, limit int) ([]models.Photo, int64, error) {
	var photos []models.Photo
	var total int64

	if len(folderIDs) == 0 {
		return photos, 0, nil
	}

	query := r.db.WithContext(ctx).Model(&models.Photo{}).
		Where("shared_folder_id IN ?", folderIDs).
		Where("is_trashed = ?", false)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("drive_created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&photos).Error

	return photos, total, err
}

func (r *PhotoRepositoryImpl) GetBySharedFoldersAndPath(ctx context.Context, folderIDs []uuid.UUID, folderPath string, offset, limit int) ([]models.Photo, int64, error) {
	var photos []models.Photo
	var total int64

	if len(folderIDs) == 0 {
		return photos, 0, nil
	}

	query := r.db.WithContext(ctx).Model(&models.Photo{}).
		Where("shared_folder_id IN ?", folderIDs).
		Where("is_trashed = ?", false)
	if folderPath != "" {
		query = query.Where("drive_folder_path = ?", folderPath)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.
		Order("drive_created_at DESC").
		Offset(offset).
		Limit(limit).
		Find(&photos).Error

	return photos, total, err
}

// Delete operations for SharedFolder

func (r *PhotoRepositoryImpl) DeleteByDriveFolderID(ctx context.Context, driveFolderID string) (int64, error) {
	var totalDeleted int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var photoIDs []uuid.UUID
		if err := tx.Model(&models.Photo{}).Where("drive_folder_id = ?", driveFolderID).Pluck("id", &photoIDs).Error; err != nil {
			return err
		}

		if len(photoIDs) == 0 {
			return nil
		}

		if err := tx.Where("photo_id IN ?", photoIDs).Delete(&models.Face{}).Error; err != nil {
			return err
		}

		result := tx.Where("drive_folder_id = ?", driveFolderID).Delete(&models.Photo{})
		if result.Error != nil {
			return result.Error
		}

		totalDeleted = result.RowsAffected
		return nil
	})

	return totalDeleted, err
}

func (r *PhotoRepositoryImpl) DeleteNotInDriveIDsForFolder(ctx context.Context, folderID uuid.UUID, driveFileIDs []string) (int64, error) {
	var totalDeleted int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var photoIDs []uuid.UUID
		var query *gorm.DB

		if len(driveFileIDs) == 0 {
			query = tx.Model(&models.Photo{}).Where("shared_folder_id = ?", folderID).Pluck("id", &photoIDs)
		} else {
			query = tx.Model(&models.Photo{}).Where("shared_folder_id = ? AND drive_file_id NOT IN ?", folderID, driveFileIDs).Pluck("id", &photoIDs)
		}

		if query.Error != nil {
			return query.Error
		}

		if len(photoIDs) == 0 {
			return nil
		}

		if err := tx.Where("photo_id IN ?", photoIDs).Delete(&models.Face{}).Error; err != nil {
			return err
		}

		var result *gorm.DB
		if len(driveFileIDs) == 0 {
			result = tx.Where("shared_folder_id = ?", folderID).Delete(&models.Photo{})
		} else {
			result = tx.Where("shared_folder_id = ? AND drive_file_id NOT IN ?", folderID, driveFileIDs).Delete(&models.Photo{})
		}

		if result.Error != nil {
			return result.Error
		}

		totalDeleted = result.RowsAffected
		return nil
	})

	return totalDeleted, err
}

// ResetFailedToPending resets all failed photos to pending status for retry
func (r *PhotoRepositoryImpl) ResetFailedToPending(ctx context.Context, folderID *uuid.UUID) (int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Photo{}).Where("face_status = ?", models.FaceStatusFailed)

	if folderID != nil {
		query = query.Where("shared_folder_id = ?", *folderID)
	}

	result := query.Updates(map[string]interface{}{
		"face_status": models.FaceStatusPending,
		"updated_at":  time.Now(),
	})

	return result.RowsAffected, result.Error
}

// ResetStuckProcessingToPending resets photos stuck in "processing" status for longer than threshold
func (r *PhotoRepositoryImpl) ResetStuckProcessingToPending(ctx context.Context, stuckThresholdMinutes int) (int64, error) {
	threshold := time.Now().Add(-time.Duration(stuckThresholdMinutes) * time.Minute)

	result := r.db.WithContext(ctx).Model(&models.Photo{}).
		Where("face_status = ?", models.FaceStatusProcessing).
		Where("updated_at < ?", threshold). // Only reset if stuck for longer than threshold
		Updates(map[string]interface{}{
			"face_status": models.FaceStatusPending,
			"updated_at":  time.Now(),
		})

	return result.RowsAffected, result.Error
}
