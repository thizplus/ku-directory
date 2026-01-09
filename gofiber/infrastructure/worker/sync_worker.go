package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/api/drive/v3"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/infrastructure/googledrive"
	"gofiber-template/infrastructure/websocket"
	"gofiber-template/pkg/logger"
)

// SyncWorker processes background sync jobs for shared folders
type SyncWorker struct {
	driveClient      *googledrive.DriveClient
	sharedFolderRepo repositories.SharedFolderRepository
	photoRepo        repositories.PhotoRepository
	syncJobRepo      repositories.SyncJobRepository

	// Worker control
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	isRunning  bool
	mu         sync.Mutex
	triggerCh  chan struct{} // Channel to trigger immediate processing

	// Configuration
	pollInterval    time.Duration
	maxConcurrent   int
	batchSize       int // Batch size for photo creation
	checkpointEvery int // Save checkpoint every N files
	broadcastEvery  int // Broadcast progress every N files
}

// SyncJobMetadata contains metadata for sync jobs
type SyncJobMetadata struct {
	PageToken       string    `json:"page_token,omitempty"`
	CurrentFolder   string    `json:"current_folder,omitempty"`
	ProcessedFiles  int       `json:"processed_files,omitempty"`
	LastProcessedID string    `json:"last_processed_id,omitempty"`
	IsIncremental   bool      `json:"is_incremental,omitempty"`
	SharedFolderID  uuid.UUID `json:"shared_folder_id,omitempty"`
}

// NewSyncWorker creates a new sync worker
func NewSyncWorker(
	driveClient *googledrive.DriveClient,
	sharedFolderRepo repositories.SharedFolderRepository,
	photoRepo repositories.PhotoRepository,
	syncJobRepo repositories.SyncJobRepository,
) *SyncWorker {
	return &SyncWorker{
		driveClient:      driveClient,
		sharedFolderRepo: sharedFolderRepo,
		photoRepo:        photoRepo,
		syncJobRepo:      syncJobRepo,
		triggerCh:        make(chan struct{}, 10), // Buffered channel for triggers
		maxConcurrent:    2,
		batchSize:        100,
		checkpointEvery:  100,
		broadcastEvery:   50,
	}
}

// TriggerSync triggers immediate processing of pending jobs
func (w *SyncWorker) TriggerSync() {
	select {
	case w.triggerCh <- struct{}{}:
		logger.Sync("sync_triggered", "Sync triggered", nil)
	default:
		// Channel full, already triggered
	}
}

// Start starts the sync worker
func (w *SyncWorker) Start() {
	w.mu.Lock()
	if w.isRunning {
		w.mu.Unlock()
		return
	}
	w.isRunning = true
	w.ctx, w.cancel = context.WithCancel(context.Background())
	w.mu.Unlock()

	w.wg.Add(1)
	go w.run()

	logger.Sync("worker_started", "Sync worker started", nil)
}

// Stop stops the sync worker gracefully
func (w *SyncWorker) Stop() {
	w.mu.Lock()
	if !w.isRunning {
		w.mu.Unlock()
		return
	}
	w.isRunning = false
	w.mu.Unlock()

	w.cancel()
	w.wg.Wait()
	logger.Sync("worker_stopped", "Sync worker stopped", nil)
}

// IsRunning returns whether the worker is running
func (w *SyncWorker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.isRunning
}

// run is the main worker loop
func (w *SyncWorker) run() {
	defer w.wg.Done()

	// Process any pending jobs on start
	w.processPendingJobs()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-w.triggerCh:
			// Triggered - process immediately
			w.processPendingJobs()
		}
	}
}

// processPendingJobs fetches and processes pending sync jobs
func (w *SyncWorker) processPendingJobs() {
	jobs, err := w.syncJobRepo.GetPendingJobs(w.ctx, models.SyncJobTypeDriveSync, w.maxConcurrent)
	if err != nil {
		logger.SyncError("fetch_pending_jobs_failed", "Error fetching pending jobs", err, nil)
		return
	}

	if len(jobs) == 0 {
		return
	}

	logger.Sync("processing_jobs", "Processing sync jobs", map[string]interface{}{
		"job_count": len(jobs),
	})

	var jobWg sync.WaitGroup
	sem := make(chan struct{}, w.maxConcurrent)

	for _, job := range jobs {
		sem <- struct{}{}
		jobWg.Add(1)

		go func(j models.SyncJob) {
			defer jobWg.Done()
			defer func() { <-sem }()

			w.processJob(j)
		}(job)
	}

	jobWg.Wait()
}

