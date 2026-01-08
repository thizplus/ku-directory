package worker

import (
	"context"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/infrastructure/faceapi"
	"gofiber-template/infrastructure/googledrive"
	"gofiber-template/infrastructure/websocket"
	"gofiber-template/pkg/logger"
)

// FaceWorker processes photos for face detection
type FaceWorker struct {
	faceClient       *faceapi.FaceClient
	driveClient      *googledrive.DriveClient
	userRepo         repositories.UserRepository
	photoRepo        repositories.PhotoRepository
	faceRepo         repositories.FaceRepository
	sharedFolderRepo repositories.SharedFolderRepository

	// Worker control
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	isRunning bool
	mu        sync.Mutex

	// Configuration
	pollInterval  time.Duration
	maxConcurrent int
	batchSize     int

	// Retry configuration
	maxRetries     int
	baseRetryDelay time.Duration

	// Circuit breaker
	circuitBreaker *CircuitBreaker
}

// CircuitBreaker prevents cascading failures
type CircuitBreaker struct {
	failures     int32
	threshold    int32
	resetTimeout time.Duration
	lastFailure  time.Time
	mu           sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(threshold int32, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		threshold:    threshold,
		resetTimeout: resetTimeout,
	}
}

// IsOpen returns true if circuit is open (should not proceed)
func (cb *CircuitBreaker) IsOpen() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	if atomic.LoadInt32(&cb.failures) >= cb.threshold {
		// Check if reset timeout has passed
		if time.Since(cb.lastFailure) > cb.resetTimeout {
			// Allow one request through (half-open state)
			return false
		}
		return true
	}
	return false
}

// RecordSuccess resets the failure count
func (cb *CircuitBreaker) RecordSuccess() {
	atomic.StoreInt32(&cb.failures, 0)
}

// RecordFailure increments failure count
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	atomic.AddInt32(&cb.failures, 1)
	cb.lastFailure = time.Now()
}

// GetFailures returns current failure count
func (cb *CircuitBreaker) GetFailures() int32 {
	return atomic.LoadInt32(&cb.failures)
}

// NewFaceWorker creates a new face processing worker
func NewFaceWorker(
	faceClient *faceapi.FaceClient,
	driveClient *googledrive.DriveClient,
	userRepo repositories.UserRepository,
	photoRepo repositories.PhotoRepository,
	faceRepo repositories.FaceRepository,
	sharedFolderRepo repositories.SharedFolderRepository,
) *FaceWorker {
	return &FaceWorker{
		faceClient:       faceClient,
		driveClient:      driveClient,
		userRepo:         userRepo,
		photoRepo:        photoRepo,
		faceRepo:         faceRepo,
		sharedFolderRepo: sharedFolderRepo,
		pollInterval:     10 * time.Second,  // Reduced from 15s for faster processing
		maxConcurrent:    3,                 // Reduced for CPU-based Face API (prevent overload)
		batchSize:        20,                // Reduced batch size for stability
		maxRetries:       3,                 // Retry failed operations
		baseRetryDelay:   2 * time.Second,   // Base delay for exponential backoff
		circuitBreaker:   NewCircuitBreaker(10, 60*time.Second), // Open after 10 failures, reset after 60s
	}
}

// Start starts the face worker
func (w *FaceWorker) Start() {
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

	logger.Face("worker_started", "Face worker started", nil)
}

// Stop stops the face worker gracefully
func (w *FaceWorker) Stop() {
	w.mu.Lock()
	if !w.isRunning {
		w.mu.Unlock()
		return
	}
	w.isRunning = false
	w.mu.Unlock()

	w.cancel()
	w.wg.Wait()
	logger.Face("worker_stopped", "Face worker stopped", nil)
}

// IsRunning returns whether the worker is running
func (w *FaceWorker) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.isRunning
}

// run is the main worker loop
func (w *FaceWorker) run() {
	defer w.wg.Done()

	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Check if face API is available before starting
	if !w.faceClient.IsAvailable(w.ctx) {
		logger.Face("face_api_unavailable", "Face API is not available, worker will retry", nil)
	}

	// Process immediately on start
	w.processPendingPhotos()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.processPendingPhotos()
		}
	}
}

