package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
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
		log.Println("🔔 Sync triggered")
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

	log.Println("✓ Sync worker started")
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
	log.Println("✓ Sync worker stopped")
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
		log.Printf("Error fetching pending jobs: %v", err)
		return
	}

	if len(jobs) == 0 {
		return
	}

	log.Printf("Processing %d sync jobs", len(jobs))

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

	log.Println("╔════════════════════════════════════════════════════════════════")
	log.Printf("║ SYNC JOB STARTED: %s", jobID)
	log.Println("╠════════════════════════════════════════════════════════════════")

	// Parse metadata to get SharedFolderID
	log.Println("║ [STEP 1/6] Parsing job metadata...")
	var metadata SyncJobMetadata
	if job.Metadata != "" {
		json.Unmarshal([]byte(job.Metadata), &metadata)
	}

	if metadata.SharedFolderID == uuid.Nil {
		log.Println("║ ❌ FAILED: Missing shared_folder_id in job metadata")
		log.Println("╚════════════════════════════════════════════════════════════════")
		w.failJob(ctx, jobID, nil, "Missing shared_folder_id in job metadata")
		return
	}
	log.Printf("║ ✓ SharedFolderID: %s", metadata.SharedFolderID)

	// Update job status to running
	log.Println("║ [STEP 2/6] Updating job status to RUNNING...")
	if err := w.syncJobRepo.UpdateStatus(ctx, jobID, models.SyncJobStatusRunning); err != nil {
		log.Printf("║ ❌ Error updating job status: %v", err)
		log.Println("╚════════════════════════════════════════════════════════════════")
		return
	}
	log.Println("║ ✓ Job status updated")

	// Get shared folder
	log.Println("║ [STEP 3/6] Fetching shared folder from database...")
	folder, err := w.sharedFolderRepo.GetByID(ctx, metadata.SharedFolderID)
	if err != nil {
		log.Printf("║ ❌ Failed to get shared folder: %v", err)
		log.Println("╚════════════════════════════════════════════════════════════════")
		w.failJob(ctx, jobID, nil, fmt.Sprintf("Failed to get shared folder: %v", err))
		return
	}
	log.Printf("║ ✓ Folder: %s (%s)", folder.DriveFolderName, folder.DriveFolderID)
	log.Printf("║   - PageToken: %s", truncateString(folder.PageToken, 20))
	log.Printf("║   - HasRefreshToken: %v", folder.DriveRefreshToken != "")

	// Broadcast sync started to all users with access
	log.Println("║ [STEP 4/6] Broadcasting sync:started to users...")
	w.broadcastToFolderUsers(ctx, folder.ID, "sync:started", map[string]interface{}{
		"jobId":    jobID.String(),
		"folderId": folder.ID.String(),
		"status":   "running",
	})
	log.Println("║ ✓ Broadcast sent")

	// Update folder sync status
	w.sharedFolderRepo.UpdateSyncStatus(ctx, folder.ID, models.SyncStatusSyncing, "")

	// Check if folder has valid tokens
	log.Println("║ [STEP 5/6] Validating OAuth tokens...")
	if folder.DriveRefreshToken == "" {
		log.Println("║ ❌ Folder has no valid OAuth tokens")
		log.Println("╚════════════════════════════════════════════════════════════════")
		w.failJob(ctx, jobID, &folder.ID, "Folder has no valid OAuth tokens")
		return
	}
	log.Println("║ ✓ OAuth tokens present")

	// Get Drive service using folder's tokens
	log.Println("║ [STEP 6/6] Creating Google Drive service...")
	expiry := time.Now()
	if folder.DriveTokenExpiry != nil {
		expiry = *folder.DriveTokenExpiry
	}

	srv, err := w.driveClient.GetDriveService(ctx, folder.DriveAccessToken, folder.DriveRefreshToken, expiry)
	if err != nil {
		log.Printf("║ ❌ Failed to get drive service: %v", err)
		log.Println("╚════════════════════════════════════════════════════════════════")
		w.failJob(ctx, jobID, &folder.ID, fmt.Sprintf("Failed to get drive service: %v", err))
		return
	}
	log.Println("║ ✓ Drive service created")
	log.Println("╠════════════════════════════════════════════════════════════════")

	// Decide: Incremental sync or Full sync
	if folder.PageToken != "" {
		log.Println("║ MODE: INCREMENTAL SYNC (has page token)")
		log.Println("╚════════════════════════════════════════════════════════════════")
		w.processIncrementalSync(ctx, job, folder, srv)
	} else {
		log.Println("║ MODE: FULL SYNC (no page token)")
		log.Println("╚════════════════════════════════════════════════════════════════")
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

// processIncrementalSync processes changes since last sync
func (w *SyncWorker) processIncrementalSync(ctx context.Context, job models.SyncJob, folder *models.SharedFolder, srv *drive.Service) {
	jobID := job.ID
	startTime := time.Now()

	log.Println("┌────────────────────────────────────────────────────────────────")
	log.Printf("│ INCREMENTAL SYNC: %s", folder.DriveFolderName)
	log.Printf("│ Job ID: %s", jobID)
	log.Println("├────────────────────────────────────────────────────────────────")

	log.Println("│ [1/3] Fetching changes from Google Drive...")
	changes, newPageToken, err := w.driveClient.GetChanges(ctx, srv, folder.PageToken)
	if err != nil {
		log.Printf("│ ⚠ Failed to get changes: %v", err)
		log.Println("│ → Falling back to full sync...")
		log.Println("└────────────────────────────────────────────────────────────────")
		w.sharedFolderRepo.Update(ctx, folder.ID, &models.SharedFolder{PageToken: ""})
		w.processFullSync(ctx, job, folder, srv)
		return
	}

	log.Printf("│ ✓ Found %d changes", len(changes))

	if len(changes) == 0 {
		log.Println("│ → No changes to process")
		log.Println("│ [2/3] Saving new page token...")
		w.sharedFolderRepo.Update(ctx, folder.ID, &models.SharedFolder{PageToken: newPageToken})
		log.Printf("│ ✓ Page token saved: %s", truncateString(newPageToken, 20))

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
		log.Printf("│ ✓ Job completed in %v", duration)

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

		// Log to file
		logger.Sync("INCREMENTAL_SYNC_NO_CHANGES", "No changes found in incremental sync", map[string]interface{}{
			"job_id":     jobID.String(),
			"folder_id":  folder.ID.String(),
			"duration":   duration.String(),
			"page_token": truncateString(newPageToken, 20),
		})

		log.Println("└────────────────────────────────────────────────────────────────")
		return
	}

	log.Printf("│ [2/3] Processing %d changes...", len(changes))

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
			folderPath, _ := w.driveClient.GetFolderPath(ctx, srv, parentID)
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
			folderPath, _ := w.driveClient.GetFolderPath(ctx, srv, parentID)
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
				log.Printf("Error creating photo: %v", err)
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
	log.Println("│ [3/3] Saving new page token...")
	w.sharedFolderRepo.Update(ctx, folder.ID, &models.SharedFolder{PageToken: newPageToken})
	log.Printf("│ ✓ Page token saved: %s", truncateString(newPageToken, 20))

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

	// Summary log
	log.Println("├────────────────────────────────────────────────────────────────")
	log.Println("│ INCREMENTAL SYNC COMPLETED")
	log.Println("├────────────────────────────────────────────────────────────────")
	log.Printf("│ Duration:   %v", duration.Round(time.Millisecond))
	log.Printf("│ Changes:    %d total", len(changes))
	log.Printf("│ New:        %d files", totalNew)
	log.Printf("│ Updated:    %d files", totalUpdated)
	log.Printf("│ Deleted:    %d files", totalDeleted)
	log.Printf("│ Failed:     %d files", totalFailed)
	log.Println("└────────────────────────────────────────────────────────────────")
}

// processFullSync does a full sync of all images
func (w *SyncWorker) processFullSync(ctx context.Context, job models.SyncJob, folder *models.SharedFolder, srv *drive.Service) {
	jobID := job.ID
	startTime := time.Now()

	log.Println("┌────────────────────────────────────────────────────────────────")
	log.Printf("│ FULL SYNC: %s", folder.DriveFolderName)
	log.Printf("│ Job ID: %s", jobID)
	log.Println("├────────────────────────────────────────────────────────────────")

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
	log.Printf("│ [1/5] Listing all folders from Drive...")
	allFolders, err := w.driveClient.ListAllFoldersRecursive(ctx, srv, folder.DriveFolderID)
	if err != nil {
		log.Printf("│ ⚠ Warning: Failed to list folders: %v (will use API per photo)", err)
		allFolders = nil
	} else {
		log.Printf("│ ✓ Found %d folders", len(allFolders))
	}

	// Step 2: Build folder path map (O(1) lookup)
	var folderPathMap map[string]string
	if allFolders != nil {
		folderPathMap = w.driveClient.BuildFolderPathMap(allFolders, folder.DriveFolderID)
		log.Printf("│ ✓ Built path map for %d folders", len(folderPathMap))
	}

	// Step 3: List all images
	log.Printf("│ [2/5] Listing images from Drive folder: %s", folder.DriveFolderID)
	files, err := w.driveClient.ListAllImagesRecursive(ctx, srv, folder.DriveFolderID)
	if err != nil {
		log.Printf("│ ❌ Failed to list files: %v", err)
		log.Println("└────────────────────────────────────────────────────────────────")
		w.failJob(ctx, jobID, &folder.ID, fmt.Sprintf("Failed to list files: %v", err))
		return
	}
	log.Printf("│ ✓ Found %d images in Drive", len(files))

	driveFileIDs := make([]string, 0, len(files))
	for _, file := range files {
		driveFileIDs = append(driveFileIDs, file.ID)
	}

	totalItems := len(files)
	w.syncJobRepo.Update(ctx, jobID, &models.SyncJob{
		TotalItems: totalItems,
		UpdatedAt:  time.Now(),
	})

	log.Printf("│ [3/5] Processing %d images...", totalItems)

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
			folderPath, _ = w.driveClient.GetFolderPath(ctx, srv, file.ParentID)
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
			log.Printf("│ 📊 Progress: %d%% (%d/%d)", currentPercent, totalProcessed, totalItems)
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
	log.Printf("│ [4/5] Cleaning up orphaned photos...")
	deletedCount, err := w.photoRepo.DeleteNotInDriveIDsForFolder(ctx, folder.ID, driveFileIDs)
	if err != nil {
		log.Printf("│ ⚠ Warning: Failed to cleanup orphaned photos: %v", err)
	} else if deletedCount > 0 {
		totalDeleted = int(deletedCount)
		log.Printf("│ ✓ Cleaned up %d orphaned photos", deletedCount)

		w.broadcastToFolderUsers(ctx, folder.ID, "photos:deleted", map[string]interface{}{
			"count":  deletedCount,
			"reason": "cleanup_orphaned",
		})
	} else {
		log.Println("│ ✓ No orphaned photos to clean up")
	}

	// Get and save page token
	log.Printf("│ [5/5] Saving page token for future incremental sync...")
	pageToken, err := w.driveClient.GetStartPageToken(ctx, srv)
	if err != nil {
		log.Printf("│ ⚠ Warning: Failed to get page token: %v", err)
	} else {
		w.sharedFolderRepo.Update(ctx, folder.ID, &models.SharedFolder{PageToken: pageToken})
		log.Printf("│ ✓ Page token saved: %s", truncateString(pageToken, 20))
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

	// Summary log
	log.Println("├────────────────────────────────────────────────────────────────")
	log.Println("│ FULL SYNC COMPLETED")
	log.Println("├────────────────────────────────────────────────────────────────")
	log.Printf("│ Duration:   %v", duration.Round(time.Millisecond))
	log.Printf("│ Processed:  %d files", totalProcessed)
	log.Printf("│ New:        %d files", totalNew)
	log.Printf("│ Updated:    %d files", totalUpdated)
	log.Printf("│ Deleted:    %d files", totalDeleted)
	log.Printf("│ Failed:     %d files", totalFailed)
	log.Println("└────────────────────────────────────────────────────────────────")

	// File-based logging
	logger.Sync("FULL_SYNC_COMPLETED", "Full sync job completed", map[string]interface{}{
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
		log.Printf("Warning: Failed to get users for folder %s: %v", folderID, err)
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
		log.Printf("Error batch creating %d photos: %v", len(photos), err)
		*totalFailed += len(photos)
	} else {
		*totalNew += len(photos)
		log.Printf("Batch created %d photos", len(photos))
	}
}

// failJob marks a job as failed
func (w *SyncWorker) failJob(ctx context.Context, jobID uuid.UUID, folderID *uuid.UUID, errMsg string) {
	log.Printf("Sync job %s failed: %s", jobID, errMsg)

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