// processJob processes a single sync job
func (w *SyncWorker) processJob(job models.SyncJob) {
	ctx := w.ctx
	jobID := job.ID

	logger.Sync("job_started", "Sync job started", map[string]interface{}{
		"job_id": jobID.String(),
	})

	// Parse metadata to get SharedFolderID
	var metadata SyncJobMetadata
	if job.Metadata != "" {
		json.Unmarshal([]byte(job.Metadata), &metadata)
	}

	if metadata.SharedFolderID == uuid.Nil {
		logger.SyncError("job_failed", "Missing shared_folder_id in job metadata", nil, map[string]interface{}{
			"job_id": jobID.String(),
		})
		w.failJob(ctx, jobID, nil, "Missing shared_folder_id in job metadata")
		return
	}

	logger.Sync("job_metadata_parsed", "Job metadata parsed", map[string]interface{}{
		"job_id":           jobID.String(),
		"shared_folder_id": metadata.SharedFolderID.String(),
	})

	// Update job status to running
	if err := w.syncJobRepo.UpdateStatus(ctx, jobID, models.SyncJobStatusRunning); err != nil {
		logger.SyncError("update_status_failed", "Failed to update job status", err, map[string]interface{}{
			"job_id": jobID.String(),
		})
		return
	}

	// Get shared folder
	folder, err := w.sharedFolderRepo.GetByID(ctx, metadata.SharedFolderID)
	if err != nil {
		logger.SyncError("get_folder_failed", "Failed to get shared folder", err, map[string]interface{}{
			"job_id":           jobID.String(),
			"shared_folder_id": metadata.SharedFolderID.String(),
		})
		w.failJob(ctx, jobID, nil, fmt.Sprintf("Failed to get shared folder: %v", err))
		return
	}

	logger.Sync("folder_loaded", "Shared folder loaded", map[string]interface{}{
		"job_id":            jobID.String(),
		"folder_id":         folder.ID.String(),
		"folder_name":       folder.DriveFolderName,
		"drive_folder_id":   folder.DriveFolderID,
		"has_page_token":    folder.PageToken != "",
		"has_refresh_token": folder.DriveRefreshToken != "",
	})

	// Broadcast sync started to all users with access
	w.broadcastToFolderUsers(ctx, folder.ID, "sync:started", map[string]interface{}{
		"jobId":    jobID.String(),
		"folderId": folder.ID.String(),
		"status":   "running",
	})

	// Update folder sync status
	w.sharedFolderRepo.UpdateSyncStatus(ctx, folder.ID, models.SyncStatusSyncing, "")

	// Check if folder has valid tokens
	if folder.DriveRefreshToken == "" {
		logger.SyncError("no_oauth_tokens", "Folder has no valid OAuth tokens", nil, map[string]interface{}{
			"job_id":    jobID.String(),
			"folder_id": folder.ID.String(),
		})
		w.failJob(ctx, jobID, &folder.ID, "Folder has no valid OAuth tokens")
		return
	}

	// Get Drive service using folder's tokens
	logger.Sync("get_drive_service", "Getting Google Drive service", map[string]interface{}{
		"job_id":    jobID.String(),
		"folder_id": folder.ID.String(),
	})

	expiry := time.Now()
	if folder.DriveTokenExpiry != nil {
		expiry = *folder.DriveTokenExpiry
	}

	srv, err := w.driveClient.GetDriveServiceWithResourceKey(ctx, folder.DriveAccessToken, folder.DriveRefreshToken, expiry, folder.DriveFolderID, folder.DriveResourceKey)
	if err != nil {
		logger.SyncError("get_drive_service_failed", "Failed to get drive service", err, map[string]interface{}{
			"job_id":    jobID.String(),
			"folder_id": folder.ID.String(),
		})

		// Check if it's a token error and notify users
		errStr := err.Error()
		isTokenError := strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "Invalid Credentials") ||
			strings.Contains(errStr, "token") ||
			strings.Contains(errStr, "oauth")

		if isTokenError {
			// Broadcast token error to all users with access to this folder
			w.broadcastToFolderUsers(ctx, folder.ID, "folder:token_expired", map[string]interface{}{
				"folderId":   folder.ID.String(),
				"folderName": folder.DriveFolderName,
				"message":    "Google Drive token หมดอายุ กรุณา Reconnect",
			})

			// Update folder status with error
			w.sharedFolderRepo.UpdateSyncStatus(ctx, folder.ID, models.SyncStatusError, "Google token expired - please reconnect")
		}

		w.failJob(ctx, jobID, &folder.ID, fmt.Sprintf("Failed to get drive service: %v", err))
		return
	}

	logger.Sync("drive_service_ready", "Google Drive service ready", map[string]interface{}{
		"job_id":    jobID.String(),
		"folder_id": folder.ID.String(),
	})

	// Fetch and update folder metadata (name, description)
	w.updateFolderMetadata(ctx, folder, srv)

	// Decide: Incremental sync or Full sync
	// Use LastSyncedAt to determine if this is first sync (not PageToken, which may be set by webhook registration)
	isFirstSync := folder.LastSyncedAt == nil
	if !isFirstSync && folder.PageToken != "" {
		logger.Sync("sync_mode", "Starting incremental sync", map[string]interface{}{
			"job_id":      jobID.String(),
			"folder_id":   folder.ID.String(),
			"folder_name": folder.DriveFolderName,
			"mode":        "incremental",
		})
		w.processIncrementalSync(ctx, job, folder, srv)
	} else {
		logger.Sync("sync_mode", "Starting full sync", map[string]interface{}{
			"job_id":      jobID.String(),
			"folder_id":   folder.ID.String(),
			"folder_name": folder.DriveFolderName,
			"mode":        "full",
			"reason":      map[bool]string{true: "first_sync", false: "no_page_token"}[isFirstSync],
		})
		w.processFullSync(ctx, job, folder, srv)
	}
}