// processPendingPhotos fetches and processes photos with pending face status
func (w *FaceWorker) processPendingPhotos() {
	// Check circuit breaker
	if w.circuitBreaker.IsOpen() {
		logger.Face("circuit_breaker_open", "Circuit breaker open, skipping face processing", map[string]interface{}{
			"failures": w.circuitBreaker.GetFailures(),
		})
		return
	}

	// Check if face API is available
	if !w.faceClient.IsAvailable(w.ctx) {
		w.circuitBreaker.RecordFailure()
		logger.Face("face_api_unavailable_trigger", "Face API not available, circuit breaker triggered", nil)
		return
	}

	// Get photos with pending face status
	photos, err := w.photoRepo.GetByFaceStatus(w.ctx, models.FaceStatusPending, w.batchSize)
	if err != nil {
		logger.FaceError("fetch_pending_photos_failed", "Error fetching pending photos", err, nil)
		return
	}

	if len(photos) == 0 {
		return
	}

	logger.Face("processing_photos", "Processing photos for face detection", map[string]interface{}{
		"photo_count": len(photos),
		"concurrency": w.maxConcurrent,
	})

	// Process each photo
	var photoWg sync.WaitGroup
	sem := make(chan struct{}, w.maxConcurrent)

	successCount := int32(0)
	failCount := int32(0)

	for _, photo := range photos {
		sem <- struct{}{} // Acquire semaphore
		photoWg.Add(1)

		go func(p models.Photo) {
			defer photoWg.Done()
			defer func() { <-sem }() // Release semaphore

			if w.processPhotoWithRetry(p) {
				atomic.AddInt32(&successCount, 1)
				w.circuitBreaker.RecordSuccess()
			} else {
				atomic.AddInt32(&failCount, 1)
			}
		}(photo)
	}

	photoWg.Wait()

	logger.Face("batch_complete", "Photo batch processing complete", map[string]interface{}{
		"success_count": successCount,
		"fail_count":    failCount,
	})
}

// processPhotoWithRetry processes a photo with retry logic
func (w *FaceWorker) processPhotoWithRetry(photo models.Photo) bool {
	var lastErr error

	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := w.baseRetryDelay * time.Duration(1<<uint(attempt-1))
			logger.Face("photo_retry", "Retrying photo processing", map[string]interface{}{
				"photo_id":    photo.ID.String(),
				"attempt":     attempt,
				"max_retries": w.maxRetries,
				"delay_ms":    delay.Milliseconds(),
			})
			time.Sleep(delay)
		}

		err := w.processPhoto(photo)
		if err == nil {
			return true
		}

		lastErr = err
		logger.FaceError("photo_processing_failed", "Photo processing failed", err, map[string]interface{}{
			"photo_id": photo.ID.String(),
			"attempt":  attempt + 1,
			"max":      w.maxRetries + 1,
		})

		// Check if error is retryable
		if !isRetryableError(err) {
			break
		}
	}

	// All retries exhausted, mark as failed
	w.failPhotoWithBroadcast(w.ctx, photo, lastErr.Error())
	w.circuitBreaker.RecordFailure()
	return false
}

// isRetryableError determines if an error is retryable
func isRetryableError(err error) bool {
	// Network errors, timeouts, and temporary failures are retryable
	errStr := err.Error()
	retryablePatterns := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"temporary failure",
		"503",
		"502",
		"504",
		"rate limit",
	}

	for _, pattern := range retryablePatterns {
		if contains(errStr, pattern) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// processPhoto processes a single photo for face detection
func (w *FaceWorker) processPhoto(photo models.Photo) error {
	ctx := w.ctx
	photoID := photo.ID

	// Check for existing faces (prevent duplicates on restart)
	existingFaces, _ := w.faceRepo.GetByPhoto(ctx, photoID)
	if len(existingFaces) > 0 {
		// Update status to completed
		w.photoRepo.UpdateFaceStatus(ctx, photoID, models.FaceStatusCompleted, len(existingFaces))
		return nil
	}

	// Update status to processing
	if err := w.photoRepo.UpdateFaceStatus(ctx, photoID, models.FaceStatusProcessing, 0); err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Get SharedFolder for Drive credentials
	folder, err := w.sharedFolderRepo.GetByID(ctx, photo.SharedFolderID)
	if err != nil {
		return fmt.Errorf("failed to get shared folder: %w", err)
	}

	// Get token owner for Drive access
	user, err := w.userRepo.GetByID(ctx, folder.TokenOwnerID)
	if err != nil {
		return fmt.Errorf("failed to get token owner: %w", err)
	}

	// Download image from Google Drive (with authentication)
	imageData, mimeType, err := w.downloadImageWithRetry(ctx, user, photo)
	if err != nil {
		return fmt.Errorf("failed to download image: %w", err)
	}

	// Call face API with image bytes (not URL)
	result, err := w.faceClient.ExtractFacesFromBytes(ctx, imageData, mimeType)
	if err != nil {
		return fmt.Errorf("face extraction failed: %w", err)
	}

	// Process detected faces
	if len(result.Faces) == 0 {
		// No faces detected - mark as completed
		w.photoRepo.UpdateFaceStatus(ctx, photoID, models.FaceStatusCompleted, 0)
		// Broadcast to all users with folder access
		w.broadcastToFolderUsers(ctx, photo.SharedFolderID, "photo:updated", map[string]interface{}{
			"photoId":    photoID.String(),
			"faceStatus": models.FaceStatusCompleted,
			"faceCount":  0,
		})
		return nil
	}

	// Save detected faces
	faces := make([]*models.Face, 0, len(result.Faces))
	for _, detectedFace := range result.Faces {
		// Convert float32 embedding to pgvector
		embedding := make([]float32, len(detectedFace.Embedding))
		copy(embedding, detectedFace.Embedding)

		face := &models.Face{
			ID:             uuid.New(),
			SharedFolderID: photo.SharedFolderID,
			PhotoID:        photoID,
			Embedding:      pgvector.NewVector(embedding),
			BboxX:          detectedFace.BboxX,
			BboxY:          detectedFace.BboxY,
			BboxWidth:      detectedFace.BboxWidth,
			BboxHeight:     detectedFace.BboxHeight,
			Confidence:     detectedFace.Confidence,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		faces = append(faces, face)
	}

	// Batch insert faces
	if err := w.faceRepo.CreateBatch(ctx, faces); err != nil {
		return fmt.Errorf("failed to save faces: %w", err)
	}

	// Update face count and status
	w.photoRepo.UpdateFaceStatus(ctx, photoID, models.FaceStatusCompleted, len(faces))

	// Broadcast to all users with folder access
	w.broadcastToFolderUsers(ctx, photo.SharedFolderID, "photo:updated", map[string]interface{}{
		"photoId":    photoID.String(),
		"faceStatus": models.FaceStatusCompleted,
		"faceCount":  len(faces),
	})

	logger.Face("photo_processed", "Photo processed successfully", map[string]interface{}{
		"photo_id":   photoID.String(),
		"face_count": len(faces),
	})
	return nil
}

// downloadImageWithRetry downloads image with retry logic
func (w *FaceWorker) downloadImageWithRetry(ctx context.Context, user *models.User, photo models.Photo) ([]byte, string, error) {
	var lastErr error

	for attempt := 0; attempt <= 2; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * time.Second)
		}

		data, mimeType, err := w.downloadImage(ctx, user, photo)
		if err == nil {
			return data, mimeType, nil
		}
		lastErr = err
	}

	return nil, "", lastErr
}

