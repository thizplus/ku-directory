package serviceimpl

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/api/googleapi"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"gofiber-template/infrastructure/googledrive"
	"gofiber-template/infrastructure/worker"
	"gofiber-template/pkg/logger"
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

// GoogleAPIErrorDetails contains detailed information from Google API errors
type GoogleAPIErrorDetails struct {
	HTTPStatusCode int                    `json:"http_status_code"`
	ErrorMessage   string                 `json:"error_message"`
	ErrorBody      string                 `json:"error_body,omitempty"`
	ErrorDetails   []googleapi.ErrorItem  `json:"error_details,omitempty"`
	RawError       string                 `json:"raw_error"`
}

// parseGoogleAPIError extracts detailed error information from Google API errors
func parseGoogleAPIError(err error) *GoogleAPIErrorDetails {
	if err == nil {
		return nil
	}

	details := &GoogleAPIErrorDetails{
		RawError: err.Error(),
	}

	// Try to extract googleapi.Error details
	if apiErr, ok := err.(*googleapi.Error); ok {
		details.HTTPStatusCode = apiErr.Code
		details.ErrorMessage = apiErr.Message
		details.ErrorBody = apiErr.Body
		details.ErrorDetails = apiErr.Errors
	} else {
		// Try to find wrapped googleapi.Error
		errStr := err.Error()
		if strings.Contains(errStr, "googleapi:") {
			details.ErrorMessage = errStr
		}
	}

	return details
}

// logGoogleAPIError logs detailed Google API error information
func logGoogleAPIError(action, message string, err error, extraData map[string]interface{}) {
	details := parseGoogleAPIError(err)

	data := map[string]interface{}{
		"raw_error": details.RawError,
	}

	if details.HTTPStatusCode != 0 {
		data["http_status_code"] = details.HTTPStatusCode
	}
	if details.ErrorMessage != "" {
		data["google_error_message"] = details.ErrorMessage
	}
	if details.ErrorBody != "" {
		data["google_error_body"] = details.ErrorBody
	}
	if len(details.ErrorDetails) > 0 {
		// Convert error details to readable format
		errorItems := make([]map[string]string, len(details.ErrorDetails))
		for i, item := range details.ErrorDetails {
			errorItems[i] = map[string]string{
				"reason":  item.Reason,
				"message": item.Message,
			}
		}
		data["google_error_details"] = errorItems
	}

	// Merge extra data
	for k, v := range extraData {
		data[k] = v
	}

	logger.DriveError(action, message, err, data)
}

// logGoogleWebhookError logs detailed Google API error information for webhook operations
func logGoogleWebhookError(action, message string, err error, extraData map[string]interface{}) {
	details := parseGoogleAPIError(err)

	data := map[string]interface{}{
		"raw_error": details.RawError,
	}

	if details.HTTPStatusCode != 0 {
		data["http_status_code"] = details.HTTPStatusCode
	}
	if details.ErrorMessage != "" {
		data["google_error_message"] = details.ErrorMessage
	}
	if details.ErrorBody != "" {
		data["google_error_body"] = details.ErrorBody
	}
	if len(details.ErrorDetails) > 0 {
		errorItems := make([]map[string]string, len(details.ErrorDetails))
		for i, item := range details.ErrorDetails {
			errorItems[i] = map[string]string{
				"reason":  item.Reason,
				"message": item.Message,
			}
		}
		data["google_error_details"] = errorItems
	}

	// Merge extra data
	for k, v := range extraData {
		data[k] = v
	}

	logger.WebhookError(action, message, err, data)
}

type SharedFolderServiceImpl struct {
	sharedFolderRepo repositories.SharedFolderRepository
	syncJobRepo      repositories.SyncJobRepository
	photoRepo        repositories.PhotoRepository
	userRepo         repositories.UserRepository
	driveClient      *googledrive.DriveClient
	syncWorker       *worker.SyncWorker
}

func NewSharedFolderService(
	sharedFolderRepo repositories.SharedFolderRepository,
	syncJobRepo repositories.SyncJobRepository,
	photoRepo repositories.PhotoRepository,
	userRepo repositories.UserRepository,
	driveClient *googledrive.DriveClient,
	syncWorker *worker.SyncWorker,
) services.SharedFolderService {
	return &SharedFolderServiceImpl{
		sharedFolderRepo: sharedFolderRepo,
		syncJobRepo:      syncJobRepo,
		photoRepo:        photoRepo,
		userRepo:         userRepo,
		driveClient:      driveClient,
		syncWorker:       syncWorker,
	}
}