// truncateString truncates string for logging
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}

// updateFolderMetadata fetches and updates folder metadata (name, description) from Google Drive
func (w *SyncWorker) updateFolderMetadata(ctx context.Context, folder *models.SharedFolder, srv *drive.Service) {
	// Fetch folder metadata from Google Drive (SupportsAllDrives for shared drives)
	folderMeta, err := srv.Files.Get(folder.DriveFolderID).Fields("id, name, description").SupportsAllDrives(true).Do()
	if err != nil {
		logger.SyncError("fetch_folder_metadata_failed", "Failed to fetch folder metadata", err, map[string]interface{}{
			"folder_id":       folder.ID.String(),
			"drive_folder_id": folder.DriveFolderID,
		})
		return
	}

	// Check if name or description changed
	nameChanged := folderMeta.Name != folder.DriveFolderName
	descChanged := folderMeta.Description != folder.Description

	if nameChanged {
		logger.Sync("folder_name_changed", "Folder name changed", map[string]interface{}{
			"folder_id": folder.ID.String(),
			"old_name":  folder.DriveFolderName,
			"new_name":  folderMeta.Name,
		})
	}

	if descChanged {
		logger.Sync("folder_description_changed", "Folder description changed", map[string]interface{}{
			"folder_id":       folder.ID.String(),
			"old_description": truncateString(folder.Description, 100),
			"new_description": truncateString(folderMeta.Description, 100),
		})
	}

	// Update folder if needed - use map to ensure GORM updates all fields
	if nameChanged || descChanged {
		updates := map[string]interface{}{
			"updated_at": time.Now(),
		}
		if nameChanged {
			updates["drive_folder_name"] = folderMeta.Name
			updates["drive_folder_path"] = folderMeta.Name
		}
		if descChanged {
			updates["description"] = folderMeta.Description
		}

		if err := w.sharedFolderRepo.UpdateMetadata(ctx, folder.ID, updates); err != nil {
			logger.SyncError("update_folder_metadata_failed", "Failed to update folder metadata", err, map[string]interface{}{
				"folder_id": folder.ID.String(),
				"error":     err.Error(),
			})
		} else {
			// Update local folder object
			if nameChanged {
				folder.DriveFolderName = folderMeta.Name
				folder.DriveFolderPath = folderMeta.Name
			}
			if descChanged {
				folder.Description = folderMeta.Description
			}
			logger.Sync("folder_metadata_updated", "Folder metadata updated successfully", map[string]interface{}{
				"folder_id":    folder.ID.String(),
				"new_name":     folderMeta.Name,
				"name_changed": nameChanged,
				"desc_changed": descChanged,
			})
		}
	}
}

