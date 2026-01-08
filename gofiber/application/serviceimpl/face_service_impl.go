package serviceimpl

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"

	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"gofiber-template/infrastructure/faceapi"
	"gofiber-template/pkg/logger"
)

type FaceServiceImpl struct {
	faceRepo         repositories.FaceRepository
	photoRepo        repositories.PhotoRepository
	personRepo       repositories.PersonRepository
	userRepo         repositories.UserRepository
	sharedFolderRepo repositories.SharedFolderRepository
	faceClient       *faceapi.FaceClient
}

func NewFaceService(
	faceRepo repositories.FaceRepository,
	photoRepo repositories.PhotoRepository,
	personRepo repositories.PersonRepository,
	userRepo repositories.UserRepository,
	sharedFolderRepo repositories.SharedFolderRepository,
	faceClient *faceapi.FaceClient,
) services.FaceService {
	return &FaceServiceImpl{
		faceRepo:         faceRepo,
		photoRepo:        photoRepo,
		personRepo:       personRepo,
		userRepo:         userRepo,
		sharedFolderRepo: sharedFolderRepo,
		faceClient:       faceClient,
	}
}

// DetectFaces detects all faces in an uploaded image and returns their bounding boxes
func (s *FaceServiceImpl) DetectFaces(ctx context.Context, imageData []byte, mimeType string) ([]services.DetectedFace, error) {
	// Call face API to extract faces from uploaded image
	result, err := s.faceClient.ExtractFacesFromBytes(ctx, imageData, mimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to extract faces from image: %w", err)
	}

	if len(result.Faces) == 0 {
		return nil, services.ErrNoFacesDetected
	}

	// Convert to detected faces
	faces := make([]services.DetectedFace, len(result.Faces))
	for i, f := range result.Faces {
		faces[i] = services.DetectedFace{
			Index:      i,
			BboxX:      f.BboxX,
			BboxY:      f.BboxY,
			BboxWidth:  f.BboxWidth,
			BboxHeight: f.BboxHeight,
			Confidence: f.Confidence,
			Embedding:  f.Embedding,
		}
	}

	return faces, nil
}

// SearchByImageWithIndex searches for similar faces using a specific face from the uploaded image
func (s *FaceServiceImpl) SearchByImageWithIndex(ctx context.Context, userID uuid.UUID, imageData []byte, mimeType string, faceIndex int, limit int, threshold float64) ([]services.FaceSearchResult, error) {
	// Call face API to extract embedding from uploaded image
	result, err := s.faceClient.ExtractFacesFromBytes(ctx, imageData, mimeType)
	if err != nil {
		return nil, fmt.Errorf("failed to extract faces from image: %w", err)
	}

	if len(result.Faces) == 0 {
		return nil, services.ErrNoFacesDetected
	}

	// Validate face index
	if faceIndex < 0 || faceIndex >= len(result.Faces) {
		return nil, services.ErrInvalidFaceIndex
	}

	// Use the selected face for search
	queryFace := result.Faces[faceIndex]

	// Convert embedding to pgvector
	embedding := pgvector.NewVector(queryFace.Embedding)

	// Get user's accessible shared folders
	folders, err := s.sharedFolderRepo.GetFoldersByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user folders: %w", err)
	}

	if len(folders) == 0 {
		return []services.FaceSearchResult{}, nil
	}

	// Collect folder IDs
	folderIDs := make([]uuid.UUID, len(folders))
	for i, f := range folders {
		folderIDs[i] = f.ID
	}

	// Search for similar faces in user's shared folders
	searchResults, err := s.faceRepo.SearchSimilarBySharedFolders(ctx, folderIDs, embedding, limit, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar faces: %w", err)
	}

	// Convert to service result
	results := make([]services.FaceSearchResult, len(searchResults))
	for i, r := range searchResults {
		results[i] = services.FaceSearchResult{
			Face:       r.Face,
			Photo:      r.Photo,
			Similarity: r.Similarity,
		}
	}

	return results, nil
}

// SearchByImage searches for similar faces by uploading an image (uses first face - legacy)
func (s *FaceServiceImpl) SearchByImage(ctx context.Context, userID uuid.UUID, imageData []byte, mimeType string, limit int, threshold float64) ([]services.FaceSearchResult, error) {
	return s.SearchByImageWithIndex(ctx, userID, imageData, mimeType, 0, limit, threshold)
}