// AddFolder adds a new shared folder or joins an existing one
// Returns immediately after creating sync job - photos sync in background via WebSocket updates
func (s *SharedFolderServiceImpl) AddFolder(ctx context.Context, userID uuid.UUID, driveFolderID, resourceKey string, accessToken, refreshToken string) (*models.SharedFolder, error) {
	logger.Drive("add_folder_start", "Starting add folder process", map[string]interface{}{
		"user_id":          userID.String(),
		"drive_folder_id":  driveFolderID,
		"has_resource_key": resourceKey != "",
		"has_token":        accessToken != "",
	})

	// Step 0: Refresh token if needed and save to database
	// This ensures we always use a valid token and persist refreshed tokens
	tokenInfo, wasRefreshed, err := s.driveClient.RefreshTokenIfNeeded(ctx, accessToken, refreshToken, time.Time{})
	if err != nil {
		logGoogleAPIError("token_refresh_failed", "Failed to refresh Google token", err, map[string]interface{}{
			"user_id": userID.String(),
		})
		return nil, wrapGoogleAuthError(fmt.Errorf("failed to refresh token: %w", err))
	}

	// Use the (possibly refreshed) token
	accessToken = tokenInfo.AccessToken
	if tokenInfo.RefreshToken != "" {
		refreshToken = tokenInfo.RefreshToken
	}

	// If token was refreshed, save the new tokens to database
	if wasRefreshed {
		logger.Drive("token_refreshed", "Google token was refreshed, saving to database", map[string]interface{}{
			"user_id":     userID.String(),
			"new_expiry":  tokenInfo.Expiry.Format(time.RFC3339),
			"has_new_refresh_token": tokenInfo.RefreshToken != "",
		})

		// Update user's tokens in database
		if err := s.userRepo.UpdateDriveTokens(ctx, userID, accessToken, refreshToken); err != nil {
			logger.DriveError("save_token_failed", "Failed to save refreshed token to database", err, map[string]interface{}{
				"user_id": userID.String(),
			})
			// Don't fail the operation - we can still proceed with the refreshed token
			// Just log the error
		} else {
			logger.Drive("token_saved", "Refreshed token saved to database", map[string]interface{}{
				"user_id": userID.String(),
			})
		}
	}

	// Check if folder already exists
	existingFolder, _ := s.sharedFolderRepo.GetByDriveFolderID(ctx, driveFolderID)

	if existingFolder != nil {
		logger.Drive("folder_exists", "Folder already exists in database", map[string]interface{}{
			"folder_id":   existingFolder.ID.String(),
			"folder_name": existingFolder.DriveFolderName,
		})

		// Folder exists - check if user already has access
		hasAccess, err := s.sharedFolderRepo.HasUserAccess(ctx, userID, existingFolder.ID)
		if err != nil {
			logger.DriveError("check_access_failed", "Failed to check user access", err, map[string]interface{}{
				"user_id":   userID.String(),
				"folder_id": existingFolder.ID.String(),
			})
			return nil, fmt.Errorf("failed to check user access: %w", err)
		}
		if hasAccess {
			logger.Drive("user_already_has_access", "User already has access to folder", map[string]interface{}{
				"user_id":   userID.String(),
				"folder_id": existingFolder.ID.String(),
			})
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
			logger.DriveError("add_access_failed", "Failed to add user access", err, map[string]interface{}{
				"user_id":   userID.String(),
				"folder_id": existingFolder.ID.String(),
			})
			return nil, fmt.Errorf("failed to add user access: %w", err)
		}

		logger.Drive("user_joined_folder", "User joined existing folder", map[string]interface{}{
			"user_id":     userID.String(),
			"folder_id":   existingFolder.ID.String(),
			"folder_name": existingFolder.DriveFolderName,
		})
		return existingFolder, nil
	}

	// Folder doesn't exist - create new one
	logger.Drive("creating_new_folder", "Folder not found, creating new one", map[string]interface{}{
		"drive_folder_id": driveFolderID,
	})

	logger.Drive("get_drive_service", "Getting Google Drive service", map[string]interface{}{
		"has_access_token":  accessToken != "",
		"has_refresh_token": refreshToken != "",
		"has_resource_key":  resourceKey != "",
	})

	srv, err := s.driveClient.GetDriveServiceWithResourceKey(ctx, accessToken, refreshToken, time.Time{}, driveFolderID, resourceKey)
	if err != nil {
		logGoogleAPIError("get_drive_service_failed", "Failed to get drive service", err, map[string]interface{}{
			"user_id":          userID.String(),
			"drive_folder_id":  driveFolderID,
			"has_access_token": accessToken != "",
			"has_resource_key": resourceKey != "",
		})
		return nil, wrapGoogleAuthError(fmt.Errorf("failed to get drive service: %w", err))
	}

	logger.Drive("fetching_folder_metadata", "Fetching folder metadata from Google Drive", map[string]interface{}{
		"drive_folder_id": driveFolderID,
	})

	folderMeta, err := srv.Files.Get(driveFolderID).Fields("id, name, mimeType, owners, shared, capabilities").Do()
	if err != nil {
		logGoogleAPIError("get_metadata_failed", "Failed to get folder metadata from Google Drive", err, map[string]interface{}{
			"user_id":          userID.String(),
			"drive_folder_id":  driveFolderID,
			"has_resource_key": resourceKey != "",
		})
		return nil, wrapGoogleAuthError(fmt.Errorf("failed to get folder metadata: %w", err))
	}

	// Log successful response with more details
	folderDetails := map[string]interface{}{
		"drive_folder_id":   folderMeta.Id,
		"drive_folder_name": folderMeta.Name,
		"mime_type":         folderMeta.MimeType,
		"shared":            folderMeta.Shared,
	}
	if len(folderMeta.Owners) > 0 {
		folderDetails["owner_email"] = folderMeta.Owners[0].EmailAddress
	}
	if folderMeta.Capabilities != nil {
		folderDetails["can_read_revisions"] = folderMeta.Capabilities.CanReadRevisions
		folderDetails["can_list_children"] = folderMeta.Capabilities.CanListChildren
	}
	logger.Drive("folder_metadata_received", "Got folder metadata from Google Drive", folderDetails)

	// Generate webhook token
	webhookToken := uuid.New().String()

	// Create new shared folder with syncing status (photos will sync in background)
	folder := &models.SharedFolder{
		ID:                uuid.New(),
		DriveFolderID:     driveFolderID,
		DriveFolderName:   folderMeta.Name,
		DriveFolderPath:   folderMeta.Name,
		DriveResourceKey:  resourceKey, // For older shared folders (pre-2021)
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
		logger.DriveError("create_folder_failed", "Failed to create folder in database", err, map[string]interface{}{
			"folder_id":   folder.ID.String(),
			"folder_name": folder.DriveFolderName,
		})
		return nil, fmt.Errorf("failed to create shared folder: %w", err)
	}

	logger.Drive("folder_created", "Folder created in database", map[string]interface{}{
		"folder_id":       folder.ID.String(),
		"folder_name":     folder.DriveFolderName,
		"drive_folder_id": folder.DriveFolderID,
		"sync_status":     string(folder.SyncStatus),
	})

	// Add user access
	access := &models.UserFolderAccess{
		ID:             uuid.New(),
		UserID:         userID,
		SharedFolderID: folder.ID,
		CreatedAt:      time.Now(),
	}
	if err := s.sharedFolderRepo.AddUserAccess(ctx, access); err != nil {
		logger.DriveError("add_owner_access_failed", "Failed to add owner access", err, map[string]interface{}{
			"user_id":   userID.String(),
			"folder_id": folder.ID.String(),
		})
		return nil, fmt.Errorf("failed to add user access: %w", err)
	}

	logger.Drive("owner_access_added", "Owner access added to folder", map[string]interface{}{
		"user_id":   userID.String(),
		"folder_id": folder.ID.String(),
	})

	// Create sync job for background processing
	if err := s.createSyncJob(ctx, userID, folder.ID); err != nil {
		logger.DriveError("create_sync_job_failed", "Failed to create sync job", err, map[string]interface{}{
			"user_id":   userID.String(),
			"folder_id": folder.ID.String(),
		})
		// Don't fail - folder is created, user can manually trigger sync later
	} else {
		logger.Sync("sync_job_created", "Sync job created for folder", map[string]interface{}{
			"user_id":     userID.String(),
			"folder_id":   folder.ID.String(),
			"folder_name": folder.DriveFolderName,
		})
	}

	// Register webhook for real-time updates
	go func() {
		webhookCtx := context.Background()

		logger.Webhook("register_webhook_start", "Starting webhook registration", map[string]interface{}{
			"folder_id":   folder.ID.String(),
			"folder_name": folder.DriveFolderName,
		})

		startPageToken, err := s.driveClient.GetStartPageToken(webhookCtx, srv)
		if err != nil {
			logGoogleWebhookError("get_page_token_failed", "Failed to get start page token", err, map[string]interface{}{
				"folder_id":   folder.ID.String(),
				"folder_name": folder.DriveFolderName,
			})
			return
		}

		logger.Webhook("page_token_received", "Got start page token", map[string]interface{}{
			"folder_id":  folder.ID.String(),
			"page_token": truncateToken(startPageToken),
		})

		channelID := uuid.New().String()
		channel, err := s.driveClient.WatchChanges(webhookCtx, srv, channelID, webhookToken, startPageToken)
		if err != nil {
			logGoogleWebhookError("register_webhook_failed", "Failed to register webhook", err, map[string]interface{}{
				"folder_id":   folder.ID.String(),
				"folder_name": folder.DriveFolderName,
				"channel_id":  channelID,
			})
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
			logger.WebhookError("update_webhook_info_failed", "Failed to update folder with webhook info", err, map[string]interface{}{
				"folder_id":  folder.ID.String(),
				"channel_id": channel.Id,
			})
			return
		}

		logger.Webhook("webhook_registered", "Webhook registered successfully", map[string]interface{}{
			"folder_id":   folder.ID.String(),
			"folder_name": folder.DriveFolderName,
			"channel_id":  channel.Id,
			"resource_id": channel.ResourceId,
			"expires":     expiry.Format(time.RFC3339),
		})
	}()

	logger.Drive("add_folder_complete", "Add folder process completed", map[string]interface{}{
		"folder_id":       folder.ID.String(),
		"folder_name":     folder.DriveFolderName,
		"drive_folder_id": folder.DriveFolderID,
		"user_id":         userID.String(),
	})

	// Return immediately - photos will sync in background via SyncWorker
	// Frontend will receive WebSocket updates: sync:started, sync:progress, sync:completed
	return folder, nil
}

