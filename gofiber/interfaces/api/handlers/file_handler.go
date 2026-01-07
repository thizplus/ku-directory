package handlers

import (
	"strconv"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gofiber-template/domain/dto"
	"gofiber-template/domain/services"
	"gofiber-template/pkg/utils"
)

type FileHandler struct {
	fileService services.FileService
}

func NewFileHandler(fileService services.FileService) *FileHandler {
	return &FileHandler{
		fileService: fileService,
	}
}

// UploadFile handles file uploads with flexible path support
// Supports two approaches:
// 1. Custom Path: Provide 'custom_path' in form-data to specify exact upload path
// 2. Structured Path: Use 'category', 'entity_id', and 'file_type' for organized structure
// Form data parameters:
// - file: The file to upload (required) - filename and MIME type auto-extracted
// - custom_path: Custom path for the file (optional, mutually exclusive with structured approach)
// - category: Category for structured path (optional)
// - entity_id: Entity ID for structured path (optional, requires valid UUID)
// - file_type: File type for structured path (optional)
// Note: filename, MIME type, and file size are automatically extracted from the uploaded file
func (h *FileHandler) UploadFile(c *fiber.Ctx) error {
	user, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "User not authenticated")
	}

	// Validate file upload first
	file, err := c.FormFile("file")
	if err != nil {
		return utils.ValidationErrorResponse(c, "No file provided")
	}

	// Basic file validation
	if file.Size == 0 {
		return utils.ValidationErrorResponse(c, "Empty file not allowed")
	}

	// Parse upload options from form data
	options := &dto.UploadFileRequest{
		CustomPath: c.FormValue("custom_path"),
		Category:   c.FormValue("category"),
		EntityID:   c.FormValue("entity_id"),
		FileType:   c.FormValue("file_type"),
	}

	// Validate the upload options
	if err := utils.ValidateStruct(options); err != nil {
		errors := utils.GetValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Upload options validation failed",
			"errors":  errors,
		})
	}

	// Validate that either custom_path OR structured path fields are provided
	hasCustomPath := options.CustomPath != ""
	hasStructuredPath := options.Category != "" || options.EntityID != "" || options.FileType != ""

	if hasCustomPath && hasStructuredPath {
		return utils.ValidationErrorResponse(c, "Cannot use both custom_path and structured path fields (category/entity_id/file_type) simultaneously")
	}

	fileModel, err := h.fileService.UploadFile(c.Context(), user.ID, file, options)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "File upload failed", err)
	}

	// Determine path type for response
	pathType := "structured"
	if options.CustomPath != "" {
		pathType = "custom"
	}

	uploadResponse := &dto.UploadResponse{
		FileID:   fileModel.ID,
		FileName: fileModel.FileName,
		URL:      fileModel.URL,
		CDNPath:  fileModel.CDNPath,
		FileSize: fileModel.FileSize,
		MimeType: fileModel.MimeType,
		PathType: pathType,
	}

	return utils.SuccessResponse(c, "File uploaded successfully", uploadResponse)
}

func (h *FileHandler) GetFile(c *fiber.Ctx) error {
	fileIDStr := c.Params("id")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid file ID")
	}

	file, err := h.fileService.GetFile(c.Context(), fileID)
	if err != nil {
		return utils.NotFoundResponse(c, "File not found")
	}

	fileResponse := dto.FileToFileResponse(file)
	return utils.SuccessResponse(c, "File retrieved successfully", fileResponse)
}

func (h *FileHandler) DeleteFile(c *fiber.Ctx) error {
	fileIDStr := c.Params("id")
	fileID, err := uuid.Parse(fileIDStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid file ID")
	}

	err = h.fileService.DeleteFile(c.Context(), fileID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "File deletion failed", err)
	}

	return utils.SuccessResponse(c, "File deleted successfully", nil)
}

func (h *FileHandler) GetUserFiles(c *fiber.Ctx) error {
	user, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "User not authenticated")
	}

	offsetStr := c.Query("offset", "0")
	limitStr := c.Query("limit", "10")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid offset parameter")
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid limit parameter")
	}

	files, total, err := h.fileService.GetUserFiles(c.Context(), user.ID, offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to retrieve files", err)
	}

	fileResponses := make([]dto.FileResponse, len(files))
	for i, file := range files {
		fileResponses[i] = *dto.FileToFileResponse(file)
	}

	response := &dto.FileListResponse{
		Files: fileResponses,
		Meta: dto.PaginationMeta{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return utils.SuccessResponse(c, "Files retrieved successfully", response)
}

func (h *FileHandler) ListFiles(c *fiber.Ctx) error {
	offsetStr := c.Query("offset", "0")
	limitStr := c.Query("limit", "10")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid offset parameter")
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid limit parameter")
	}

	files, total, err := h.fileService.ListFiles(c.Context(), offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to retrieve files", err)
	}

	fileResponses := make([]dto.FileResponse, len(files))
	for i, file := range files {
		fileResponses[i] = *dto.FileToFileResponse(file)
	}

	response := &dto.FileListResponse{
		Files: fileResponses,
		Meta: dto.PaginationMeta{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return utils.SuccessResponse(c, "Files retrieved successfully", response)
}