package worker

import (
	"context"
	"fmt"
	"io"
	"log"
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

	log.Println("✓ Face worker started")
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
	log.Println("✓ Face worker stopped")
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
		log.Println("Warning: Face API is not available, worker will retry...")
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
		log.Printf("Circuit breaker open (failures: %d), skipping face processing", w.circuitBreaker.GetFailures())
		return
	}

	// Check if face API is available
	if !w.faceClient.IsAvailable(w.ctx) {
		w.circuitBreaker.RecordFailure()
		log.Println("Face API not available, circuit breaker triggered")
		return
	}

	// Get photos with pending face status
	photos, err := w.photoRepo.GetByFaceStatus(w.ctx, models.FaceStatusPending, w.batchSize)
	if err != nil {
		log.Printf("Error fetching pending photos: %v", err)
		return
	}

	if len(photos) == 0 {
		return
	}

	log.Printf("Processing %d photos for face detection (concurrency: %d)", len(photos), w.maxConcurrent)

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

	log.Printf("Batch complete: %d success, %d failed", successCount, failCount)
}

// processPhotoWithRetry processes a photo with retry logic
func (w *FaceWorker) processPhotoWithRetry(photo models.Photo) bool {
	var lastErr error

	for attempt := 0; attempt <= w.maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			delay := w.baseRetryDelay * time.Duration(1<<uint(attempt-1))
			log.Printf("Retry %d/%d for photo %s after %v", attempt, w.maxRetries, photo.ID, delay)
			time.Sleep(delay)
		}

		err := w.processPhoto(photo)
		if err == nil {
			return true
		}

		lastErr = err
		log.Printf("Photo %s processing failed (attempt %d/%d): %v", photo.ID, attempt+1, w.maxRetries+1, err)

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
		log.Printf("Photo %s already has %d faces, skipping", photoID, len(existingFaces))
		// Update status to completed
		w.photoRepo.UpdateFaceStatus(ctx, photoID, models.FaceStatusCompleted, len(existingFaces))
		return nil
	}

	log.Printf("Processing photo %s: %s", photoID, photo.FileName)

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
		if err := w.photoRepo.UpdateFaceStatus(ctx, photoID, models.FaceStatusCompleted, 0); err != nil {
			log.Printf("Error updating photo status: %v", err)
		}
		// Broadcast to all users with folder access
		w.broadcastToFolderUsers(ctx, photo.SharedFolderID, "photo:updated", map[string]interface{}{
			"photoId":    photoID.String(),
			"faceStatus": models.FaceStatusCompleted,
			"faceCount":  0,
		})
		log.Printf("No faces found in photo %s", photoID)
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
	if err := w.photoRepo.UpdateFaceStatus(ctx, photoID, models.FaceStatusCompleted, len(faces)); err != nil {
		log.Printf("Error updating photo status: %v", err)
	}

	// Broadcast to all users with folder access
	w.broadcastToFolderUsers(ctx, photo.SharedFolderID, "photo:updated", map[string]interface{}{
		"photoId":    photoID.String(),
		"faceStatus": models.FaceStatusCompleted,
		"faceCount":  len(faces),
	})

	log.Printf("Photo %s processed: %d faces detected", photoID, len(faces))
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
		log.Printf("Thumbnail download failed, trying full file: %v", err)

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
	log.Printf("Photo %s face processing failed: %s", photo.ID, errMsg)

	if err := w.photoRepo.UpdateFaceStatus(ctx, photo.ID, models.FaceStatusFailed, 0); err != nil {
		log.Printf("Error updating photo status: %v", err)
	}

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
		log.Printf("Error getting folder users for broadcast: %v", err)
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
