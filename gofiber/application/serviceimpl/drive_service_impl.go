package serviceimpl

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"gofiber-template/infrastructure/googledrive"
	"gofiber-template/pkg/logger"
)

type DriveServiceImpl struct {
	driveClient *googledrive.DriveClient
	userRepo    repositories.UserRepository
	photoRepo   repositories.PhotoRepository
	syncJobRepo repositories.SyncJobRepository
}

func NewDriveService(
	driveClient *googledrive.DriveClient,
	userRepo repositories.UserRepository,
	photoRepo repositories.PhotoRepository,
	syncJobRepo repositories.SyncJobRepository,
) services.DriveService {
	return &DriveServiceImpl{
		driveClient: driveClient,
		userRepo:    userRepo,
		photoRepo:   photoRepo,
		syncJobRepo: syncJobRepo,
	}
}

// GetAuthURL generates the OAuth authorization URL
func (s *DriveServiceImpl) GetAuthURL(state string) string {
	return s.driveClient.GetAuthURL(state)
}

// HandleCallback handles OAuth callback and stores tokens
func (s *DriveServiceImpl) HandleCallback(ctx context.Context, userID uuid.UUID, code string) error {
	logger.Drive("oauth_callback_start", "HandleCallback starting", map[string]interface{}{
		"user_id": userID.String(),
	})

	// Exchange code for tokens
	tokenInfo, err := s.driveClient.ExchangeCode(ctx, code)
	if err != nil {
		logger.DriveError("oauth_exchange_failed", "ExchangeCode failed", err, map[string]interface{}{
			"user_id": userID.String(),
		})
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	logger.Drive("oauth_tokens_received", "Got tokens", map[string]interface{}{
		"user_id":              userID.String(),
		"access_token_length":  len(tokenInfo.AccessToken),
		"refresh_token_length": len(tokenInfo.RefreshToken),
	})

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		logger.DriveError("oauth_get_user_failed", "GetByID failed", err, map[string]interface{}{
			"user_id": userID.String(),
		})
		return fmt.Errorf("failed to get user: %w", err)
	}

	logger.Drive("oauth_user_found", "Got user", map[string]interface{}{
		"user_id": userID.String(),
		"email":   user.Email,
	})

	// Update user with Drive tokens
	user.DriveAccessToken = tokenInfo.AccessToken
	user.DriveRefreshToken = tokenInfo.RefreshToken
	user.DriveTokenExpiry = &tokenInfo.Expiry
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, userID, user); err != nil {
		logger.DriveError("oauth_update_failed", "Update failed", err, map[string]interface{}{
			"user_id": userID.String(),
		})
		return fmt.Errorf("failed to update user: %w", err)
	}

	logger.Drive("oauth_callback_complete", "User updated with Drive tokens", map[string]interface{}{
		"user_id": userID.String(),
		"email":   user.Email,
	})

	return nil
}

// IsConnected checks if user has connected Google Drive
func (s *DriveServiceImpl) IsConnected(ctx context.Context, userID uuid.UUID) bool {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false
	}
	return user.DriveRefreshToken != ""
}

// Disconnect removes Google Drive connection
func (s *DriveServiceImpl) Disconnect(ctx context.Context, userID uuid.UUID) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	user.DriveAccessToken = ""
	user.DriveRefreshToken = ""
	user.DriveTokenExpiry = nil
	user.DriveRootFolderID = ""
	user.UpdatedAt = time.Now()

	return s.userRepo.Update(ctx, userID, user)
}

