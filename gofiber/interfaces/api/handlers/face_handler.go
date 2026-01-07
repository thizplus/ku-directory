package handlers

import (
	"errors"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"gofiber-template/domain/services"
	"gofiber-template/pkg/utils"
)

type FaceHandler struct {
	faceService services.FaceService
}

func NewFaceHandler(faceService services.FaceService) *FaceHandler {
	return &FaceHandler{
		faceService: faceService,
	}
}

// SearchByImageRequest is the request for searching by image upload
type SearchByImageRequest struct {
	Limit     int     `json:"limit" form:"limit"`
	Threshold float64 `json:"threshold" form:"threshold"`
}

// SearchByFaceIDRequest is the request for searching by existing face
type SearchByFaceIDRequest struct {
	FaceID    string  `json:"face_id" validate:"required,uuid"`
	Limit     int     `json:"limit"`
	Threshold float64 `json:"threshold"`
}

// FaceSearchResultResponse is the response for face search (updated)
type FaceSearchResultResponse struct {
	FaceID         string  `json:"face_id"`
	PhotoID        string  `json:"photo_id"`
	SharedFolderID string  `json:"shared_folder_id"`
	DriveFileID    string  `json:"drive_file_id"`
	DriveFolderID  string  `json:"drive_folder_id"`
	FileName       string  `json:"file_name"`
	ThumbnailURL   string  `json:"thumbnail_url"`
	WebViewURL     string  `json:"web_view_url"`
	FolderPath     string  `json:"folder_path"`
	BboxX          float64 `json:"bbox_x"`
	BboxY          float64 `json:"bbox_y"`
	BboxWidth      float64 `json:"bbox_width"`
	BboxHeight     float64 `json:"bbox_height"`
	Similarity     float64 `json:"similarity"`
}

// DetectedFaceResponse is the response for detected faces
type DetectedFaceResponse struct {
	Index      int     `json:"index"`
	BboxX      float64 `json:"bbox_x"`
	BboxY      float64 `json:"bbox_y"`
	BboxWidth  float64 `json:"bbox_width"`
	BboxHeight float64 `json:"bbox_height"`
	Confidence float64 `json:"confidence"`
}

// DetectFaces detects all faces in an uploaded image
// @Summary Detect faces in an image
// @Tags Faces
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Image file"
// @Success 200 {object} utils.Response
// @Router /api/v1/faces/detect [post]
func (h *FaceHandler) DetectFaces(c *fiber.Ctx) error {
	// Get the uploaded file
	file, err := c.FormFile("image")
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Image file is required", err)
	}

	// Validate file size (max 10MB)
	if file.Size > 10*1024*1024 {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "File size exceeds 10MB limit", nil)
	}

	// Validate content type
	contentType := file.Header.Get("Content-Type")
	if !isValidImageType(contentType) {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid image type. Allowed: jpeg, png, webp, gif", nil)
	}

	// Open the file
	f, err := file.Open()
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to read file", err)
	}
	defer f.Close()

	// Read file content
	imageData := make([]byte, file.Size)
	_, err = f.Read(imageData)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to read file", err)
	}

	// Detect faces
	faces, err := h.faceService.DetectFaces(c.Context(), imageData, contentType)
	if err != nil {
		if errors.Is(err, services.ErrNoFacesDetected) {
			return utils.ErrorResponse(c, fiber.StatusBadRequest, "ไม่พบใบหน้าในรูปภาพที่อัปโหลด กรุณาใช้รูปที่เห็นใบหน้าชัดเจน", err)
		}
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Face detection failed", err)
	}

	// Convert to response format
	response := make([]DetectedFaceResponse, len(faces))
	for i, f := range faces {
		response[i] = DetectedFaceResponse{
			Index:      f.Index,
			BboxX:      f.BboxX,
			BboxY:      f.BboxY,
			BboxWidth:  f.BboxWidth,
			BboxHeight: f.BboxHeight,
			Confidence: f.Confidence,
		}
	}

	return utils.SuccessResponse(c, "Faces detected", fiber.Map{
		"faces": response,
		"count": len(response),
	})
}