// processIncrementalSync processes changes since last sync
func (w *SyncWorker) processIncrementalSync(ctx context.Context, job models.SyncJob, folder *models.SharedFolder, srv *drive.Service) {
	jobID := job.ID
	startTime := time.Now()

	logger.Sync("incremental_sync_start", "Starting incremental sync", map[string]interface{}{
		"job_id":      jobID.String(),
		"folder_id":   folder.ID.String(),
		"folder_name": folder.DriveFolderName,
	})

	changes, newPageToken, err := w.driveClient.GetChanges(ctx, srv, folder.PageToken)
	if err != nil {
		logger.SyncError("get_changes_failed", "Failed to get changes", err, map[string]interface{}{
			"job_id":    jobID.String(),
			"folder_id": folder.ID.String(),
		})

		// Check if it's a token error
		errStr := err.Error()
		isTokenError := strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "Invalid Credentials") ||
			strings.Contains(errStr, "authError") ||
			strings.Contains(errStr, "token") ||
			strings.Contains(errStr, "oauth")

		if isTokenError {
			// Broadcast token error to all users with access to this folder
			w.broadcastToFolderUsers(ctx, folder.ID, "folder:token_expired", map[string]interface{}{
				"folderId":   folder.ID.String(),
				"folderName": folder.DriveFolderName,
				"message":    "Google Drive token หมดอายุ กรุณา Reconnect",
			})

			// Update folder status with error and fail the job
			w.sharedFolderRepo.UpdateSyncStatus(ctx, folder.ID, models.SyncStatusError, "Google token expired - please reconnect")
			w.failJob(ctx, jobID, &folder.ID, fmt.Sprintf("Failed to get changes: %v", err))
			return
		}

		// For other errors, fall back to full sync
		w.sharedFolderRepo.Update(ctx, folder.ID, &models.SharedFolder{PageToken: ""})
		w.processFullSync(ctx, job, folder, srv)
		return
	}

	logger.Sync("changes_found", "Found changes", map[string]interface{}{
		"job_id":       jobID.String(),
		"change_count": len(changes),
	})

	if len(changes) == 0 {
		w.sharedFolderRepo.Update(ctx, folder.ID, &models.SharedFolder{PageToken: newPageToken})

		// Mark job as completed
		now := time.Now()
		duration := now.Sub(startTime)
		w.syncJobRepo.Update(ctx, jobID, &models.SyncJob{
			Status:         models.SyncJobStatusCompleted,
			ProcessedItems: 0,
			FailedItems:    0,
			CompletedAt:    &now,
			UpdatedAt:      now,
		})

		// Update folder status
		w.sharedFolderRepo.UpdateSyncStatus(ctx, folder.ID, models.SyncStatusIdle, "")

		// Broadcast completed (no changes)
		w.broadcastToFolderUsers(ctx, folder.ID, "sync:completed", map[string]interface{}{
			"jobId":          jobID.String(),
			"folderId":       folder.ID.String(),
			"processedFiles": 0,
			"newFiles":       0,
			"updatedFiles":   0,
			"deletedFiles":   0,
			"failedFiles":    0,
			"duration":       duration.String(),
			"isIncremental":  true,
			"noChanges":      true,
		})

		logger.Sync("incremental_sync_no_changes", "No changes found in incremental sync", map[string]interface{}{
			"job_id":      jobID.String(),
			"folder_id":   folder.ID.String(),
			"folder_name": folder.DriveFolderName,
			"duration_ms": duration.Milliseconds(),
		})
		return
	}

	logger.Sync("processing_changes", "Processing changes", map[string]interface{}{
		"job_id":       jobID.String(),
		"change_count": len(changes),
	})

	var totalProcessed, totalNew, totalUpdated, totalDeleted, totalFailed int

	w.syncJobRepo.Update(ctx, jobID, &models.SyncJob{
		TotalItems: len(changes),
		UpdatedAt:  time.Now(),
	})

	for i, change := range changes {
		select {
		case <-ctx.Done():
			w.saveIncrementalProgress(ctx, jobID, folder.ID, newPageToken, totalProcessed)
			return
		default:
		}

		if change.Removed || change.File == nil {
			if change.FileId != "" {
				existingPhoto, _ := w.photoRepo.GetByDriveFileID(ctx, change.FileId)
				if existingPhoto != nil {
					w.photoRepo.Delete(ctx, existingPhoto.ID)
					totalDeleted++
				}

				deletedFromFolder, err := w.photoRepo.DeleteByDriveFolderID(ctx, change.FileId)
				if err == nil && deletedFromFolder > 0 {
					totalDeleted += int(deletedFromFolder)
				}
			}
			totalProcessed++
			continue
		}

		file := change.File

		// Handle folder changes (renamed folders)
		if file.MimeType == "application/vnd.google-apps.folder" {
			// Check if this folder is within our root folder
			if w.isWithinRootFolder(ctx, srv, file.Id, folder.DriveFolderID) || file.Id == folder.DriveFolderID {
				// Get the new folder path
				newFolderPath, err := w.driveClient.GetFolderPath(ctx, srv, file.Id, folder.DriveFolderID)
				if err == nil && newFolderPath != "" {
					// Update all photos with this folder ID to have the new path
					updatedCount, err := w.photoRepo.UpdateFolderPath(ctx, file.Id, newFolderPath)
					if err == nil && updatedCount > 0 {
						totalUpdated += int(updatedCount)
						logger.Sync("folder_path_updated", "Updated folder path for photos", map[string]interface{}{
							"job_id":          jobID.String(),
							"drive_folder_id": file.Id,
							"new_path":        newFolderPath,
							"photo_count":     updatedCount,
						})
					}
				}
			}
			totalProcessed++
			continue
		}

		if file.MimeType == "" || !isImageMimeType(file.MimeType) {
			totalProcessed++
			continue
		}

		if file.Trashed {
			existingPhoto, _ := w.photoRepo.GetByDriveFileID(ctx, file.Id)
			if existingPhoto != nil {
				w.photoRepo.Delete(ctx, existingPhoto.ID)
				totalDeleted++
			}
			totalProcessed++
			continue
		}

		parentID := ""
		if len(file.Parents) > 0 {
			parentID = file.Parents[0]
		}

		if !w.isWithinRootFolder(ctx, srv, parentID, folder.DriveFolderID) {
			totalProcessed++
			continue
		}

		existingPhoto, _ := w.photoRepo.GetByDriveFileID(ctx, file.Id)
		if existingPhoto != nil {
			folderPath, _ := w.driveClient.GetFolderPath(ctx, srv, parentID, folder.DriveFolderID)
			modifiedTime, _ := time.Parse(time.RFC3339, file.ModifiedTime)

			existingPhoto.FileName = file.Name
			existingPhoto.ThumbnailURL = file.ThumbnailLink
			existingPhoto.WebViewURL = file.WebViewLink
			existingPhoto.DriveFolderID = parentID
			existingPhoto.DriveFolderPath = folderPath
			existingPhoto.DriveModifiedAt = &modifiedTime
			existingPhoto.UpdatedAt = time.Now()
			w.photoRepo.Update(ctx, existingPhoto.ID, existingPhoto)
			totalUpdated++
		} else {
			folderPath, _ := w.driveClient.GetFolderPath(ctx, srv, parentID, folder.DriveFolderID)
			createdTime, _ := time.Parse(time.RFC3339, file.CreatedTime)
			modifiedTime, _ := time.Parse(time.RFC3339, file.ModifiedTime)

			photo := &models.Photo{
				ID:              uuid.New(),
				SharedFolderID:  folder.ID,
				DriveFileID:     file.Id,
				DriveFolderID:   parentID,
				DriveFolderPath: folderPath,
				FileName:        file.Name,
				MimeType:        file.MimeType,
				FileSize:        file.Size,
				ThumbnailURL:    file.ThumbnailLink,
				WebViewURL:      file.WebViewLink,
				DriveCreatedAt:  &createdTime,
				DriveModifiedAt: &modifiedTime,
				FaceStatus:      models.FaceStatusPending,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			if err := w.photoRepo.Create(ctx, photo); err != nil {
				logger.SyncError("photo_create_failed", "Error creating photo", err, map[string]interface{}{
					"job_id":   jobID.String(),
					"photo_id": photo.ID.String(),
				})
				totalFailed++
			} else {
				totalNew++
				w.broadcastToFolderUsers(ctx, folder.ID, "photos:added", map[string]interface{}{
					"count":    1,
					"photoIds": []string{photo.ID.String()},
				})
			}
		}
		totalProcessed++

		if (i+1)%w.broadcastEvery == 0 || i == len(changes)-1 {
			w.broadcastToFolderUsers(ctx, folder.ID, "sync:progress", map[string]interface{}{
				"jobId":          jobID.String(),
				"processedFiles": totalProcessed,
				"totalFiles":     len(changes),
				"newFiles":       totalNew,
				"updatedFiles":   totalUpdated,
				"deletedFiles":   totalDeleted,
				"failedFiles":    totalFailed,
				"isIncremental":  true,
			})
		}
	}

	// Save new page token
	w.sharedFolderRepo.Update(ctx, folder.ID, &models.SharedFolder{PageToken: newPageToken})

	// Mark job as completed
	now := time.Now()
	duration := now.Sub(startTime)
	w.syncJobRepo.Update(ctx, jobID, &models.SyncJob{
		Status:         models.SyncJobStatusCompleted,
		ProcessedItems: totalProcessed,
		FailedItems:    totalFailed,
		CompletedAt:    &now,
		UpdatedAt:      now,
	})

	// Update folder status
	w.sharedFolderRepo.UpdateSyncStatus(ctx, folder.ID, models.SyncStatusIdle, "")

	// Broadcast completed
	w.broadcastToFolderUsers(ctx, folder.ID, "sync:completed", map[string]interface{}{
		"jobId":          jobID.String(),
		"folderId":       folder.ID.String(),
		"status":         "completed",
		"processedFiles": totalProcessed,
		"totalFiles":     len(changes),
		"newFiles":       totalNew,
		"updatedFiles":   totalUpdated,
		"deletedFiles":   totalDeleted,
		"failedFiles":    totalFailed,
		"isIncremental":  true,
	})

	logger.Sync("incremental_sync_completed", "Incremental sync completed", map[string]interface{}{
		"job_id":        jobID.String(),
		"folder_id":     folder.ID.String(),
		"folder_name":   folder.DriveFolderName,
		"duration_ms":   duration.Milliseconds(),
		"total_changes": len(changes),
		"new_files":     totalNew,
		"updated_files": totalUpdated,
		"deleted_files": totalDeleted,
		"failed_files":  totalFailed,
	})
}

