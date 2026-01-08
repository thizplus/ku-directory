package serviceimpl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"gofiber-template/infrastructure/googledrive"
	"gofiber-template/infrastructure/worker"
)

// Error codes for frontend handling
const (
	ErrCodeGoogleTokenExpired = "GOOGLE_TOKEN_EXPIRED"
)

// GoogleTokenError represents an error when Google OAuth token is invalid/expired
type GoogleTokenError struct {
	Code    string
	Message string
}

func (e *GoogleTokenError) Error() string {
	return e.Message
}

// isGoogleAuthError checks if error is related to Google OAuth authentication
func isGoogleAuthError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "401") ||
		strings.Contains(errStr, "Invalid Credentials") ||
		strings.Contains(errStr, "invalid_grant") ||
		strings.Contains(errStr, "Token has been expired or revoked")
}

// wrapGoogleAuthError wraps Google auth errors with a user-friendly message
func wrapGoogleAuthError(err error) error {
	if isGoogleAuthError(err) {
		return &GoogleTokenError{
			Code:    ErrCodeGoogleTokenExpired,
			Message: "Google Drive token หมดอายุ กรุณาเชื่อมต่อ Google Drive ใหม่",
		}
	}
	return err
}

type SharedFolderServiceImpl struct {
	sharedFolderRepo repositories.SharedFolderRepository
	syncJobRepo      repositories.SyncJobRepository
	photoRepo        repositories.PhotoRepository
	driveClient      *googledrive.DriveClient
	syncWorker       *worker.SyncWorker
}

func NewSharedFolderService(
	sharedFolderRepo repositories.SharedFolderRepository,
	syncJobRepo repositories.SyncJobRepository,
	photoRepo repositories.PhotoRepository,
	driveClient *googledrive.DriveClient,
	syncWorker *worker.SyncWorker,
) services.SharedFolderService {
	return &SharedFolderServiceImpl{
		sharedFolderRepo: sharedFolderRepo,
		syncJobRepo:      syncJobRepo,
		photoRepo:        photoRepo,
		driveClient:      driveClient,
		syncWorker:       syncWorker,
	}
}