// getDriveService gets Drive API service for user (handles token refresh)
func (s *DriveServiceImpl) getDriveService(ctx context.Context, userID uuid.UUID) (*googledrive.DriveClient, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.DriveRefreshToken == "" {
		return nil, fmt.Errorf("user has not connected Google Drive")
	}

	// Check if token needs refresh
	if user.DriveTokenExpiry != nil && user.DriveTokenExpiry.Before(time.Now()) {
		tokenInfo, err := s.driveClient.RefreshToken(ctx, user.DriveRefreshToken)
		if err != nil {
			return nil, fmt.Errorf("failed to refresh token: %w", err)
		}

		user.DriveAccessToken = tokenInfo.AccessToken
		if tokenInfo.RefreshToken != "" {
			user.DriveRefreshToken = tokenInfo.RefreshToken
		}
		user.DriveTokenExpiry = &tokenInfo.Expiry
		user.UpdatedAt = time.Now()

		if err := s.userRepo.Update(ctx, userID, user); err != nil {
			return nil, fmt.Errorf("failed to update token: %w", err)
		}
	}

	return s.driveClient, nil
}

// ListFolders lists folders in Google Drive
func (s *DriveServiceImpl) ListFolders(ctx context.Context, userID uuid.UUID, parentID string) ([]services.DriveFolder, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	expiry := time.Now()
	if user.DriveTokenExpiry != nil {
		expiry = *user.DriveTokenExpiry
	}

	srv, err := s.driveClient.GetDriveService(ctx, user.DriveAccessToken, user.DriveRefreshToken, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to get drive service: %w", err)
	}

	folders, err := s.driveClient.ListFolders(ctx, srv, parentID)
	if err != nil {
		return nil, err
	}

	var result []services.DriveFolder
	for _, f := range folders {
		result = append(result, services.DriveFolder{
			ID:       f.ID,
			Name:     f.Name,
			Path:     f.Path,
			ParentID: f.ParentID,
		})
	}

	return result, nil
}

// SetRootFolder sets the root folder for sync and registers a webhook
func (s *DriveServiceImpl) SetRootFolder(ctx context.Context, userID uuid.UUID, folderID string) error {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Get Drive service
	expiry := time.Now()
	if user.DriveTokenExpiry != nil {
		expiry = *user.DriveTokenExpiry
	}

	srv, err := s.driveClient.GetDriveService(ctx, user.DriveAccessToken, user.DriveRefreshToken, expiry)
	if err != nil {
		return fmt.Errorf("failed to get drive service: %w", err)
	}

	// Generate webhook token if not exists
	if user.DriveWebhookToken == "" {
		user.DriveWebhookToken = uuid.New().String()
	}

	// Get start page token for change tracking
	startPageToken, err := s.driveClient.GetStartPageToken(ctx, srv)
	if err != nil {
		logger.DriveError("page_token_failed", "Failed to get start page token", err, map[string]interface{}{
			"user_id": userID.String(),
		})
	} else {
		user.DrivePageToken = startPageToken
	}

	// Register webhook using Changes API (watches ALL changes in Drive)
	// This is the correct way to detect new files being added to a folder
	channelID := uuid.New().String()
	_, watchErr := s.driveClient.WatchChanges(ctx, srv, channelID, user.DriveWebhookToken, startPageToken)
	if watchErr != nil {
		// Log but don't fail - webhook is optional for local development
		logger.WebhookError("webhook_register_failed", "Failed to register Drive webhook (this is normal for localhost)", watchErr, map[string]interface{}{
			"user_id": userID.String(),
		})
	} else {
		logger.Webhook("webhook_registered", "Drive Changes webhook registered", map[string]interface{}{
			"user_id":    userID.String(),
			"channel_id": channelID,
		})
	}

	// Get folder name for path-based queries (shared folder model)
	folderInfo, err := s.driveClient.GetFile(ctx, srv, folderID)
	if err != nil {
		logger.DriveError("get_folder_name_failed", "Failed to get folder name", err, map[string]interface{}{
			"user_id":   userID.String(),
			"folder_id": folderID,
		})
	} else {
		user.DriveRootFolderName = folderInfo.Name
	}

	user.DriveRootFolderID = folderID
	user.UpdatedAt = time.Now()

	return s.userRepo.Update(ctx, userID, user)
}