// processFullSync does a full sync of all images
func (w *SyncWorker) processFullSync(ctx context.Context, job models.SyncJob, folder *models.SharedFolder, srv *drive.Service) {
	jobID := job.ID
	startTime := time.Now()

	logger.Sync("full_sync_start", "Starting full sync", map[string]interface{}{
		"job_id":      jobID.String(),
		"folder_id":   folder.ID.String(),
		"folder_name": folder.DriveFolderName,
	})

	var metadata SyncJobMetadata
	if job.Metadata != "" {
		json.Unmarshal([]byte(job.Metadata), &metadata)
	}

	totalProcessed := metadata.ProcessedFiles
	totalFailed := job.FailedItems
	totalNew := 0
	totalUpdated := 0
	totalDeleted := 0

	// Step 1: List ALL folders first for path mapping (optimization)
	allFolders, err := w.driveClient.ListAllFoldersRecursive(ctx, srv, folder.DriveFolderID)
	if err != nil {
		logger.SyncError("list_folders_failed", "Failed to list folders (will use API per photo)", err, map[string]interface{}{
			"job_id":    jobID.String(),
			"folder_id": folder.ID.String(),
		})
		allFolders = nil
	} else {
		logger.Sync("folders_listed", "Listed folders from Drive", map[string]interface{}{
			"job_id":       jobID.String(),
			"folder_count": len(allFolders),
		})
	}

	// Step 2: Build folder path map (O(1) lookup)
	var folderPathMap map[string]string
	if allFolders != nil {
		folderPathMap = w.driveClient.BuildFolderPathMap(allFolders, folder.DriveFolderID)
	}

	// Step 3: List all images
	files, err := w.driveClient.ListAllImagesRecursive(ctx, srv, folder.DriveFolderID)
	if err != nil {
		logger.SyncError("list_images_failed", "Failed to list images", err, map[string]interface{}{
			"job_id":    jobID.String(),
			"folder_id": folder.ID.String(),
		})

		// Check if it's a token error and notify users
		errStr := err.Error()
		isTokenError := strings.Contains(errStr, "401") ||
			strings.Contains(errStr, "Invalid Credentials") ||
			strings.Contains(errStr, "authError") ||
			strings.Contains(errStr, "token") ||
			strings.Contains(errStr, "oauth")

		if isTokenError {
			// Broadcast token error to all users with access to this folder
			w.broadcastToFolderUsers(ctx, folder.ID, "folder:token_expired", map[string]interface{}{
				"folderId":   folder.ID.String(),
				"folderName": folder.DriveFolderName,
				"message":    "Google Drive token หมดอายุ กรุณา Reconnect",
			})

			// Update folder status with error
			w.sharedFolderRepo.UpdateSyncStatus(ctx, folder.ID, models.SyncStatusError, "Google token expired - please reconnect")
		}

		w.failJob(ctx, jobID, &folder.ID, fmt.Sprintf("Failed to list files: %v", err))
		return
	}
	logger.Sync("images_listed", "Listed images from Drive", map[string]interface{}{
		"job_id":      jobID.String(),
		"image_count": len(files),
	})

	driveFileIDs := make([]string, 0, len(files))
	for _, file := range files {
		driveFileIDs = append(driveFileIDs, file.ID)
	}

	totalItems := len(files)
	w.syncJobRepo.Update(ctx, jobID, &models.SyncJob{
		TotalItems: totalItems,
		UpdatedAt:  time.Now(),
	})

	startIndex := 0
	if metadata.LastProcessedID != "" {
		for i, file := range files {
			if file.ID == metadata.LastProcessedID {
				startIndex = i + 1
				break
			}
		}
	}

	photoBatch := make([]*models.Photo, 0, w.batchSize)
	newPhotoIDs := make([]string, 0, w.batchSize)
	lastBroadcastPercent := 0

	for i := startIndex; i < len(files); i++ {
		file := files[i]

		select {
		case <-ctx.Done():
			if len(photoBatch) > 0 {
				w.flushPhotoBatch(ctx, photoBatch, &totalNew, &totalFailed)
			}
			metadata.LastProcessedID = file.ID
			w.saveProgress(ctx, jobID, totalProcessed, totalFailed, metadata)
			return
		default:
		}

		// Get folder path from map (O(1)) or fallback to API
		var folderPath string
		if folderPathMap != nil {
			folderPath = folderPathMap[file.ParentID]
		} else {
			folderPath, _ = w.driveClient.GetFolderPath(ctx, srv, file.ParentID, folder.DriveFolderID)
		}

		existingPhoto, _ := w.photoRepo.GetByDriveFileID(ctx, file.ID)
		if existingPhoto != nil {
			needsUpdate := file.ModifiedTime.After(existingPhoto.UpdatedAt) ||
				existingPhoto.DriveFolderID != file.ParentID ||
				existingPhoto.DriveFolderPath != folderPath

			if needsUpdate {
				existingPhoto.FileName = file.Name
				existingPhoto.ThumbnailURL = file.ThumbnailURL
				existingPhoto.WebViewURL = file.WebViewURL
				existingPhoto.DriveFolderID = file.ParentID
				existingPhoto.DriveFolderPath = folderPath
				existingPhoto.DriveModifiedAt = &file.ModifiedTime
				existingPhoto.UpdatedAt = time.Now()
				w.photoRepo.Update(ctx, existingPhoto.ID, existingPhoto)
				totalUpdated++
			}
			totalProcessed++
		} else {
			photo := &models.Photo{
				ID:              uuid.New(),
				SharedFolderID:  folder.ID,
				DriveFileID:     file.ID,
				DriveFolderID:   file.ParentID,
				DriveFolderPath: folderPath,
				FileName:        file.Name,
				MimeType:        file.MimeType,
				FileSize:        file.Size,
				ThumbnailURL:    file.ThumbnailURL,
				WebViewURL:      file.WebViewURL,
				DriveCreatedAt:  &file.CreatedTime,
				DriveModifiedAt: &file.ModifiedTime,
				FaceStatus:      models.FaceStatusPending,
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			}

			photoBatch = append(photoBatch, photo)
			newPhotoIDs = append(newPhotoIDs, photo.ID.String())
			totalProcessed++
		}

		if len(photoBatch) >= w.batchSize {
			w.flushPhotoBatch(ctx, photoBatch, &totalNew, &totalFailed)

			if len(newPhotoIDs) > 0 {
				w.broadcastToFolderUsers(ctx, folder.ID, "photos:added", map[string]interface{}{
					"count":    len(newPhotoIDs),
					"photoIds": newPhotoIDs,
				})
			}

			photoBatch = make([]*models.Photo, 0, w.batchSize)
			newPhotoIDs = make([]string, 0, w.batchSize)
		}

		// Send progress every 5%
		currentPercent := 0
		if totalItems > 0 {
			currentPercent = (totalProcessed * 100) / totalItems
		}
		if currentPercent >= lastBroadcastPercent+5 || i == totalItems-1 {
			lastBroadcastPercent = currentPercent
			w.syncJobRepo.UpdateProgress(ctx, jobID, totalProcessed, totalFailed)

			w.broadcastToFolderUsers(ctx, folder.ID, "sync:progress", map[string]interface{}{
				"jobId":          jobID.String(),
				"folderId":       folder.ID.String(),
				"processedFiles": totalProcessed,
				"totalFiles":     totalItems,
				"percent":        currentPercent,
				"newFiles":       totalNew,
				"updatedFiles":   totalUpdated,
				"failedFiles":    totalFailed,
			})
		}

		if (i+1)%w.checkpointEvery == 0 {
			metadata.LastProcessedID = file.ID
			metadata.ProcessedFiles = totalProcessed
			w.saveCheckpoint(ctx, jobID, totalProcessed, totalFailed, metadata)
		}
	}

	if len(photoBatch) > 0 {
		w.flushPhotoBatch(ctx, photoBatch, &totalNew, &totalFailed)

		if len(newPhotoIDs) > 0 {
			w.broadcastToFolderUsers(ctx, folder.ID, "photos:added", map[string]interface{}{
				"count":    len(newPhotoIDs),
				"photoIds": newPhotoIDs,
			})
		}
	}

	// Cleanup orphaned photos
	deletedCount, err := w.photoRepo.DeleteNotInDriveIDsForFolder(ctx, folder.ID, driveFileIDs)
	if err != nil {
		logger.SyncError("cleanup_orphaned_failed", "Failed to cleanup orphaned photos", err, map[string]interface{}{
			"job_id":    jobID.String(),
			"folder_id": folder.ID.String(),
		})
	} else if deletedCount > 0 {
		totalDeleted = int(deletedCount)
		logger.Sync("orphaned_photos_deleted", "Cleaned up orphaned photos", map[string]interface{}{
			"job_id":        jobID.String(),
			"folder_id":     folder.ID.String(),
			"deleted_count": deletedCount,
		})

		w.broadcastToFolderUsers(ctx, folder.ID, "photos:deleted", map[string]interface{}{
			"count":  deletedCount,
			"reason": "cleanup_orphaned",
		})
	}

	// Get and save page token
	pageToken, err := w.driveClient.GetStartPageToken(ctx, srv)
	if err != nil {
		logger.SyncError("get_page_token_failed", "Failed to get page token", err, map[string]interface{}{
			"job_id":    jobID.String(),
			"folder_id": folder.ID.String(),
		})
	} else {
		w.sharedFolderRepo.Update(ctx, folder.ID, &models.SharedFolder{PageToken: pageToken})
	}

	// Mark job as completed
	now := time.Now()
	duration := now.Sub(startTime)
	w.syncJobRepo.Update(ctx, jobID, &models.SyncJob{
		Status:         models.SyncJobStatusCompleted,
		ProcessedItems: totalProcessed,
		FailedItems:    totalFailed,
		CompletedAt:    &now,
		UpdatedAt:      now,
	})

	// Update folder status
	w.sharedFolderRepo.UpdateSyncStatus(ctx, folder.ID, models.SyncStatusIdle, "")

	// Broadcast completed
	w.broadcastToFolderUsers(ctx, folder.ID, "sync:completed", map[string]interface{}{
		"jobId":          jobID.String(),
		"folderId":       folder.ID.String(),
		"status":         "completed",
		"processedFiles": totalProcessed,
		"totalFiles":     totalItems,
		"newFiles":       totalNew,
		"deletedFiles":   totalDeleted,
		"failedFiles":    totalFailed,
	})

	logger.Sync("full_sync_completed", "Full sync completed", map[string]interface{}{
		"job_id":          jobID.String(),
		"folder_id":       folder.ID.String(),
		"folder_name":     folder.DriveFolderName,
		"duration_ms":     duration.Milliseconds(),
		"total_files":     totalItems,
		"processed_files": totalProcessed,
		"new_files":       totalNew,
		"updated_files":   totalUpdated,
		"deleted_files":   totalDeleted,
		"failed_files":    totalFailed,
	})
}

