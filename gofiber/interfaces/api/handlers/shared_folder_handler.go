package handlers

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"gofiber-template/domain/dto"
	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"gofiber-template/pkg/utils"
)

type SharedFolderHandler struct {
	sharedFolderService services.SharedFolderService
	photoRepo           repositories.PhotoRepository
	sharedFolderRepo    repositories.SharedFolderRepository
	userRepo            repositories.UserRepository
}

func NewSharedFolderHandler(
	sharedFolderService services.SharedFolderService,
	photoRepo repositories.PhotoRepository,
	sharedFolderRepo repositories.SharedFolderRepository,
	userRepo repositories.UserRepository,
) *SharedFolderHandler {
	return &SharedFolderHandler{
		sharedFolderService: sharedFolderService,
		photoRepo:           photoRepo,
		sharedFolderRepo:    sharedFolderRepo,
		userRepo:            userRepo,
	}
}

// ListFolders returns all folders the user has access to (with sub-folders as children)
// @Summary List user's folders
// @Tags Folders
// @Security BearerAuth
// @Success 200 {object} dto.SharedFolderListResponse
// @Router /folders [get]
func (h *SharedFolderHandler) ListFolders(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	folders, err := h.sharedFolderService.GetUserFolders(c.Context(), userCtx.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Build response with counts and children (sub-folders)
	responses := make([]dto.SharedFolderResponse, 0, len(folders))
	for _, folder := range folders {
		photoCount, _ := h.photoRepo.CountBySharedFolder(c.Context(), folder.ID)
		userCount, _ := h.sharedFolderRepo.CountUsers(c.Context(), folder.ID)
		response := dto.SharedFolderToResponse(&folder, photoCount, userCount)

		// Get sub-folders (children) for this folder
		paths, _ := h.photoRepo.GetFolderPathsInSharedFolder(c.Context(), folder.ID)
		children := make([]dto.SubFolderInfo, 0, len(paths))
		for _, path := range paths {
			// Get photo count for this path
			_, count, _ := h.photoRepo.GetBySharedFolderAndPath(c.Context(), folder.ID, path, 0, 0)

			// Extract folder name from path (last segment)
			name := path
			for i := len(path) - 1; i >= 0; i-- {
				if path[i] == '/' {
					name = path[i+1:]
					break
				}
			}

			children = append(children, dto.SubFolderInfo{
				Path:       path,
				Name:       name,
				PhotoCount: count,
			})
		}
		response.Children = children

		responses = append(responses, *response)
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.SharedFolderListResponse{
			Folders: responses,
		},
	})
}

// GetFolder returns a single folder by ID
// @Summary Get folder by ID
// @Tags Folders
// @Security BearerAuth
// @Param id path string true "Folder ID"
// @Success 200 {object} dto.SharedFolderResponse
// @Router /folders/{id} [get]
func (h *SharedFolderHandler) GetFolder(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid folder ID",
		})
	}

	folder, err := h.sharedFolderService.GetFolderByID(c.Context(), userCtx.ID, folderID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Folder not found",
		})
	}

	photoCount, _ := h.photoRepo.CountBySharedFolder(c.Context(), folder.ID)
	userCount, _ := h.sharedFolderRepo.CountUsers(c.Context(), folder.ID)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.SharedFolderToResponse(folder, photoCount, userCount),
	})
}

// AddFolder adds a new folder or joins existing one
// @Summary Add folder
// @Tags Folders
// @Security BearerAuth
// @Accept json
// @Param body body dto.AddFolderRequest true "Folder info"
// @Success 200 {object} dto.SharedFolderResponse
// @Router /folders [post]
func (h *SharedFolderHandler) AddFolder(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	var req dto.AddFolderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid request body",
		})
	}

	if req.DriveFolderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "drive_folder_id is required",
		})
	}

	// Get user's OAuth tokens from database
	user, err := h.userRepo.GetByID(c.Context(), userCtx.ID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   "Failed to get user",
		})
	}

	if user.DriveAccessToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Google Drive not connected. Please reconnect your Google account.",
		})
	}

	folder, err := h.sharedFolderService.AddFolder(c.Context(), userCtx.ID, req.DriveFolderID, user.DriveAccessToken, user.DriveRefreshToken)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	photoCount, _ := h.photoRepo.CountBySharedFolder(c.Context(), folder.ID)
	userCount, _ := h.sharedFolderRepo.CountUsers(c.Context(), folder.ID)

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.SharedFolderToResponse(folder, photoCount, userCount),
	})
}

