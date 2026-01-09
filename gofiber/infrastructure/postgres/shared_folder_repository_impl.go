package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
)

type SharedFolderRepositoryImpl struct {
	db *gorm.DB
}

func NewSharedFolderRepository(db *gorm.DB) repositories.SharedFolderRepository {
	return &SharedFolderRepositoryImpl{db: db}
}

// Create creates a new shared folder
func (r *SharedFolderRepositoryImpl) Create(ctx context.Context, folder *models.SharedFolder) error {
	return r.db.WithContext(ctx).Create(folder).Error
}

// GetByID gets a shared folder by ID
func (r *SharedFolderRepositoryImpl) GetByID(ctx context.Context, id uuid.UUID) (*models.SharedFolder, error) {
	var folder models.SharedFolder
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&folder).Error
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

// GetByDriveFolderID gets a shared folder by Google Drive folder ID
func (r *SharedFolderRepositoryImpl) GetByDriveFolderID(ctx context.Context, driveFolderID string) (*models.SharedFolder, error) {
	var folder models.SharedFolder
	err := r.db.WithContext(ctx).Where("drive_folder_id = ?", driveFolderID).First(&folder).Error
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

// GetByWebhookToken gets a shared folder by webhook token
func (r *SharedFolderRepositoryImpl) GetByWebhookToken(ctx context.Context, token string) (*models.SharedFolder, error) {
	var folder models.SharedFolder
	err := r.db.WithContext(ctx).Where("webhook_token = ?", token).First(&folder).Error
	if err != nil {
		return nil, err
	}
	return &folder, nil
}

// GetAll gets all shared folders
func (r *SharedFolderRepositoryImpl) GetAll(ctx context.Context) ([]models.SharedFolder, error) {
	var folders []models.SharedFolder
	err := r.db.WithContext(ctx).Find(&folders).Error
	return folders, err
}

// GetAllNeedingSync gets all folders that need syncing (have valid tokens)
func (r *SharedFolderRepositoryImpl) GetAllNeedingSync(ctx context.Context) ([]models.SharedFolder, error) {
	var folders []models.SharedFolder
	err := r.db.WithContext(ctx).
		Where("drive_refresh_token != ''").
		Where("sync_status != ?", models.SyncStatusSyncing).
		Find(&folders).Error
	return folders, err
}

// Update updates a shared folder
func (r *SharedFolderRepositoryImpl) Update(ctx context.Context, id uuid.UUID, folder *models.SharedFolder) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Updates(folder).Error
}

// UpdateMetadata updates folder metadata using a map (ensures all fields are updated)
func (r *SharedFolderRepositoryImpl) UpdateMetadata(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.SharedFolder{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateSyncStatus updates the sync status of a folder
func (r *SharedFolderRepositoryImpl) UpdateSyncStatus(ctx context.Context, id uuid.UUID, status models.SyncStatus, lastError string) error {
	updates := map[string]interface{}{
		"sync_status": status,
		"last_error":  lastError,
		"updated_at":  time.Now(),
	}
	if status == models.SyncStatusIdle && lastError == "" {
		updates["last_synced_at"] = time.Now()
	}
	return r.db.WithContext(ctx).Model(&models.SharedFolder{}).Where("id = ?", id).Updates(updates).Error
}

// UpdateTokens updates the OAuth tokens for a folder
func (r *SharedFolderRepositoryImpl) UpdateTokens(ctx context.Context, id uuid.UUID, accessToken, refreshToken string, expiry *time.Time, ownerID uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.SharedFolder{}).Where("id = ?", id).Updates(map[string]interface{}{
		"drive_access_token":  accessToken,
		"drive_refresh_token": refreshToken,
		"drive_token_expiry":  expiry,
		"token_owner_id":      ownerID,
		"updated_at":          time.Now(),
	}).Error
}

// ResetSyncState resets PageToken and LastSyncedAt to force a full sync
func (r *SharedFolderRepositoryImpl) ResetSyncState(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&models.SharedFolder{}).Where("id = ?", id).Updates(map[string]interface{}{
		"page_token":     "",
		"last_synced_at": nil,
		"updated_at":     time.Now(),
	}).Error
}

// Delete deletes a shared folder
func (r *SharedFolderRepositoryImpl) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&models.SharedFolder{}).Error
}

// AddUserAccess adds user access to a folder
func (r *SharedFolderRepositoryImpl) AddUserAccess(ctx context.Context, access *models.UserFolderAccess) error {
	return r.db.WithContext(ctx).Create(access).Error
}

// RemoveUserAccess removes user access from a folder
func (r *SharedFolderRepositoryImpl) RemoveUserAccess(ctx context.Context, userID, folderID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND shared_folder_id = ?", userID, folderID).
		Delete(&models.UserFolderAccess{}).Error
}

// GetUserAccess gets a specific user's access to a folder
func (r *SharedFolderRepositoryImpl) GetUserAccess(ctx context.Context, userID, folderID uuid.UUID) (*models.UserFolderAccess, error) {
	var access models.UserFolderAccess
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND shared_folder_id = ?", userID, folderID).
		First(&access).Error
	if err != nil {
		return nil, err
	}
	return &access, nil
}

// GetUsersByFolder gets all users with access to a folder
func (r *SharedFolderRepositoryImpl) GetUsersByFolder(ctx context.Context, folderID uuid.UUID) ([]models.User, error) {
	var users []models.User
	err := r.db.WithContext(ctx).
		Joins("JOIN user_folder_access ON users.id = user_folder_access.user_id").
		Where("user_folder_access.shared_folder_id = ?", folderID).
		Find(&users).Error
	return users, err
}

// GetFoldersByUser gets all folders a user has access to
func (r *SharedFolderRepositoryImpl) GetFoldersByUser(ctx context.Context, userID uuid.UUID) ([]models.SharedFolder, error) {
	var folders []models.SharedFolder
	err := r.db.WithContext(ctx).
		Joins("JOIN user_folder_access ON shared_folders.id = user_folder_access.shared_folder_id").
		Where("user_folder_access.user_id = ?", userID).
		Find(&folders).Error
	return folders, err
}

// HasUserAccess checks if a user has access to a folder
func (r *SharedFolderRepositoryImpl) HasUserAccess(ctx context.Context, userID, folderID uuid.UUID) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.UserFolderAccess{}).
		Where("user_id = ? AND shared_folder_id = ?", userID, folderID).
		Count(&count).Error
	return count > 0, err
}

// CountUsers counts the number of users with access to a folder
func (r *SharedFolderRepositoryImpl) CountUsers(ctx context.Context, folderID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.UserFolderAccess{}).
		Where("shared_folder_id = ?", folderID).
		Count(&count).Error
	return count, err
}

// GetFoldersWithExpiringWebhooks gets folders with webhooks expiring before the given threshold
func (r *SharedFolderRepositoryImpl) GetFoldersWithExpiringWebhooks(ctx context.Context, expiryThreshold time.Time) ([]models.SharedFolder, error) {
	var folders []models.SharedFolder
	err := r.db.WithContext(ctx).
		Where("webhook_expiry IS NOT NULL").
		Where("webhook_expiry < ?", expiryThreshold).
		Where("webhook_channel_id != ''").
		Where("drive_refresh_token != ''"). // Must have valid tokens to renew
		Find(&folders).Error
	return folders, err
}