// GetRootFolder gets the root folder ID
func (s *DriveServiceImpl) GetRootFolder(ctx context.Context, userID uuid.UUID) (string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get user: %w", err)
	}
	return user.DriveRootFolderID, nil
}

// GetRootFolderInfo gets the root folder info (ID and name) from Google Drive
func (s *DriveServiceImpl) GetRootFolderInfo(ctx context.Context, userID uuid.UUID) (*services.DriveFolder, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.DriveRootFolderID == "" {
		return nil, nil // No root folder set
	}

	// Get Drive service
	expiry := time.Now()
	if user.DriveTokenExpiry != nil {
		expiry = *user.DriveTokenExpiry
	}

	srv, err := s.driveClient.GetDriveService(ctx, user.DriveAccessToken, user.DriveRefreshToken, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to get drive service: %w", err)
	}

	// Get folder info from Google Drive
	file, err := s.driveClient.GetFile(ctx, srv, user.DriveRootFolderID)
	if err != nil {
		return nil, fmt.Errorf("failed to get folder info: %w", err)
	}

	return &services.DriveFolder{
		ID:   file.ID,
		Name: file.Name,
	}, nil
}

// StartSync starts a sync job for the user
func (s *DriveServiceImpl) StartSync(ctx context.Context, userID uuid.UUID) (*models.SyncJob, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.DriveRootFolderID == "" {
		return nil, fmt.Errorf("root folder not set")
	}

	// Check if there's already a running sync job
	existingJob, err := s.syncJobRepo.GetLatestByUserAndType(ctx, userID, models.SyncJobTypeDriveSync)
	if err == nil && existingJob != nil {
		if existingJob.Status == models.SyncJobStatusPending || existingJob.Status == models.SyncJobStatusRunning {
			// Check if job is stuck (more than 30 minutes old)
			if time.Since(existingJob.UpdatedAt) > 30*time.Minute {
				// Mark as failed and create new job
				existingJob.Status = models.SyncJobStatusFailed
				existingJob.LastError = "Job timed out"
				s.syncJobRepo.Update(ctx, existingJob.ID, existingJob)
			} else {
				return existingJob, nil // Return existing job
			}
		}
	}

	// Create new sync job
	now := time.Now()
	job := &models.SyncJob{
		ID:        uuid.New(),
		UserID:    userID,
		JobType:   models.SyncJobTypeDriveSync,
		Status:    models.SyncJobStatusPending,
		Metadata:  "{}",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.syncJobRepo.Create(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create sync job: %w", err)
	}

	// TODO: Trigger background sync worker
	// This should be done via a message queue or background worker

	return job, nil
}

// GetSyncStatus gets the latest sync job status
func (s *DriveServiceImpl) GetSyncStatus(ctx context.Context, userID uuid.UUID) (*models.SyncJob, error) {
	job, err := s.syncJobRepo.GetLatestByUserAndType(ctx, userID, models.SyncJobTypeDriveSync)
	if err != nil {
		return nil, err
	}

	// Check if job is stuck (running/pending for more than 30 minutes)
	if job != nil && (job.Status == models.SyncJobStatusPending || job.Status == models.SyncJobStatusRunning) {
		if time.Since(job.UpdatedAt) > 30*time.Minute {
			// Auto-reset stuck job to failed
			job.Status = models.SyncJobStatusFailed
			job.LastError = "Job timed out after 30 minutes"
			now := time.Now()
			job.CompletedAt = &now
			job.UpdatedAt = now
			s.syncJobRepo.Update(ctx, job.ID, job)
			logger.Sync("job_auto_reset", "Auto-reset stuck sync job", map[string]interface{}{
				"job_id":  job.ID.String(),
				"user_id": userID.String(),
				"reason":  "timeout_30_minutes",
			})
		}
	}

	return job, nil
}

// GetPhotos gets paginated photos for user
func (s *DriveServiceImpl) GetPhotos(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Photo, int64, error) {
	offset := (page - 1) * limit

	// Shared model: Get user's root folder name and query by path prefix
	// This allows users to see photos synced by others in the same shared folder
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	if user.DriveRootFolderName != "" {
		// Query all photos under user's root folder tree (shared)
		return s.photoRepo.GetByFolderPathPrefix(ctx, user.DriveRootFolderName, offset, limit)
	}

	// Fallback: no root folder set, use original user-based query
	return s.photoRepo.GetByUser(ctx, userID, offset, limit)
}

// GetPhotosByFolder gets photos by folder path
func (s *DriveServiceImpl) GetPhotosByFolder(ctx context.Context, userID uuid.UUID, folderPath string, page, limit int) ([]models.Photo, int64, error) {
	offset := (page - 1) * limit
	return s.photoRepo.GetByUserAndFolder(ctx, userID, folderPath, offset, limit)
}

// GetPhotosByFolderId gets photos by folder ID
func (s *DriveServiceImpl) GetPhotosByFolderId(ctx context.Context, userID uuid.UUID, folderId string, page, limit int) ([]models.Photo, int64, error) {
	offset := (page - 1) * limit
	return s.photoRepo.GetByUserAndFolderId(ctx, userID, folderId, offset, limit)
}

// SearchPhotos searches photos by folder path (activity name)
func (s *DriveServiceImpl) SearchPhotos(ctx context.Context, userID uuid.UUID, searchQuery string, page, limit int) ([]models.Photo, int64, error) {
	offset := (page - 1) * limit
	return s.photoRepo.SearchByFolderPath(ctx, userID, searchQuery, offset, limit)
}

// GetPhotoThumbnail gets photo thumbnail from Google Drive
func (s *DriveServiceImpl) GetPhotoThumbnail(ctx context.Context, userID uuid.UUID, driveFileID string, size int) ([]byte, string, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get user: %w", err)
	}

	if user.DriveRefreshToken == "" {
		return nil, "", fmt.Errorf("user has not connected Google Drive")
	}

	expiry := time.Now()
	if user.DriveTokenExpiry != nil {
		expiry = *user.DriveTokenExpiry
	}

	return s.driveClient.DownloadThumbnail(ctx, user.DriveAccessToken, user.DriveRefreshToken, expiry, driveFileID, size)
}