// AddFolder adds a new shared folder or joins an existing one
// Returns immediately after creating sync job - photos sync in background via WebSocket updates
func (s *SharedFolderServiceImpl) AddFolder(ctx context.Context, userID uuid.UUID, driveFolderID string, accessToken, refreshToken string) (*models.SharedFolder, error) {
	// Check if folder already exists
	existingFolder, _ := s.sharedFolderRepo.GetByDriveFolderID(ctx, driveFolderID)

	if existingFolder != nil {
		// Folder exists - check if user already has access
		hasAccess, err := s.sharedFolderRepo.HasUserAccess(ctx, userID, existingFolder.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to check user access: %w", err)
		}
		if hasAccess {
			return existingFolder, nil // Already has access
		}

		// Add user access to existing folder
		access := &models.UserFolderAccess{
			ID:             uuid.New(),
			UserID:         userID,
			SharedFolderID: existingFolder.ID,
			CreatedAt:      time.Now(),
		}
		if err := s.sharedFolderRepo.AddUserAccess(ctx, access); err != nil {
			return nil, fmt.Errorf("failed to add user access: %w", err)
		}

		// For existing folder, photos already exist - no need to sync
		fmt.Printf("✅ User %s joined existing folder %s\n", userID, existingFolder.ID)
		return existingFolder, nil
	}

	// Folder doesn't exist - create new one
	srv, err := s.driveClient.GetDriveService(ctx, accessToken, refreshToken, time.Time{})
	if err != nil {
		return nil, wrapGoogleAuthError(fmt.Errorf("failed to get drive service: %w", err))
	}

	folderMeta, err := srv.Files.Get(driveFolderID).Fields("id, name").Do()
	if err != nil {
		return nil, wrapGoogleAuthError(fmt.Errorf("failed to get folder metadata: %w", err))
	}

	// Generate webhook token
	webhookToken := uuid.New().String()

	// Create new shared folder with syncing status (photos will sync in background)
	folder := &models.SharedFolder{
		ID:                uuid.New(),
		DriveFolderID:     driveFolderID,
		DriveFolderName:   folderMeta.Name,
		DriveFolderPath:   folderMeta.Name,
		SyncStatus:        models.SyncStatusSyncing, // Mark as syncing - will update via WebSocket
		DriveAccessToken:  accessToken,
		DriveRefreshToken: refreshToken,
		TokenOwnerID:      userID,
		WebhookToken:      webhookToken,
		PageToken:         "", // Will be set after full sync completes
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}

	if err := s.sharedFolderRepo.Create(ctx, folder); err != nil {
		return nil, fmt.Errorf("failed to create shared folder: %w", err)
	}

	fmt.Printf("📁 Created folder %s (%s)\n", folder.DriveFolderName, folder.ID)

	// Add user access
	access := &models.UserFolderAccess{
		ID:             uuid.New(),
		UserID:         userID,
		SharedFolderID: folder.ID,
		CreatedAt:      time.Now(),
	}
	if err := s.sharedFolderRepo.AddUserAccess(ctx, access); err != nil {
		return nil, fmt.Errorf("failed to add user access: %w", err)
	}

	// Create sync job for background processing
	if err := s.createSyncJob(ctx, userID, folder.ID); err != nil {
		fmt.Printf("⚠️ Warning: Failed to create sync job: %v\n", err)
		// Don't fail - folder is created, user can manually trigger sync later
	} else {
		fmt.Printf("🔄 Sync job created for folder %s - photos will sync in background\n", folder.ID)
	}

	// Register webhook for real-time updates
	go func() {
		webhookCtx := context.Background()
		startPageToken, err := s.driveClient.GetStartPageToken(webhookCtx, srv)
		if err != nil {
			fmt.Printf("⚠️ Warning: Failed to get start page token for webhook: %v\n", err)
			return
		}

		channelID := uuid.New().String()
		channel, err := s.driveClient.WatchChanges(webhookCtx, srv, channelID, webhookToken, startPageToken)
		if err != nil {
			fmt.Printf("⚠️ Warning: Failed to register webhook (this is normal for localhost): %v\n", err)
			return
		}

		// Update folder with webhook info
		expiry := time.UnixMilli(channel.Expiration)
		folder.WebhookChannelID = channel.Id
		folder.WebhookResourceID = channel.ResourceId
		folder.WebhookExpiry = &expiry
		folder.PageToken = startPageToken
		folder.UpdatedAt = time.Now()

		if err := s.sharedFolderRepo.Update(webhookCtx, folder.ID, folder); err != nil {
			fmt.Printf("⚠️ Warning: Failed to update folder with webhook info: %v\n", err)
			return
		}

		fmt.Printf("✅ Webhook registered for folder %s (channel: %s, expires: %v)\n",
			folder.DriveFolderName, channel.Id, expiry)
	}()

	// Return immediately - photos will sync in background via SyncWorker
	// Frontend will receive WebSocket updates: sync:started, sync:progress, sync:completed
	return folder, nil
}

// GetUserFolders returns all folders the user has access to
func (s *SharedFolderServiceImpl) GetUserFolders(ctx context.Context, userID uuid.UUID) ([]models.SharedFolder, error) {
	return s.sharedFolderRepo.GetFoldersByUser(ctx, userID)
}

// GetFolderByID returns a folder if user has access
func (s *SharedFolderServiceImpl) GetFolderByID(ctx context.Context, userID uuid.UUID, folderID uuid.UUID) (*models.SharedFolder, error) {
	// Verify user has access
	hasAccess, err := s.sharedFolderRepo.HasUserAccess(ctx, userID, folderID)
	if err != nil {
		return nil, fmt.Errorf("failed to check access: %w", err)
	}
	if !hasAccess {
		return nil, fmt.Errorf("folder not found")
	}

	return s.sharedFolderRepo.GetByID(ctx, folderID)
}