// SearchByFaceID searches for similar faces using an existing face's embedding
func (s *FaceServiceImpl) SearchByFaceID(ctx context.Context, userID uuid.UUID, faceID uuid.UUID, limit int, threshold float64) ([]services.FaceSearchResult, error) {
	// Get the source face
	sourceFace, err := s.faceRepo.GetByID(ctx, faceID)
	if err != nil {
		return nil, fmt.Errorf("face not found: %w", err)
	}

	// Verify user has access to the face's folder
	hasAccess, err := s.sharedFolderRepo.HasUserAccess(ctx, userID, sourceFace.SharedFolderID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify access: %w", err)
	}
	if !hasAccess {
		return nil, fmt.Errorf("face not found")
	}

	// Get user's accessible shared folders
	folders, err := s.sharedFolderRepo.GetFoldersByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user folders: %w", err)
	}

	if len(folders) == 0 {
		return []services.FaceSearchResult{}, nil
	}

	// Collect folder IDs
	folderIDs := make([]uuid.UUID, len(folders))
	for i, f := range folders {
		folderIDs[i] = f.ID
	}

	// Search for similar faces using the source face's embedding
	searchResults, err := s.faceRepo.SearchSimilarBySharedFolders(ctx, folderIDs, sourceFace.Embedding, limit+1, threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to search similar faces: %w", err)
	}

	// Convert to service result, excluding the source face itself
	results := make([]services.FaceSearchResult, 0, len(searchResults))
	for _, r := range searchResults {
		if r.Face.ID == faceID {
			continue // Skip the source face
		}
		results = append(results, services.FaceSearchResult{
			Face:       r.Face,
			Photo:      r.Photo,
			Similarity: r.Similarity,
		})
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetFacesByPhoto returns all faces detected in a photo
func (s *FaceServiceImpl) GetFacesByPhoto(ctx context.Context, userID uuid.UUID, photoID uuid.UUID) ([]models.Face, error) {
	// Get photo
	photo, err := s.photoRepo.GetByID(ctx, photoID)
	if err != nil {
		return nil, fmt.Errorf("photo not found: %w", err)
	}

	// Verify user has access to the photo's folder
	hasAccess, err := s.sharedFolderRepo.HasUserAccess(ctx, userID, photo.SharedFolderID)
	if err != nil {
		return nil, fmt.Errorf("failed to verify access: %w", err)
	}
	if !hasAccess {
		return nil, fmt.Errorf("photo not found")
	}

	return s.faceRepo.GetByPhoto(ctx, photoID)
}

// GetFacesByPerson returns all faces assigned to a person
func (s *FaceServiceImpl) GetFacesByPerson(ctx context.Context, userID uuid.UUID, personID uuid.UUID) ([]models.Face, error) {
	// Verify person ownership
	person, err := s.personRepo.GetByID(ctx, personID)
	if err != nil {
		return nil, fmt.Errorf("person not found: %w", err)
	}

	if person.UserID != userID {
		return nil, fmt.Errorf("person not found")
	}

	return s.faceRepo.GetByPerson(ctx, personID)
}

// GetFaces returns paginated faces for a user
func (s *FaceServiceImpl) GetFaces(ctx context.Context, userID uuid.UUID, page, limit int) ([]models.Face, int64, error) {
	// Get user's accessible shared folders
	folders, err := s.sharedFolderRepo.GetFoldersByUser(ctx, userID)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get user folders: %w", err)
	}

	if len(folders) == 0 {
		return []models.Face{}, 0, nil
	}

	// Collect folder IDs
	folderIDs := make([]uuid.UUID, len(folders))
	for i, f := range folders {
		folderIDs[i] = f.ID
	}

	offset := (page - 1) * limit
	return s.faceRepo.GetBySharedFolders(ctx, folderIDs, offset, limit)
}

// AssignFaceToPerson assigns a face to a person
func (s *FaceServiceImpl) AssignFaceToPerson(ctx context.Context, userID uuid.UUID, faceID uuid.UUID, personID uuid.UUID) error {
	// Get face
	face, err := s.faceRepo.GetByID(ctx, faceID)
	if err != nil {
		return fmt.Errorf("face not found: %w", err)
	}

	// Verify user has access to the face's folder
	hasAccess, err := s.sharedFolderRepo.HasUserAccess(ctx, userID, face.SharedFolderID)
	if err != nil {
		return fmt.Errorf("failed to verify access: %w", err)
	}
	if !hasAccess {
		return fmt.Errorf("face not found")
	}

	// Verify person ownership (persons are still user-owned)
	person, err := s.personRepo.GetByID(ctx, personID)
	if err != nil {
		return fmt.Errorf("person not found: %w", err)
	}

	if person.UserID != userID {
		return fmt.Errorf("person not found")
	}

	// Assign face to person
	return s.faceRepo.UpdatePersonID(ctx, faceID, &personID)
}

// RemoveFaceFromPerson removes a face from its assigned person
func (s *FaceServiceImpl) RemoveFaceFromPerson(ctx context.Context, userID uuid.UUID, faceID uuid.UUID) error {
	// Get face
	face, err := s.faceRepo.GetByID(ctx, faceID)
	if err != nil {
		return fmt.Errorf("face not found: %w", err)
	}

	// Verify user has access to the face's folder
	hasAccess, err := s.sharedFolderRepo.HasUserAccess(ctx, userID, face.SharedFolderID)
	if err != nil {
		return fmt.Errorf("failed to verify access: %w", err)
	}
	if !hasAccess {
		return fmt.Errorf("face not found")
	}

	// Remove person assignment
	return s.faceRepo.UpdatePersonID(ctx, faceID, nil)
}