// SearchByImage handles face search by uploading an image
// @Summary Search faces by uploading an image
// @Tags Faces
// @Accept multipart/form-data
// @Produce json
// @Param image formData file true "Image file"
// @Param face_index query int false "Face index to search (default: 0)" default(0)
// @Param limit query int false "Max results" default(20)
// @Param threshold query number false "Similarity threshold (0-1)" default(0.6)
// @Success 200 {object} utils.Response
// @Router /api/v1/faces/search/image [post]
func (h *FaceHandler) SearchByImage(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	// Get the uploaded file
	file, err := c.FormFile("image")
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Image file is required", err)
	}

	// Validate file size (max 10MB)
	if file.Size > 10*1024*1024 {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "File size exceeds 10MB limit", nil)
	}

	// Validate content type
	contentType := file.Header.Get("Content-Type")
	if !isValidImageType(contentType) {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid image type. Allowed: jpeg, png, webp, gif", nil)
	}

	// Open the file
	f, err := file.Open()
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to read file", err)
	}
	defer f.Close()

	// Read file content
	imageData := make([]byte, file.Size)
	_, err = f.Read(imageData)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to read file", err)
	}

	// Get query parameters
	faceIndex := c.QueryInt("face_index", 0)
	limit := c.QueryInt("limit", 20)
	threshold := c.QueryFloat("threshold", 0.6)

	// Validate parameters
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if threshold < 0 || threshold > 1 {
		threshold = 0.6
	}

	// Search for similar faces with selected face index
	results, err := h.faceService.SearchByImageWithIndex(c.Context(), userCtx.ID, imageData, contentType, faceIndex, limit, threshold)
	if err != nil {
		// Check for user-caused errors (should be 400, not 500)
		if errors.Is(err, services.ErrNoFacesDetected) {
			return utils.ErrorResponse(c, fiber.StatusBadRequest, "ไม่พบใบหน้าในรูปภาพที่อัปโหลด กรุณาใช้รูปที่เห็นใบหน้าชัดเจน", err)
		}
		if errors.Is(err, services.ErrInvalidFaceIndex) {
			return utils.ErrorResponse(c, fiber.StatusBadRequest, "ตำแหน่งใบหน้าไม่ถูกต้อง", err)
		}
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Face search failed", err)
	}

	// Convert to response format
	response := make([]FaceSearchResultResponse, len(results))
	for i, r := range results {
		response[i] = FaceSearchResultResponse{
			FaceID:         r.Face.ID.String(),
			PhotoID:        r.Photo.ID.String(),
			SharedFolderID: r.Photo.SharedFolderID.String(),
			DriveFileID:    r.Photo.DriveFileID,
			DriveFolderID:  r.Photo.DriveFolderID,
			FileName:       r.Photo.FileName,
			ThumbnailURL:   r.Photo.ThumbnailURL,
			WebViewURL:     r.Photo.WebViewURL,
			FolderPath:     r.Photo.DriveFolderPath,
			BboxX:          r.Face.BboxX,
			BboxY:          r.Face.BboxY,
			BboxWidth:      r.Face.BboxWidth,
			BboxHeight:     r.Face.BboxHeight,
			Similarity:     r.Similarity,
		}
	}

	return utils.SuccessResponse(c, "Face search completed", fiber.Map{
		"results":   response,
		"count":     len(response),
		"limit":     limit,
		"threshold": threshold,
	})
}