// RemoveUserAccess removes user's access to a folder
func (s *SharedFolderServiceImpl) RemoveUserAccess(ctx context.Context, userID uuid.UUID, folderID uuid.UUID) error {
	// Verify user has access
	hasAccess, err := s.sharedFolderRepo.HasUserAccess(ctx, userID, folderID)
	if err != nil {
		return fmt.Errorf("failed to check access: %w", err)
	}
	if !hasAccess {
		return fmt.Errorf("folder not found")
	}

	// Remove access
	if err := s.sharedFolderRepo.RemoveUserAccess(ctx, userID, folderID); err != nil {
		return fmt.Errorf("failed to remove access: %w", err)
	}

	// Check if folder has any remaining users
	count, err := s.sharedFolderRepo.CountUsers(ctx, folderID)
	if err != nil {
		return nil // Access removed, don't fail on count error
	}

	// If no users left, we could delete the folder and its photos
	// For now, just leave it - can be cleaned up later
	if count == 0 {
		// TODO: Consider cleaning up orphaned folders
	}

	return nil
}

// TriggerSync triggers a sync for a specific folder
func (s *SharedFolderServiceImpl) TriggerSync(ctx context.Context, userID uuid.UUID, folderID uuid.UUID) error {
	// Verify user has access
	hasAccess, err := s.sharedFolderRepo.HasUserAccess(ctx, userID, folderID)
	if err != nil {
		return fmt.Errorf("failed to check access: %w", err)
	}
	if !hasAccess {
		return fmt.Errorf("folder not found")
	}

	// Create sync job
	if err := s.createSyncJob(ctx, userID, folderID); err != nil {
		return fmt.Errorf("failed to create sync job: %w", err)
	}

	fmt.Printf("✅ Created sync job for folder %s (triggered by user %s)\n", folderID, userID)
	return nil
}