// RemoveFolder removes user's access to a folder
// @Summary Leave folder
// @Tags Folders
// @Security BearerAuth
// @Param id path string true "Folder ID"
// @Success 200
// @Router /folders/{id} [delete]
func (h *SharedFolderHandler) RemoveFolder(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid folder ID",
		})
	}

	if err := h.sharedFolderService.RemoveUserAccess(c.Context(), userCtx.ID, folderID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Left folder successfully",
	})
}

// TriggerSync triggers sync for a folder
// @Summary Trigger folder sync
// @Tags Folders
// @Security BearerAuth
// @Param id path string true "Folder ID"
// @Success 200
// @Router /folders/{id}/sync [post]
func (h *SharedFolderHandler) TriggerSync(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid folder ID",
		})
	}

	if err := h.sharedFolderService.TriggerSync(c.Context(), userCtx.ID, folderID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Sync triggered",
	})
}

// GetPhotos returns photos from a folder
// @Summary Get photos from folder
// @Tags Folders
// @Security BearerAuth
// @Param id path string true "Folder ID"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Param folder_path query string false "Filter by sub-folder path"
// @Success 200 {object} dto.PhotoListResponse
// @Router /folders/{id}/photos [get]
func (h *SharedFolderHandler) GetPhotos(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid folder ID",
		})
	}

	// Verify access
	hasAccess, err := h.sharedFolderRepo.HasUserAccess(c.Context(), userCtx.ID, folderID)
	if err != nil || !hasAccess {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Folder not found",
		})
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	offset := (page - 1) * limit
	folderPath := c.Query("folder_path", "")

	var photos []models.Photo
	var total int64

	if folderPath != "" {
		// Filter by specific sub-folder path
		photos, total, err = h.photoRepo.GetBySharedFolderAndPath(c.Context(), folderID, folderPath, offset, limit)
	} else {
		// Get all photos in shared folder
		photos, total, err = h.photoRepo.GetBySharedFolder(c.Context(), folderID, offset, limit)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": dto.PhotoListResponse{
			Photos: dto.PhotosToPhotoResponses(photos),
			Total:  total,
			Page:   page,
			Limit:  limit,
		},
	})
}

// RegisterWebhook registers a webhook for an existing folder
// @Summary Register webhook for folder
// @Tags Folders
// @Security BearerAuth
// @Param id path string true "Folder ID"
// @Success 200
// @Router /folders/{id}/webhook [post]
func (h *SharedFolderHandler) RegisterWebhook(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid folder ID",
		})
	}

	if err := h.sharedFolderService.RegisterWebhook(c.Context(), userCtx.ID, folderID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"message": "Webhook registered successfully",
	})
}

// GetSubFolders returns distinct sub-folder paths within a shared folder
// @Summary Get sub-folders in a shared folder
// @Tags Folders
// @Security BearerAuth
// @Param id path string true "Folder ID"
// @Success 200 {object} map[string]interface{}
// @Router /folders/{id}/subfolders [get]
func (h *SharedFolderHandler) GetSubFolders(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	folderID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid folder ID",
		})
	}

	// Verify access
	hasAccess, err := h.sharedFolderRepo.HasUserAccess(c.Context(), userCtx.ID, folderID)
	if err != nil || !hasAccess {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"success": false,
			"error":   "Folder not found",
		})
	}

	// Get distinct folder paths
	paths, err := h.photoRepo.GetFolderPathsInSharedFolder(c.Context(), folderID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Build response with photo counts for each sub-folder
	type SubFolderInfo struct {
		Path       string `json:"path"`
		Name       string `json:"name"`
		PhotoCount int64  `json:"photo_count"`
	}

	subFolders := make([]SubFolderInfo, 0, len(paths))
	for _, path := range paths {
		// Get photo count for this path
		photos, count, _ := h.photoRepo.GetBySharedFolderAndPath(c.Context(), folderID, path, 0, 0)
		_ = photos // We only need the count

		// Extract the folder name from path (last segment)
		name := path
		if idx := len(path) - 1; idx >= 0 {
			for i := len(path) - 1; i >= 0; i-- {
				if path[i] == '/' {
					name = path[i+1:]
					break
				}
			}
		}

		subFolders = append(subFolders, SubFolderInfo{
			Path:       path,
			Name:       name,
			PhotoCount: count,
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"subfolders": subFolders,
			"total":      len(subFolders),
		},
	})
}
