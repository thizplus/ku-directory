package handlers

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"gorm.io/gorm"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/infrastructure/faceapi"
	"gofiber-template/infrastructure/redis"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	db              *gorm.DB
	redisClient     *redis.RedisClient
	faceClient      *faceapi.FaceClient
	photoRepository repositories.PhotoRepository
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(
	db *gorm.DB,
	redisClient *redis.RedisClient,
	faceClient *faceapi.FaceClient,
	photoRepository repositories.PhotoRepository,
) *HealthHandler {
	return &HealthHandler{
		db:              db,
		redisClient:     redisClient,
		faceClient:      faceClient,
		photoRepository: photoRepository,
	}
}

// ComponentHealth represents health status of a component
type ComponentHealth struct {
	Status  string `json:"status"` // "ok", "error", "unavailable"
	Message string `json:"message,omitempty"`
	Latency string `json:"latency,omitempty"`
}

// DetailedHealthResponse represents detailed health check response
type DetailedHealthResponse struct {
	Status     string                     `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp  time.Time                  `json:"timestamp"`
	Components map[string]ComponentHealth `json:"components"`
	Metrics    *HealthMetrics             `json:"metrics,omitempty"`
}

// HealthMetrics contains various system metrics
type HealthMetrics struct {
	PendingPhotos    int64 `json:"pending_photos"`
	ProcessingPhotos int64 `json:"processing_photos"`
	StuckPhotos      int64 `json:"stuck_photos"`
	FailedPhotos     int64 `json:"failed_photos"`
	TotalPhotos      int64 `json:"total_photos"`
}

// DetailedHealth godoc
// @Summary Get detailed system health
// @Description Returns detailed health status of all system components
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} DetailedHealthResponse
// @Router /health/detailed [get]
func (h *HealthHandler) DetailedHealth(c *fiber.Ctx) error {
	ctx, cancel := context.WithTimeout(c.UserContext(), 10*time.Second)
	defer cancel()

	response := DetailedHealthResponse{
		Timestamp:  time.Now(),
		Components: make(map[string]ComponentHealth),
	}

	allHealthy := true
	hasCriticalFailure := false

	// Check Database
	dbHealth := h.checkDatabase(ctx)
	response.Components["database"] = dbHealth
	if dbHealth.Status != "ok" {
		hasCriticalFailure = true
	}

	// Check Redis
	redisHealth := h.checkRedis(ctx)
	response.Components["redis"] = redisHealth
	if redisHealth.Status == "error" {
		allHealthy = false
	}

	// Check Face API
	faceHealth := h.checkFaceAPI(ctx)
	response.Components["face_api"] = faceHealth
	if faceHealth.Status == "error" {
		allHealthy = false
	}

	// Get metrics (only if DB is ok)
	if dbHealth.Status == "ok" {
		metrics := h.getMetrics(ctx)
		response.Metrics = metrics

		// Check if there are stuck photos (processing > 5 minutes)
		if metrics.StuckPhotos > 0 {
			allHealthy = false
		}
	}

	// Determine overall status
	if hasCriticalFailure {
		response.Status = "unhealthy"
	} else if !allHealthy {
		response.Status = "degraded"
	} else {
		response.Status = "healthy"
	}

	// Return 503 for unhealthy, 200 for others
	statusCode := fiber.StatusOK
	if response.Status == "unhealthy" {
		statusCode = fiber.StatusServiceUnavailable
	}

	return c.Status(statusCode).JSON(response)
}

func (h *HealthHandler) checkDatabase(ctx context.Context) ComponentHealth {
	start := time.Now()

	if h.db == nil {
		return ComponentHealth{
			Status:  "error",
			Message: "Database not configured",
		}
	}

	sqlDB, err := h.db.DB()
	if err != nil {
		return ComponentHealth{
			Status:  "error",
			Message: "Failed to get database connection: " + err.Error(),
		}
	}

	if err := sqlDB.PingContext(ctx); err != nil {
		return ComponentHealth{
			Status:  "error",
			Message: "Database ping failed: " + err.Error(),
		}
	}

	return ComponentHealth{
		Status:  "ok",
		Message: "Connected",
		Latency: time.Since(start).String(),
	}
}

func (h *HealthHandler) checkRedis(ctx context.Context) ComponentHealth {
	start := time.Now()

	if h.redisClient == nil {
		return ComponentHealth{
			Status:  "unavailable",
			Message: "Redis not configured",
		}
	}

	if err := h.redisClient.Ping(ctx); err != nil {
		return ComponentHealth{
			Status:  "error",
			Message: "Redis ping failed: " + err.Error(),
		}
	}

	return ComponentHealth{
		Status:  "ok",
		Message: "Connected",
		Latency: time.Since(start).String(),
	}
}

func (h *HealthHandler) checkFaceAPI(ctx context.Context) ComponentHealth {
	start := time.Now()

	if h.faceClient == nil {
		return ComponentHealth{
			Status:  "unavailable",
			Message: "Face API not configured",
		}
	}

	health, err := h.faceClient.Health(ctx)
	if err != nil {
		return ComponentHealth{
			Status:  "error",
			Message: "Face API health check failed: " + err.Error(),
		}
	}

	return ComponentHealth{
		Status:  "ok",
		Message: "Model: " + health.Model + ", Version: " + health.Version,
		Latency: time.Since(start).String(),
	}
}

func (h *HealthHandler) getMetrics(ctx context.Context) *HealthMetrics {
	if h.photoRepository == nil {
		return nil
	}

	metrics := &HealthMetrics{}

	// Count pending photos
	pendingPhotos, err := h.photoRepository.GetByFaceStatus(ctx, models.FaceStatusPending, 0)
	if err == nil {
		metrics.PendingPhotos = int64(len(pendingPhotos))
	}

	// Count processing photos
	processingPhotos, err := h.photoRepository.GetByFaceStatus(ctx, models.FaceStatusProcessing, 0)
	if err == nil {
		metrics.ProcessingPhotos = int64(len(processingPhotos))
	}

	// Count failed photos
	failedPhotos, err := h.photoRepository.GetByFaceStatus(ctx, models.FaceStatusFailed, 0)
	if err == nil {
		metrics.FailedPhotos = int64(len(failedPhotos))
	}

	// Count stuck photos (processing for > 5 minutes)
	if metrics.ProcessingPhotos > 0 {
		// We don't have a direct query for this, so we estimate by checking updated_at
		// Stuck photos are those in processing status that haven't been updated in 5 minutes
		stuckCount := int64(0)
		threshold := time.Now().Add(-5 * time.Minute)
		for _, photo := range processingPhotos {
			if photo.UpdatedAt.Before(threshold) {
				stuckCount++
			}
		}
		metrics.StuckPhotos = stuckCount
	}

	return metrics
}