// broadcastToFolderUsers broadcasts a message to all users with access to a folder
func (w *SyncWorker) broadcastToFolderUsers(ctx context.Context, folderID uuid.UUID, messageType string, data map[string]interface{}) {
	users, err := w.sharedFolderRepo.GetUsersByFolder(ctx, folderID)
	if err != nil {
		return
	}

	for _, user := range users {
		websocket.Manager.BroadcastToUser(user.ID, messageType, data)
	}
}

// flushPhotoBatch inserts a batch of photos
func (w *SyncWorker) flushPhotoBatch(ctx context.Context, photos []*models.Photo, totalNew *int, totalFailed *int) {
	if len(photos) == 0 {
		return
	}

	if err := w.photoRepo.CreateBatch(ctx, photos); err != nil {
		logger.SyncError("batch_create_failed", "Error batch creating photos", err, map[string]interface{}{
			"batch_size": len(photos),
		})
		*totalFailed += len(photos)
	} else {
		*totalNew += len(photos)
	}
}

// failJob marks a job as failed
func (w *SyncWorker) failJob(ctx context.Context, jobID uuid.UUID, folderID *uuid.UUID, errMsg string) {
	logData := map[string]interface{}{
		"job_id": jobID.String(),
		"error":  errMsg,
	}
	if folderID != nil {
		logData["folder_id"] = folderID.String()
	}
	logger.SyncError("job_failed", "Sync job failed", nil, logData)

	now := time.Now()
	w.syncJobRepo.Update(ctx, jobID, &models.SyncJob{
		Status:      models.SyncJobStatusFailed,
		LastError:   errMsg,
		CompletedAt: &now,
		UpdatedAt:   now,
	})

	if folderID != nil {
		w.sharedFolderRepo.UpdateSyncStatus(ctx, *folderID, models.SyncStatusError, errMsg)

		w.broadcastToFolderUsers(ctx, *folderID, "sync:failed", map[string]interface{}{
			"jobId":    jobID.String(),
			"folderId": folderID.String(),
			"status":   "failed",
			"message":  errMsg,
		})
	}
}

