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

type ActivityLogHandler struct {
	activityLogService services.ActivityLogService
	sharedFolderRepo   repositories.SharedFolderRepository
}

func NewActivityLogHandler(
	activityLogService services.ActivityLogService,
	sharedFolderRepo repositories.SharedFolderRepository,
) *ActivityLogHandler {
	return &ActivityLogHandler{
		activityLogService: activityLogService,
		sharedFolderRepo:   sharedFolderRepo,
	}
}

// GetActivityLogs returns activity logs for a folder
// @Summary Get activity logs for a folder
// @Tags Activity
// @Security BearerAuth
// @Param folderId query string true "Folder ID"
// @Param activityType query string false "Filter by activity type"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(50)
// @Success 200 {object} map[string]interface{}
// @Router /activity-logs [get]
func (h *ActivityLogHandler) GetActivityLogs(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	// Parse folder ID
	folderIDStr := c.Query("folderId")
	if folderIDStr == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "folderId is required",
		})
	}

	folderID, err := uuid.Parse(folderIDStr)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid folder ID",
		})
	}

	// Verify user has access to this folder
	hasAccess, err := h.sharedFolderRepo.HasUserAccess(c.Context(), userCtx.ID, folderID)
	if err != nil || !hasAccess {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"success": false,
			"error":   "Access denied",
		})
	}

	// Parse pagination
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100
	}

	// Parse optional activity type filter
	activityType := c.Query("activityType")

	var logs []models.ActivityLog
	var total int64

	if activityType != "" {
		logs, total, err = h.activityLogService.GetByFolderAndType(c.Context(), folderID, models.ActivityType(activityType), page, limit)
	} else {
		logs, total, err = h.activityLogService.GetByFolder(c.Context(), folderID, page, limit)
	}

	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Calculate total pages
	totalPages := int(total) / limit
	if int(total)%limit > 0 {
		totalPages++
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ActivityLogsToResponse(logs),
		"meta": fiber.Map{
			"total":      total,
			"page":       page,
			"limit":      limit,
			"totalPages": totalPages,
			"hasNext":    page < totalPages,
			"hasPrev":    page > 1,
		},
	})
}

// GetRecentActivityLogs returns recent activity logs across all folders (for admin)
// @Summary Get recent activity logs
// @Tags Activity
// @Security BearerAuth
// @Param limit query int false "Number of logs" default(50)
// @Success 200 {object} map[string]interface{}
// @Router /activity-logs/recent [get]
func (h *ActivityLogHandler) GetRecentActivityLogs(c *fiber.Ctx) error {
	_, err := utils.GetUserFromContext(c)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Unauthorized",
		})
	}

	limit := c.QueryInt("limit", 50)
	if limit > 100 {
		limit = 100
	}

	logs, err := h.activityLogService.GetRecent(c.Context(), limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    dto.ActivityLogsToResponse(logs),
	})
}

// GetActivityTypes returns all available activity types
// @Summary Get activity types
// @Tags Activity
// @Success 200 {object} map[string]interface{}
// @Router /activity-logs/types [get]
func (h *ActivityLogHandler) GetActivityTypes(c *fiber.Ctx) error {
	types := []fiber.Map{
		{"value": "sync_started", "label": "เริ่มซิงค์", "category": "sync"},
		{"value": "sync_completed", "label": "ซิงค์สำเร็จ", "category": "sync"},
		{"value": "sync_failed", "label": "ซิงค์ล้มเหลว", "category": "sync"},
		{"value": "photos_added", "label": "เพิ่มรูปภาพ", "category": "photo"},
		{"value": "photos_trashed", "label": "ย้ายรูปไปถังขยะ", "category": "photo"},
		{"value": "photos_restored", "label": "กู้คืนรูปภาพ", "category": "photo"},
		{"value": "photos_deleted", "label": "ลบรูปภาพถาวร", "category": "photo"},
		{"value": "folder_trashed", "label": "ย้ายโฟลเดอร์ไปถังขยะ", "category": "folder"},
		{"value": "folder_restored", "label": "กู้คืนโฟลเดอร์", "category": "folder"},
		{"value": "folder_renamed", "label": "เปลี่ยนชื่อโฟลเดอร์", "category": "folder"},
		{"value": "folder_deleted", "label": "ลบโฟลเดอร์ถาวร", "category": "folder"},
		{"value": "webhook_received", "label": "รับ Webhook", "category": "webhook"},
		{"value": "webhook_renewed", "label": "ต่ออายุ Webhook", "category": "webhook"},
		{"value": "webhook_expired", "label": "Webhook หมดอายุ", "category": "webhook"},
		{"value": "token_expired", "label": "Token หมดอายุ", "category": "error"},
		{"value": "sync_error", "label": "เกิดข้อผิดพลาด", "category": "error"},
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data":    types,
	})
}