// SearchByFaceID handles face search using an existing face's embedding
// @Summary Search similar faces using an existing face
// @Tags Faces
// @Accept json
// @Produce json
// @Param request body SearchByFaceIDRequest true "Search request"
// @Success 200 {object} utils.Response
// @Router /api/v1/faces/search/face [post]
func (h *FaceHandler) SearchByFaceID(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	var req SearchByFaceIDRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request body", err)
	}

	// Parse face ID
	faceID, err := uuid.Parse(req.FaceID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid face ID", err)
	}

	// Validate parameters
	limit := req.Limit
	if limit < 1 || limit > 100 {
		limit = 20
	}

	threshold := req.Threshold
	if threshold < 0 || threshold > 1 {
		threshold = 0.6
	}

	// Search for similar faces
	results, err := h.faceService.SearchByFaceID(c.Context(), userCtx.ID, faceID, limit, threshold)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Face search failed", err)
	}

	// Convert to response format
	response := make([]FaceSearchResultResponse, len(results))
	for i, r := range results {
		response[i] = FaceSearchResultResponse{
			FaceID:         r.Face.ID.String(),
			PhotoID:        r.Photo.ID.String(),
			SharedFolderID: r.Photo.SharedFolderID.String(),
			DriveFileID:    r.Photo.DriveFileID,
			DriveFolderID:  r.Photo.DriveFolderID,
			FileName:       r.Photo.FileName,
			ThumbnailURL:   r.Photo.ThumbnailURL,
			WebViewURL:     r.Photo.WebViewURL,
			FolderPath:     r.Photo.DriveFolderPath,
			BboxX:          r.Face.BboxX,
			BboxY:          r.Face.BboxY,
			BboxWidth:      r.Face.BboxWidth,
			BboxHeight:     r.Face.BboxHeight,
			Similarity:     r.Similarity,
		}
	}

	return utils.SuccessResponse(c, "Face search completed", fiber.Map{
		"results":   response,
		"count":     len(response),
		"limit":     limit,
		"threshold": threshold,
	})
}

// GetFacesByPhoto returns all faces detected in a photo
// @Summary Get faces by photo ID
// @Tags Faces
// @Produce json
// @Param photo_id path string true "Photo ID"
// @Success 200 {object} utils.Response
// @Router /api/v1/faces/photo/{photo_id} [get]
func (h *FaceHandler) GetFacesByPhoto(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	photoID, err := uuid.Parse(c.Params("photo_id"))
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid photo ID", err)
	}

	faces, err := h.faceService.GetFacesByPhoto(c.Context(), userCtx.ID, photoID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to get faces", err)
	}

	// Convert to response format
	response := make([]fiber.Map, len(faces))
	for i, f := range faces {
		response[i] = fiber.Map{
			"id":          f.ID.String(),
			"photo_id":    f.PhotoID.String(),
			"bbox_x":      f.BboxX,
			"bbox_y":      f.BboxY,
			"bbox_width":  f.BboxWidth,
			"bbox_height": f.BboxHeight,
			"confidence":  f.Confidence,
			"person_id":   nil,
		}
		if f.PersonID != nil {
			response[i]["person_id"] = f.PersonID.String()
		}
	}

	return utils.SuccessResponse(c, "Faces retrieved", fiber.Map{
		"faces": response,
		"count": len(response),
	})
}

// GetProcessingStats returns face processing statistics
// @Summary Get face processing statistics
// @Tags Faces
// @Produce json
// @Success 200 {object} utils.Response
// @Router /api/v1/faces/stats [get]
func (h *FaceHandler) GetProcessingStats(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	stats, err := h.faceService.GetProcessingStats(c.Context(), userCtx.ID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to get stats", err)
	}

	return utils.SuccessResponse(c, "Stats retrieved", stats)
}

// GetFaces returns paginated faces for a user
// @Summary Get all faces with pagination
// @Tags Faces
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Page size" default(50)
// @Success 200 {object} utils.Response
// @Router /api/v1/faces [get]
func (h *FaceHandler) GetFaces(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 50
	}

	faces, total, err := h.faceService.GetFaces(c.Context(), userCtx.ID, page, limit)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to get faces", err)
	}

	// Convert to response format
	response := make([]fiber.Map, len(faces))
	for i, f := range faces {
		response[i] = fiber.Map{
			"id":          f.ID.String(),
			"photo_id":    f.PhotoID.String(),
			"bbox_x":      f.BboxX,
			"bbox_y":      f.BboxY,
			"bbox_width":  f.BboxWidth,
			"bbox_height": f.BboxHeight,
			"confidence":  f.Confidence,
			"person_id":   nil,
			"created_at":  f.CreatedAt,
		}
		if f.PersonID != nil {
			response[i]["person_id"] = f.PersonID.String()
		}
	}

	return utils.SuccessResponse(c, "Faces retrieved", fiber.Map{
		"faces": response,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// isValidImageType checks if the content type is a valid image
func isValidImageType(contentType string) bool {
	validTypes := []string{
		"image/jpeg",
		"image/png",
		"image/webp",
		"image/gif",
	}
	for _, t := range validTypes {
		if contentType == t {
			return true
		}
	}
	return false
}