// GetFaceCount returns the total number of faces for a user
func (s *FaceServiceImpl) GetFaceCount(ctx context.Context, userID uuid.UUID) (int64, error) {
	// Get user's accessible shared folders
	folders, err := s.sharedFolderRepo.GetFoldersByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user folders: %w", err)
	}

	if len(folders) == 0 {
		return 0, nil
	}

	// Collect folder IDs
	folderIDs := make([]uuid.UUID, len(folders))
	for i, f := range folders {
		folderIDs[i] = f.ID
	}

	return s.faceRepo.CountBySharedFolders(ctx, folderIDs)
}

// GetProcessingStats returns face processing statistics
func (s *FaceServiceImpl) GetProcessingStats(ctx context.Context, userID uuid.UUID) (*services.FaceProcessingStats, error) {
	logger.Face("get_processing_stats", "GetProcessingStats called", map[string]interface{}{
		"user_id": userID.String(),
	})

	// Get user's accessible shared folders
	folders, err := s.sharedFolderRepo.GetFoldersByUser(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user folders: %w", err)
	}

	logger.Face("folders_found", "Found folders for user", map[string]interface{}{
		"user_id":      userID.String(),
		"folder_count": len(folders),
	})

	// If user has no folders, return empty stats
	if len(folders) == 0 {
		logger.Face("no_folders", "User has no folders, returning empty stats", map[string]interface{}{
			"user_id": userID.String(),
		})
		return &services.FaceProcessingStats{
			TotalPhotos:     0,
			ProcessedPhotos: 0,
			PendingPhotos:   0,
			FailedPhotos:    0,
			TotalFaces:      0,
		}, nil
	}

	// Collect folder IDs
	folderIDs := make([]uuid.UUID, len(folders))
	for i, f := range folders {
		folderIDs[i] = f.ID
	}

	// Aggregate stats across all user's folders
	var totalPhotos, pendingPhotos, processingPhotos, completedPhotos, failedPhotos int64

	for _, folderID := range folderIDs {
		count, err := s.photoRepo.CountBySharedFolder(ctx, folderID)
		if err != nil {
			return nil, fmt.Errorf("failed to get photo count: %w", err)
		}
		totalPhotos += count

		pending, err := s.photoRepo.CountBySharedFolderAndFaceStatus(ctx, folderID, models.FaceStatusPending)
		if err != nil {
			return nil, fmt.Errorf("failed to get pending photos count: %w", err)
		}
		pendingPhotos += pending

		processing, err := s.photoRepo.CountBySharedFolderAndFaceStatus(ctx, folderID, models.FaceStatusProcessing)
		if err != nil {
			return nil, fmt.Errorf("failed to get processing photos count: %w", err)
		}
		processingPhotos += processing

		completed, err := s.photoRepo.CountBySharedFolderAndFaceStatus(ctx, folderID, models.FaceStatusCompleted)
		if err != nil {
			return nil, fmt.Errorf("failed to get completed photos count: %w", err)
		}
		completedPhotos += completed

		failed, err := s.photoRepo.CountBySharedFolderAndFaceStatus(ctx, folderID, models.FaceStatusFailed)
		if err != nil {
			return nil, fmt.Errorf("failed to get failed photos count: %w", err)
		}
		failedPhotos += failed
	}

	// Get total faces across all folders
	totalFaces, err := s.faceRepo.CountBySharedFolders(ctx, folderIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to get face count: %w", err)
	}

	return &services.FaceProcessingStats{
		TotalPhotos:     totalPhotos,
		ProcessedPhotos: completedPhotos,
		PendingPhotos:   pendingPhotos + processingPhotos,
		FailedPhotos:    failedPhotos,
		TotalFaces:      totalFaces,
	}, nil
}

// RetryFailedPhotos resets failed photos to pending for reprocessing
func (s *FaceServiceImpl) RetryFailedPhotos(ctx context.Context, userID uuid.UUID, folderID *uuid.UUID) (int64, error) {
	logger.Face("retry_failed_start", "RetryFailedPhotos called", map[string]interface{}{
		"user_id":   userID.String(),
		"folder_id": folderID,
	})

	// Get user's accessible folders to validate
	folders, err := s.sharedFolderRepo.GetFoldersByUser(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("failed to get user folders: %w", err)
	}

	if len(folders) == 0 {
		return 0, nil
	}

	// If specific folder requested, verify user has access
	if folderID != nil {
		hasAccess := false
		for _, f := range folders {
			if f.ID == *folderID {
				hasAccess = true
				break
			}
		}
		if !hasAccess {
			return 0, fmt.Errorf("access denied to folder")
		}
	}

	count, err := s.photoRepo.ResetFailedToPending(ctx, folderID)
	if err != nil {
		return 0, fmt.Errorf("failed to reset failed photos: %w", err)
	}

	logger.Face("retry_failed_complete", "Reset failed photos to pending", map[string]interface{}{
		"user_id":     userID.String(),
		"folder_id":   folderID,
		"reset_count": count,
	})
	return count, nil
}