// truncateToken truncates token for logging
func truncateToken(token string) string {
	if len(token) > 20 {
		return token[:20] + "..."
	}
	return token
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

	logger.Sync("manual_sync_created", "Created sync job for folder", map[string]interface{}{
		"folder_id": folderID.String(),
		"user_id":   userID.String(),
	})
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
	logger.Webhook("shared_folder_webhook_received", "SharedFolder HandleWebhook", map[string]interface{}{
		"channel_id":     channelID,
		"resource_state": resourceState,
		"token_length":   len(token),
	})

	// Find shared folder by webhook token
	folder, err := s.sharedFolderRepo.GetByWebhookToken(ctx, token)
	if err != nil {
		logger.Webhook("shared_folder_webhook_token_not_found", "Webhook token not found for shared folder", map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("webhook token not found: %w", err)
	}

	logger.Webhook("shared_folder_found", "Found shared folder for webhook token", map[string]interface{}{
		"folder_id":   folder.ID.String(),
		"folder_name": folder.DriveFolderName,
	})

	// If it's a sync or change event, trigger sync for this folder
	if resourceState == "sync" || resourceState == "change" {
		logger.Webhook("shared_folder_trigger_sync", "Triggering sync for shared folder", map[string]interface{}{
			"folder_id":      folder.ID.String(),
			"resource_state": resourceState,
		})

		// Get the token owner to trigger sync
		if err := s.triggerSyncForFolder(ctx, folder); err != nil {
			logger.WebhookError("shared_folder_sync_failed", "Failed to trigger sync", err, map[string]interface{}{
				"folder_id": folder.ID.String(),
			})
			return fmt.Errorf("failed to trigger sync: %w", err)
		}
		logger.Webhook("shared_folder_sync_triggered", "Sync triggered successfully for shared folder", map[string]interface{}{
			"folder_id": folder.ID.String(),
		})
	} else {
		logger.Webhook("shared_folder_webhook_ignored", "Ignoring webhook with state", map[string]interface{}{
			"resource_state": resourceState,
		})
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

	logger.Webhook("webhook_registered", "Webhook registered for folder", map[string]interface{}{
		"folder_id":   folder.ID.String(),
		"folder_name": folder.DriveFolderName,
		"channel_id":  channel.Id,
		"expires":     expiry.Format(time.RFC3339),
	})

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

	logger.Sync("webhook_sync_job_created", "Created sync job for shared folder", map[string]interface{}{
		"job_id":    job.ID.String(),
		"folder_id": folder.ID.String(),
	})
	return nil
}