// DownloadPhotosAsZip downloads multiple photos and returns them as a zip file
func (s *DriveServiceImpl) DownloadPhotosAsZip(ctx context.Context, userID uuid.UUID, driveFileIDs []string, onProgress services.DownloadProgressCallback) ([]byte, error) {
	if len(driveFileIDs) == 0 {
		return nil, fmt.Errorf("no files to download")
	}

	if len(driveFileIDs) > 50 {
		return nil, fmt.Errorf("too many files, maximum is 50")
	}

	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.DriveRefreshToken == "" {
		return nil, fmt.Errorf("user has not connected Google Drive")
	}

	expiry := time.Now()
	if user.DriveTokenExpiry != nil {
		expiry = *user.DriveTokenExpiry
	}

	// Get Drive service
	srv, err := s.driveClient.GetDriveService(ctx, user.DriveAccessToken, user.DriveRefreshToken, expiry)
	if err != nil {
		return nil, fmt.Errorf("failed to get drive service: %w", err)
	}

	// Create zip buffer
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Track filenames to handle duplicates
	filenameCount := make(map[string]int)
	totalFiles := len(driveFileIDs)

	for i, fileID := range driveFileIDs {
		// Get file metadata
		file, err := srv.Files.Get(fileID).Fields("id, name, mimeType").Do()
		if err != nil {
			logger.DriveError("zip_get_file_failed", "Failed to get file metadata", err, map[string]interface{}{
				"file_id": fileID,
				"user_id": userID.String(),
			})
			continue
		}

		// Send progress callback
		if onProgress != nil {
			onProgress(services.DownloadProgress{
				Current:  i + 1,
				Total:    totalFiles,
				FileName: file.Name,
			})
		}

		// Download file content
		resp, err := srv.Files.Get(fileID).Download()
		if err != nil {
			logger.DriveError("zip_download_failed", "Failed to download file", err, map[string]interface{}{
				"file_id":   fileID,
				"file_name": file.Name,
				"user_id":   userID.String(),
			})
			continue
		}

		// Handle duplicate filenames
		filename := file.Name
		if count, exists := filenameCount[filename]; exists {
			ext := filepath.Ext(filename)
			base := filename[:len(filename)-len(ext)]
			filename = fmt.Sprintf("%s_%d%s", base, count+1, ext)
		}
		filenameCount[file.Name]++

		// Create file in zip
		zipFile, err := zipWriter.Create(filename)
		if err != nil {
			resp.Body.Close()
			logger.DriveError("zip_create_entry_failed", "Failed to create zip entry", err, map[string]interface{}{
				"file_name": filename,
				"user_id":   userID.String(),
			})
			continue
		}

		// Copy file content to zip
		_, err = io.Copy(zipFile, resp.Body)
		resp.Body.Close()
		if err != nil {
			logger.DriveError("zip_write_failed", "Failed to write file to zip", err, map[string]interface{}{
				"file_name": filename,
				"user_id":   userID.String(),
			})
			continue
		}
	}

	// Close zip writer
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip: %w", err)
	}

	return buf.Bytes(), nil
}