// saveProgress saves progress for resuming
func (w *SyncWorker) saveProgress(ctx context.Context, jobID uuid.UUID, processed, failed int, metadata SyncJobMetadata) {
	metadata.ProcessedFiles = processed
	metadataJSON, _ := json.Marshal(metadata)

	w.syncJobRepo.Update(ctx, jobID, &models.SyncJob{
		Status:         models.SyncJobStatusPending,
		ProcessedItems: processed,
		FailedItems:    failed,
		Metadata:       string(metadataJSON),
		UpdatedAt:      time.Now(),
	})
}

// saveCheckpoint saves checkpoint without changing status
func (w *SyncWorker) saveCheckpoint(ctx context.Context, jobID uuid.UUID, processed, failed int, metadata SyncJobMetadata) {
	metadataJSON, _ := json.Marshal(metadata)

	w.syncJobRepo.Update(ctx, jobID, &models.SyncJob{
		ProcessedItems: processed,
		FailedItems:    failed,
		Metadata:       string(metadataJSON),
		UpdatedAt:      time.Now(),
	})
}

// saveIncrementalProgress saves progress for incremental sync
func (w *SyncWorker) saveIncrementalProgress(ctx context.Context, jobID uuid.UUID, folderID uuid.UUID, pageToken string, processed int) {
	w.sharedFolderRepo.Update(ctx, folderID, &models.SharedFolder{PageToken: pageToken})

	w.syncJobRepo.Update(ctx, jobID, &models.SyncJob{
		Status:         models.SyncJobStatusPending,
		ProcessedItems: processed,
		UpdatedAt:      time.Now(),
	})
}

// isImageMimeType checks if the mime type is an image
func isImageMimeType(mimeType string) bool {
	return strings.HasPrefix(mimeType, "image/")
}

// isWithinRootFolder checks if a folder is within the root folder
func (w *SyncWorker) isWithinRootFolder(ctx context.Context, srv *drive.Service, folderID, rootFolderID string) bool {
	if folderID == "" {
		return false
	}
	if folderID == rootFolderID {
		return true
	}

	currentID := folderID
	visited := make(map[string]bool)

	for i := 0; i < 20; i++ {
		if visited[currentID] {
			return false
		}
		visited[currentID] = true

		if currentID == rootFolderID {
			return true
		}

		folder, err := srv.Files.Get(currentID).Fields("parents").SupportsAllDrives(true).Do()
		if err != nil {
			return false
		}

		if len(folder.Parents) == 0 {
			return false
		}

		currentID = folder.Parents[0]
	}

	return false
}
