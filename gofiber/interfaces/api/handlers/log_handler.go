package handlers

import (
	"os"

	"github.com/gofiber/fiber/v2"

	"gofiber-template/pkg/config"
	"gofiber-template/pkg/logger"
)

// LogHandler handles log-related API requests
type LogHandler struct {
	adminToken string
}

// NewLogHandler creates a new log handler
func NewLogHandler(cfg *config.Config) *LogHandler {
	// Use JWT secret as admin token for simplicity
	// In production, you might want a separate admin token
	return &LogHandler{
		adminToken: cfg.JWT.Secret,
	}
}

// GetLogs returns log entries
// @Summary Get application logs
// @Tags Admin
// @Security AdminToken
// @Param lines query int false "Number of lines" default(100)
// @Param level query string false "Filter by level (DEBUG, INFO, WARN, ERROR)"
// @Param category query string false "Filter by category (auth, webhook, websocket, sync, api, db, drive)"
// @Param search query string false "Search in message/action"
// @Success 200 {object} map[string]interface{}
// @Router /admin/logs [get]
func (h *LogHandler) GetLogs(c *fiber.Ctx) error {
	// Check admin token
	token := c.Get("X-Admin-Token")
	if token == "" {
		token = c.Query("token")
	}
	if token != h.adminToken {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid admin token",
		})
	}

	// Parse options
	opts := logger.ReadLogsOptions{
		Lines:    c.QueryInt("lines", 100),
		Level:    logger.Level(c.Query("level")),
		Category: logger.Category(c.Query("category")),
		Search:   c.Query("search"),
	}

	// Read logs
	entries, err := logger.ReadLogs(opts)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"entries": entries,
			"count":   len(entries),
			"filters": fiber.Map{
				"lines":    opts.Lines,
				"level":    opts.Level,
				"category": opts.Category,
				"search":   opts.Search,
			},
		},
	})
}

// GetLogFiles returns list of log files
// @Summary List log files
// @Tags Admin
// @Security AdminToken
// @Success 200 {object} map[string]interface{}
// @Router /admin/logs/files [get]
func (h *LogHandler) GetLogFiles(c *fiber.Ctx) error {
	// Check admin token
	token := c.Get("X-Admin-Token")
	if token == "" {
		token = c.Query("token")
	}
	if token != h.adminToken {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid admin token",
		})
	}

	files, err := logger.ListLogFiles()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"files":  files,
			"logDir": logger.GetLogDir(),
		},
	})
}

// GetFolderLogs returns all logs related to a specific folder
// @Summary Get logs for a specific folder
// @Tags Admin
// @Security AdminToken
// @Param id path string true "Folder ID (UUID or Drive Folder ID)"
// @Param days query int false "Number of days to search" default(7)
// @Success 200 {object} map[string]interface{}
// @Router /admin/logs/folder/{id} [get]
func (h *LogHandler) GetFolderLogs(c *fiber.Ctx) error {
	// Check admin token
	token := c.Get("X-Admin-Token")
	if token == "" {
		token = c.Query("token")
	}
	if token != h.adminToken {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid admin token",
		})
	}

	// Get folder ID from path
	folderID := c.Params("id")
	if folderID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"error":   "Folder ID is required",
		})
	}

	// Get days parameter
	days := c.QueryInt("days", 7)

	// Read logs for this folder
	entries, err := logger.ReadLogsByFolderID(folderID, days)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"success": false,
			"error":   err.Error(),
		})
	}

	// Group entries by action for summary
	actionSummary := make(map[string]int)
	var errors []fiber.Map
	for _, entry := range entries {
		actionSummary[entry.Action]++
		if entry.Level == logger.LevelError {
			errorDetail := fiber.Map{
				"timestamp": entry.Timestamp,
				"action":    entry.Action,
				"message":   entry.Message,
				"error":     entry.Error,
			}
			// Include Google API error details if available
			if entry.Data != nil {
				if httpCode, ok := entry.Data["http_status_code"]; ok {
					errorDetail["http_status_code"] = httpCode
				}
				if googleMsg, ok := entry.Data["google_error_message"]; ok {
					errorDetail["google_error_message"] = googleMsg
				}
				if googleDetails, ok := entry.Data["google_error_details"]; ok {
					errorDetail["google_error_details"] = googleDetails
				}
				if rawErr, ok := entry.Data["raw_error"]; ok {
					errorDetail["raw_error"] = rawErr
				}
			}
			errors = append(errors, errorDetail)
		}
	}

	// Determine status
	status := "success"
	if len(errors) > 0 {
		status = "has_errors"
	}
	if len(entries) == 0 {
		status = "no_logs_found"
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"folder_id":      folderID,
			"days_searched":  days,
			"status":         status,
			"total_entries":  len(entries),
			"error_count":    len(errors),
			"errors":         errors,
			"action_summary": actionSummary,
			"timeline":       entries,
		},
	})
}

// GetLogStats returns log statistics
// @Summary Get log statistics
// @Tags Admin
// @Security AdminToken
// @Success 200 {object} map[string]interface{}
// @Router /admin/logs/stats [get]
func (h *LogHandler) GetLogStats(c *fiber.Ctx) error {
	// Check admin token
	token := c.Get("X-Admin-Token")
	if token == "" {
		token = c.Query("token")
	}
	if token != h.adminToken {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"success": false,
			"error":   "Invalid admin token",
		})
	}

	// Get all logs for today
	allLogs, _ := logger.ReadLogs(logger.ReadLogsOptions{Lines: 1000})

	// Count by level
	levelCounts := map[string]int{
		"DEBUG": 0,
		"INFO":  0,
		"WARN":  0,
		"ERROR": 0,
	}

	// Count by category
	categoryCounts := map[string]int{}

	for _, entry := range allLogs {
		levelCounts[string(entry.Level)]++
		categoryCounts[string(entry.Category)]++
	}

	// Get log directory size
	var totalSize int64
	files, _ := logger.ListLogFiles()
	logDir := logger.GetLogDir()
	for _, f := range files {
		if info, err := os.Stat(logDir + "/" + f); err == nil {
			totalSize += info.Size()
		}
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"total_entries":    len(allLogs),
			"by_level":         levelCounts,
			"by_category":      categoryCounts,
			"total_files":      len(files),
			"total_size_bytes": totalSize,
		},
	})
}