// downloadImage downloads the image from Google Drive with authentication
func (w *FaceWorker) downloadImage(ctx context.Context, user *models.User, photo models.Photo) ([]byte, string, error) {
	if user.DriveRefreshToken == "" {
		return nil, "", fmt.Errorf("user has no Drive credentials")
	}

	expiry := time.Now()
	if user.DriveTokenExpiry != nil {
		expiry = *user.DriveTokenExpiry
	}

	// Download high-resolution thumbnail (1024px) for face detection
	// This is faster than downloading the full image and works well for face detection
	imageData, mimeType, err := w.driveClient.DownloadThumbnail(
		ctx,
		user.DriveAccessToken,
		user.DriveRefreshToken,
		expiry,
		photo.DriveFileID,
		1024, // 1024px resolution for better face detection
	)
	if err != nil {
		// Fallback: try to download the full file

		srv, srvErr := w.driveClient.GetDriveService(ctx, user.DriveAccessToken, user.DriveRefreshToken, expiry)
		if srvErr != nil {
			return nil, "", fmt.Errorf("failed to get drive service: %w", srvErr)
		}

		reader, downloadErr := w.driveClient.DownloadFile(ctx, srv, photo.DriveFileID)
		if downloadErr != nil {
			return nil, "", fmt.Errorf("failed to download file: %w", downloadErr)
		}
		defer reader.Close()

		data, readErr := io.ReadAll(reader)
		if readErr != nil {
			return nil, "", fmt.Errorf("failed to read file: %w", readErr)
		}

		return data, photo.MimeType, nil
	}

	return imageData, mimeType, nil
}

// failPhotoWithBroadcast marks a photo as failed and broadcasts the update
func (w *FaceWorker) failPhotoWithBroadcast(ctx context.Context, photo models.Photo, errMsg string) {
	logger.FaceError("photo_face_failed", "Photo face processing failed", nil, map[string]interface{}{
		"photo_id": photo.ID.String(),
		"error":    errMsg,
	})

	w.photoRepo.UpdateFaceStatus(ctx, photo.ID, models.FaceStatusFailed, 0)

	// Broadcast to all users with folder access
	w.broadcastToFolderUsers(ctx, photo.SharedFolderID, "photo:updated", map[string]interface{}{
		"photoId":    photo.ID.String(),
		"faceStatus": models.FaceStatusFailed,
		"faceCount":  0,
		"error":      errMsg,
	})
}

// broadcastToFolderUsers sends a websocket message to all users with access to a folder
func (w *FaceWorker) broadcastToFolderUsers(ctx context.Context, folderID uuid.UUID, messageType string, data map[string]interface{}) {
	users, err := w.sharedFolderRepo.GetUsersByFolder(ctx, folderID)
	if err != nil {
		return
	}

	for _, user := range users {
		websocket.Manager.BroadcastToUser(user.ID, messageType, data)
	}
}

// GetStats returns worker statistics
func (w *FaceWorker) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"isRunning":        w.IsRunning(),
		"maxConcurrent":    w.maxConcurrent,
		"batchSize":        w.batchSize,
		"circuitBreaker":   !w.circuitBreaker.IsOpen(),
		"circuitFailures":  w.circuitBreaker.GetFailures(),
	}
}
