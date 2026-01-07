package services

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"gofiber-template/domain/models"
)

// Custom errors for face service
var (
	ErrNoFacesDetected   = errors.New("no faces detected in the uploaded image")
	ErrFaceNotFound      = errors.New("face not found")
	ErrInvalidFaceIndex  = errors.New("invalid face index")
)

// FaceSearchResult represents a face search result
type FaceSearchResult struct {
	Face       models.Face
	Photo      models.Photo
	Similarity float64
}

// DetectedFace represents a face detected in an uploaded image
type DetectedFace struct {
	Index      int       `json:"index"`
	BboxX      float64   `json:"bbox_x"`
	BboxY      float64   `json:"bbox_y"`
	BboxWidth  float64   `json:"bbox_width"`
	BboxHeight float64   `json:"bbox_height"`
	Confidence float64   `json:"confidence"`
	Embedding  []float32 `json:"-"` // Hidden from JSON, used internally
}

// FaceService handles face-related operations
type FaceService interface {
	// Detect faces in an uploaded image (returns all faces with bounding boxes)
	DetectFaces(ctx context.Context, imageData []byte, mimeType string) ([]DetectedFace, error)

	// Search by uploading a photo with face index selection
	SearchByImageWithIndex(ctx context.Context, userID uuid.UUID, imageData []byte, mimeType string, faceIndex int, limit int, threshold float64) ([]FaceSearchResult, error)

	// Search by uploading a photo (uses first face - legacy)
	SearchByImage(ctx context.Context, userID uuid.UUID, imageData []byte, mimeType string, limit int, threshold float64) ([]FaceSearchResult, error)

	// Search by existing face ID
	SearchByFaceID(ctx context.Context, userID uuid.UUID, faceID uuid.UUID, limit int, threshold float64) ([]FaceSearchResult, error)

	// Get faces for a photo
	GetFacesByPhoto(ctx context.Context, userID uuid.UUID, photoID uuid.UUID) ([]models.Face, error)

	// Get faces for a person
	GetFacesByPerson(ctx context.Context, userID uuid.UUID, personID uuid.UUID) ([]models.Face, error)

	// Get all faces with pagination
	GetFaces(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Face, int64, error)

	// Assign face to a person
	AssignFaceToPerson(ctx context.Context, userID uuid.UUID, faceID uuid.UUID, personID uuid.UUID) error

	// Remove face from person
	RemoveFaceFromPerson(ctx context.Context, userID uuid.UUID, faceID uuid.UUID) error

	// Get face count
	GetFaceCount(ctx context.Context, userID uuid.UUID) (int64, error)

	// Get processing stats
	GetProcessingStats(ctx context.Context, userID uuid.UUID) (*FaceProcessingStats, error)

	// Retry failed photos
	RetryFailedPhotos(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID) (int64, error)
}

// FaceProcessingStats contains face processing statistics
type FaceProcessingStats struct {
	TotalPhotos     int64 `json:"total_photos"`
	ProcessedPhotos int64 `json:"processed_photos"`
	PendingPhotos   int64 `json:"pending_photos"`
	FailedPhotos    int64 `json:"failed_photos"`
	TotalFaces      int64 `json:"total_faces"`
}