// createSyncJob creates a new sync job for a folder
func (s *SharedFolderServiceImpl) createSyncJob(ctx context.Context, userID uuid.UUID, folderID uuid.UUID) error {
	// Create metadata JSON
	metadata := worker.SyncJobMetadata{
		SharedFolderID: folderID,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	// Create sync job
	now := time.Now()
	job := &models.SyncJob{
		ID:        uuid.New(),
		UserID:    userID,
		JobType:   models.SyncJobTypeDriveSync,
		Status:    models.SyncJobStatusPending,
		Metadata:  string(metadataJSON),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.syncJobRepo.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create sync job: %w", err)
	}

	// Trigger sync worker immediately
	if s.syncWorker != nil {
		s.syncWorker.TriggerSync()
	}

	return nil
}

// GetSyncStatus returns the current sync status of a folder
func (s *SharedFolderServiceImpl) GetSyncStatus(ctx context.Context, folderID uuid.UUID) (*models.SharedFolder, error) {
	return s.sharedFolderRepo.GetByID(ctx, folderID)
}

// HandleWebhook handles webhook notifications for shared folders
func (s *SharedFolderServiceImpl) HandleWebhook(ctx context.Context, channelID, resourceID, resourceState, token string) error {
	fmt.Printf("📩 SharedFolder HandleWebhook: channelID=%s, resourceState=%s, token=%s\n", channelID, resourceState, token)

	// Find shared folder by webhook token
	folder, err := s.sharedFolderRepo.GetByWebhookToken(ctx, token)
	if err != nil {
		fmt.Printf("⚠️ Webhook token not found for shared folder: %v\n", err)
		return fmt.Errorf("webhook token not found: %w", err)
	}

	fmt.Printf("✅ Found shared folder %s (%s) for webhook token\n", folder.ID, folder.DriveFolderName)

	// If it's a sync or change event, trigger sync for this folder
	if resourceState == "sync" || resourceState == "change" {
		fmt.Printf("🔄 Triggering sync for shared folder %s due to %s event\n", folder.ID, resourceState)

		// Get the token owner to trigger sync
		if err := s.triggerSyncForFolder(ctx, folder); err != nil {
			fmt.Printf("❌ Failed to trigger sync: %v\n", err)
			return fmt.Errorf("failed to trigger sync: %w", err)
		}
		fmt.Printf("✅ Sync triggered successfully for shared folder %s\n", folder.ID)
	} else {
		fmt.Printf("ℹ️ Ignoring webhook with state: %s\n", resourceState)
	}

	return nil
}

// RegisterWebhook registers a webhook for an existing folder
func (s *SharedFolderServiceImpl) RegisterWebhook(ctx context.Context, userID uuid.UUID, folderID uuid.UUID) error {
	// Verify user has access
	hasAccess, err := s.sharedFolderRepo.HasUserAccess(ctx, userID, folderID)
	if err != nil {
		return fmt.Errorf("failed to check access: %w", err)
	}
	if !hasAccess {
		return fmt.Errorf("folder not found")
	}

	// Get folder
	folder, err := s.sharedFolderRepo.GetByID(ctx, folderID)
	if err != nil {
		return fmt.Errorf("failed to get folder: %w", err)
	}

	// Get Drive service using folder's tokens (pass expired time to force token refresh)
	srv, err := s.driveClient.GetDriveService(ctx, folder.DriveAccessToken, folder.DriveRefreshToken, time.Time{})
	if err != nil {
		return fmt.Errorf("failed to get drive service: %w", err)
	}

	// Get start page token
	startPageToken, err := s.driveClient.GetStartPageToken(ctx, srv)
	if err != nil {
		return fmt.Errorf("failed to get start page token: %w", err)
	}

	// Generate new webhook token if not exists
	webhookToken := folder.WebhookToken
	if webhookToken == "" {
		webhookToken = uuid.New().String()
	}

	// Register webhook
	channelID := uuid.New().String()
	channel, err := s.driveClient.WatchChanges(ctx, srv, channelID, webhookToken, startPageToken)
	if err != nil {
		return fmt.Errorf("failed to register webhook: %w", err)
	}

	// Update folder with webhook info
	expiry := time.UnixMilli(channel.Expiration)
	folder.WebhookToken = webhookToken
	folder.WebhookChannelID = channel.Id
	folder.WebhookResourceID = channel.ResourceId
	folder.WebhookExpiry = &expiry
	folder.PageToken = startPageToken
	folder.UpdatedAt = time.Now()

	if err := s.sharedFolderRepo.Update(ctx, folder.ID, folder); err != nil {
		return fmt.Errorf("failed to update folder: %w", err)
	}

	fmt.Printf("✅ Webhook registered for folder %s (channel: %s, expires: %v)\n",
		folder.DriveFolderName, channel.Id, expiry)

	return nil
}

// triggerSyncForFolder triggers sync for a specific shared folder
func (s *SharedFolderServiceImpl) triggerSyncForFolder(ctx context.Context, folder *models.SharedFolder) error {
	// Create sync job using the token owner's ID
	metadata := worker.SyncJobMetadata{
		SharedFolderID: folder.ID,
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	now := time.Now()
	job := &models.SyncJob{
		ID:        uuid.New(),
		UserID:    folder.TokenOwnerID, // Use token owner
		JobType:   models.SyncJobTypeDriveSync,
		Status:    models.SyncJobStatusPending,
		Metadata:  string(metadataJSON),
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.syncJobRepo.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create sync job: %w", err)
	}

	// Trigger sync worker immediately
	if s.syncWorker != nil {
		s.syncWorker.TriggerSync()
	}

	fmt.Printf("✅ Created sync job %s for shared folder %s\n", job.ID, folder.ID)
	return nil
}
