package serviceimpl

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"gofiber-template/infrastructure/googledrive"
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
	fmt.Printf("🔄 HandleCallback starting for user: %s\n", userID)

	// Exchange code for tokens
	tokenInfo, err := s.driveClient.ExchangeCode(ctx, code)
	if err != nil {
		fmt.Printf("❌ ExchangeCode failed: %v\n", err)
		return fmt.Errorf("failed to exchange code: %w", err)
	}

	fmt.Printf("✅ Got tokens - AccessToken: %d chars, RefreshToken: %d chars\n",
		len(tokenInfo.AccessToken), len(tokenInfo.RefreshToken))

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		fmt.Printf("❌ GetByID failed: %v\n", err)
		return fmt.Errorf("failed to get user: %w", err)
	}

	fmt.Printf("✅ Got user: %s\n", user.Email)

	// Update user with Drive tokens
	user.DriveAccessToken = tokenInfo.AccessToken
	user.DriveRefreshToken = tokenInfo.RefreshToken
	user.DriveTokenExpiry = &tokenInfo.Expiry
	user.UpdatedAt = time.Now()

	if err := s.userRepo.Update(ctx, userID, user); err != nil {
		fmt.Printf("❌ Update failed: %v\n", err)
		return fmt.Errorf("failed to update user: %w", err)
	}

	fmt.Println("✅ User updated with Drive tokens")

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
		log.Printf("Warning: Failed to get start page token: %v", err)
	} else {
		user.DrivePageToken = startPageToken
	}

	// Register webhook using Changes API (watches ALL changes in Drive)
	// This is the correct way to detect new files being added to a folder
	channelID := uuid.New().String()
	_, watchErr := s.driveClient.WatchChanges(ctx, srv, channelID, user.DriveWebhookToken, startPageToken)
	if watchErr != nil {
		// Log but don't fail - webhook is optional for local development
		log.Printf("Warning: Failed to register Drive webhook (this is normal for localhost): %v", watchErr)
	} else {
		log.Printf("Drive Changes webhook registered for user %s (watching all Drive changes)", userID)
	}

	// Get folder name for path-based queries (shared folder model)
	folderInfo, err := s.driveClient.GetFile(ctx, srv, folderID)
	if err != nil {
		log.Printf("Warning: Failed to get folder name: %v", err)
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
			log.Printf("Auto-reset stuck sync job %s for user %s", job.ID, userID)
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
			log.Printf("Warning: failed to get file %s: %v", fileID, err)
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
			log.Printf("Warning: failed to download file %s: %v", fileID, err)
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
			log.Printf("Warning: failed to create zip entry for %s: %v", filename, err)
			continue
		}

		// Copy file content to zip
		_, err = io.Copy(zipFile, resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("Warning: failed to write file %s to zip: %v", filename, err)
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
	log.Printf("📩 HandleWebhook: channelID=%s, resourceState=%s, token=%s", channelID, resourceState, token)

	// Try to find user by webhook token first
	user, err := s.userRepo.GetByDriveWebhookToken(ctx, token)
	if err != nil {
		log.Printf("⚠️ Webhook token not found for user, might be shared folder token: %v", err)
		// Token not found for user - this is expected for shared folder webhooks
		// The shared folder handler will handle it separately
		return fmt.Errorf("webhook token not found: %w", err)
	}

	log.Printf("✅ Found user %s for webhook token", user.ID)

	// If it's a sync or change event, trigger incremental sync
	if resourceState == "sync" || resourceState == "change" {
		log.Printf("🔄 Triggering sync for user %s due to %s event", user.ID, resourceState)
		_, err := s.StartSync(ctx, user.ID)
		if err != nil {
			log.Printf("❌ Failed to start sync: %v", err)
			return fmt.Errorf("failed to start sync: %w", err)
		}
		log.Printf("✅ Sync triggered successfully for user %s", user.ID)
	} else {
		log.Printf("ℹ️ Ignoring webhook with state: %s", resourceState)
	}

	return nil
}