// HandleWebhook handles Google Drive webhook notifications
func (s *DriveServiceImpl) HandleWebhook(ctx context.Context, channelID, resourceID, resourceState, token string) error {
	logger.Webhook("webhook_received", "HandleWebhook received", map[string]interface{}{
		"channel_id":     channelID,
		"resource_state": resourceState,
		"token_length":   len(token),
	})

	// Try to find user by webhook token first
	user, err := s.userRepo.GetByDriveWebhookToken(ctx, token)
	if err != nil {
		logger.Webhook("webhook_token_not_found", "Webhook token not found for user, might be shared folder token", map[string]interface{}{
			"error": err.Error(),
		})
		// Token not found for user - this is expected for shared folder webhooks
		// The shared folder handler will handle it separately
		return fmt.Errorf("webhook token not found: %w", err)
	}

	logger.Webhook("webhook_user_found", "Found user for webhook token", map[string]interface{}{
		"user_id": user.ID.String(),
	})

	// Only trigger sync for "change" events, NOT "sync" events
	// "sync" is just Google's test notification when webhook is registered
	// "change" is the actual notification when files/folders change
	if resourceState == "change" {
		logger.Webhook("webhook_trigger_sync", "Triggering sync due to change event", map[string]interface{}{
			"user_id":        user.ID.String(),
			"resource_state": resourceState,
		})
		_, err := s.StartSync(ctx, user.ID)
		if err != nil {
			logger.WebhookError("webhook_sync_failed", "Failed to start sync", err, map[string]interface{}{
				"user_id": user.ID.String(),
			})
			return fmt.Errorf("failed to start sync: %w", err)
		}
		logger.Webhook("webhook_sync_triggered", "Sync triggered successfully", map[string]interface{}{
			"user_id": user.ID.String(),
		})
	} else if resourceState == "sync" {
		// "sync" is just Google testing if our endpoint works - no action needed
		logger.Webhook("webhook_sync_ack", "Received sync acknowledgment (no action)", map[string]interface{}{
			"user_id":        user.ID.String(),
			"resource_state": resourceState,
		})
	} else {
		logger.Webhook("webhook_ignored", "Ignoring webhook with unknown state", map[string]interface{}{
			"user_id":        user.ID.String(),
			"resource_state": resourceState,
		})
	}

	return nil
}
