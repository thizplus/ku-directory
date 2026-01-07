package handlers

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"gofiber-template/domain/dto"
	"gofiber-template/domain/models"
	"gofiber-template/domain/services"
	"gofiber-template/infrastructure/googledrive"
	"gofiber-template/infrastructure/websocket"
	"gofiber-template/pkg/logger"
	"gofiber-template/pkg/utils"
)

// getFrontendURL returns the frontend URL for redirects
func getFrontendURL() string {
	if url := os.Getenv("FRONTEND_URL"); url != "" {
		return url
	}
	return "http://localhost:5173" // Default for development
}

type DriveHandler struct {
	driveService        services.DriveService
	sharedFolderService services.SharedFolderService
}

func NewDriveHandler(driveService services.DriveService) *DriveHandler {
	return &DriveHandler{
		driveService: driveService,
	}
}

// SetSharedFolderService sets the shared folder service for webhook handling
func (h *DriveHandler) SetSharedFolderService(svc services.SharedFolderService) {
	h.sharedFolderService = svc
}

// getJWTSecret returns the JWT secret for HMAC signing
func getJWTSecret() string {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		return secret
	}
	return "default-secret-change-in-production"
}

// createSignedState creates a signed state containing userID
// Format: randomState.userID.timestamp.signature
func createSignedState(userID string) (string, error) {
	randomBytes := make([]byte, 16)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	randomPart := base64.URLEncoding.EncodeToString(randomBytes)

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	data := randomPart + "." + userID + "." + timestamp

	// Create HMAC signature
	h := hmac.New(sha256.New, []byte(getJWTSecret()))
	h.Write([]byte(data))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	return data + "." + signature, nil
}

// parseSignedState parses and validates a signed state
// Returns userID if valid, error otherwise
func parseSignedState(state string) (string, error) {
	parts := strings.Split(state, ".")
	if len(parts) != 4 {
		return "", fmt.Errorf("invalid state format")
	}

	randomPart, userID, timestampStr, providedSig := parts[0], parts[1], parts[2], parts[3]

	// Verify timestamp (not older than 10 minutes)
	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid timestamp")
	}
	if time.Now().Unix()-timestamp > 600 { // 10 minutes
		return "", fmt.Errorf("state expired")
	}

	// Verify signature
	data := randomPart + "." + userID + "." + timestampStr
	h := hmac.New(sha256.New, []byte(getJWTSecret()))
	h.Write([]byte(data))
	expectedSig := base64.URLEncoding.EncodeToString(h.Sum(nil))

	if !hmac.Equal([]byte(providedSig), []byte(expectedSig)) {
		return "", fmt.Errorf("invalid signature")
	}

	return userID, nil
}

// Connect initiates Google Drive OAuth flow
// Returns the auth URL for frontend to redirect
func (h *DriveHandler) Connect(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	logger.Drive("DRIVE_CONNECT_START", "User initiating Drive connection", map[string]interface{}{
		"user_id": userCtx.ID.String(),
	})

	// Create signed state containing user ID
	state, err := createSignedState(userCtx.ID.String())
	if err != nil {
		logger.DriveError("DRIVE_CONNECT_ERROR", "Failed to generate state", err, nil)
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to generate state", err)
	}

	// Return auth URL for frontend to redirect
	authURL := h.driveService.GetAuthURL(state)

	logger.Drive("DRIVE_CONNECT_URL", "Auth URL generated for Drive connection", map[string]interface{}{
		"user_id": userCtx.ID.String(),
	})

	return utils.SuccessResponse(c, "Auth URL generated", fiber.Map{
		"authUrl": authURL,
	})
}

// Callback handles Google Drive OAuth callback
func (h *DriveHandler) Callback(c *fiber.Ctx) error {
	frontendURL := getFrontendURL()

	logger.Drive("DRIVE_CALLBACK_START", "Received Drive OAuth callback", map[string]interface{}{
		"query": c.OriginalURL(),
	})

	// Get and verify state (contains signed userID)
	state := c.Query("state")

	if state == "" {
		logger.DriveError("DRIVE_CALLBACK_ERROR", "Missing state parameter", nil, nil)
		return c.Redirect(frontendURL + "/settings?error=missing_state")
	}

	// Parse signed state to get userID
	userIDStr, err := parseSignedState(state)
	if err != nil {
		logger.DriveError("DRIVE_CALLBACK_ERROR", "Invalid state", err, nil)
		return c.Redirect(frontendURL + "/settings?error=invalid_state")
	}

	logger.Drive("DRIVE_CALLBACK_STATE", "Parsed userID from state", map[string]interface{}{
		"user_id": userIDStr,
	})

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		logger.DriveError("DRIVE_CALLBACK_ERROR", "Invalid user ID format", err, map[string]interface{}{
			"user_id_str": userIDStr,
		})
		return c.Redirect(frontendURL + "/settings?error=invalid_user")
	}

	// Check for error from Google
	if errMsg := c.Query("error"); errMsg != "" {
		logger.DriveError("DRIVE_CALLBACK_ERROR", "Google returned error", nil, map[string]interface{}{
			"google_error": errMsg,
		})
		return c.Redirect(fmt.Sprintf("%s/settings?error=%s", frontendURL, errMsg))
	}

	// Get authorization code
	code := c.Query("code")
	if code == "" {
		logger.DriveError("DRIVE_CALLBACK_ERROR", "Missing authorization code", nil, nil)
		return c.Redirect(frontendURL + "/settings?error=missing_code")
	}

	logger.Drive("DRIVE_CALLBACK_EXCHANGE", "Exchanging code for token", map[string]interface{}{
		"user_id":     userID.String(),
		"code_length": len(code),
	})

	// Handle callback
	if err := h.driveService.HandleCallback(c.Context(), userID, code); err != nil {
		logger.DriveError("DRIVE_CALLBACK_ERROR", "HandleCallback failed", err, map[string]interface{}{
			"user_id": userID.String(),
		})
		return c.Redirect(fmt.Sprintf("%s/settings?error=auth_failed&message=%s", frontendURL, err.Error()))
	}

	logger.Drive("DRIVE_CALLBACK_SUCCESS", "Drive connected successfully", map[string]interface{}{
		"user_id": userID.String(),
	})

	// Redirect to settings with success
	return c.Redirect(frontendURL + "/settings?drive=connected")
}

// Disconnect removes Google Drive connection
func (h *DriveHandler) Disconnect(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	if err := h.driveService.Disconnect(c.Context(), userCtx.ID); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to disconnect", err)
	}

	return utils.SuccessResponse(c, "Google Drive disconnected", nil)
}

// Status returns the connection status
func (h *DriveHandler) Status(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	connected := h.driveService.IsConnected(c.Context(), userCtx.ID)

	var rootFolder interface{}
	if connected {
		folderInfo, err := h.driveService.GetRootFolderInfo(c.Context(), userCtx.ID)
		if err == nil && folderInfo != nil {
			rootFolder = folderInfo
		}
	}

	return utils.SuccessResponse(c, "Drive status", fiber.Map{
		"connected":  connected,
		"rootFolder": rootFolder,
	})
}

// ListFolders lists folders in Google Drive
func (h *DriveHandler) ListFolders(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	parentID := c.Query("parent", "")

	folders, err := h.driveService.ListFolders(c.Context(), userCtx.ID, parentID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to list folders", err)
	}

	return utils.SuccessResponse(c, "Folders retrieved", folders)
}

// SetRootFolder sets the root folder for sync
func (h *DriveHandler) SetRootFolder(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	var req struct {
		FolderID string `json:"folderId"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request", err)
	}

	if err := h.driveService.SetRootFolder(c.Context(), userCtx.ID, req.FolderID); err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to set root folder", err)
	}

	return utils.SuccessResponse(c, "Root folder set", nil)
}

// StartSync starts a sync job
func (h *DriveHandler) StartSync(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	job, err := h.driveService.StartSync(c.Context(), userCtx.ID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to start sync", err)
	}

	return utils.SuccessResponse(c, "Sync started", job)
}

// GetSyncStatus gets the sync status
func (h *DriveHandler) GetSyncStatus(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	job, err := h.driveService.GetSyncStatus(c.Context(), userCtx.ID)
	if err != nil {
		return utils.SuccessResponse(c, "No sync jobs", nil)
	}

	return utils.SuccessResponse(c, "Sync status", job)
}

// GetPhotos gets paginated photos
func (h *DriveHandler) GetPhotos(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 20)
	folderId := c.Query("folder", "")
	search := c.Query("search", "")

	var photos []models.Photo
	var total int64

	if search != "" {
		// Search by folder path (activity name)
		photos, total, err = h.driveService.SearchPhotos(c.Context(), userCtx.ID, search, page, limit)
	} else if folderId != "" {
		photos, total, err = h.driveService.GetPhotosByFolderId(c.Context(), userCtx.ID, folderId, page, limit)
	} else {
		photos, total, err = h.driveService.GetPhotos(c.Context(), userCtx.ID, page, limit)
	}

	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to get photos", err)
	}

	// Convert to DTO
	photoResponses := dto.PhotosToPhotoResponses(photos)

	return utils.SuccessResponse(c, "Photos retrieved", dto.PhotoListResponse{
		Photos: photoResponses,
		Total:  total,
		Page:   page,
		Limit:  limit,
	})
}

// GetThumbnail proxies thumbnail requests to Google Drive with authentication
func (h *DriveHandler) GetThumbnail(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	driveFileID := c.Params("driveFileId")
	if driveFileID == "" {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Drive file ID is required", nil)
	}

	size := c.QueryInt("size", 400) // Default thumbnail size

	data, contentType, err := h.driveService.GetPhotoThumbnail(c.Context(), userCtx.ID, driveFileID, size)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to get thumbnail", err)
	}

	// Set cache headers (cache for 1 hour)
	c.Set("Cache-Control", "public, max-age=3600")
	c.Set("Content-Type", contentType)

	return c.Send(data)
}

// DownloadPhotos downloads multiple photos as a zip file
func (h *DriveHandler) DownloadPhotos(c *fiber.Ctx) error {
	userCtx, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "Not authenticated")
	}

	var req struct {
		DriveFileIDs []string `json:"driveFileIds"`
	}

	if err := c.BodyParser(&req); err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Invalid request", err)
	}

	if len(req.DriveFileIDs) == 0 {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "No files specified", nil)
	}

	// Progress callback - sends WebSocket message for each file
	onProgress := func(progress services.DownloadProgress) {
		websocket.Manager.BroadcastToUser(userCtx.ID, "download:progress", map[string]interface{}{
			"current":  progress.Current,
			"total":    progress.Total,
			"fileName": progress.FileName,
		})
	}

	zipData, err := h.driveService.DownloadPhotosAsZip(c.Context(), userCtx.ID, req.DriveFileIDs, onProgress)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to create zip", err)
	}

	// Send completion message
	websocket.Manager.BroadcastToUser(userCtx.ID, "download:completed", map[string]interface{}{
		"total": len(req.DriveFileIDs),
	})

	// Set headers for zip download
	c.Set("Content-Type", "application/zip")
	c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"photos_%s.zip\"", time.Now().Format("20060102_150405")))
	c.Set("Content-Length", strconv.Itoa(len(zipData)))

	return c.Send(zipData)
}

// Webhook handles Google Drive push notifications
func (h *DriveHandler) Webhook(c *fiber.Ctx) error {
	payload := googledrive.ParseWebhookHeaders(c.GetReqHeaders())

	// Log webhook received
	logger.Webhook("WEBHOOK_RECEIVED", "Google Drive webhook received", map[string]interface{}{
		"channel_id":     payload.ChannelID,
		"resource_id":    payload.ResourceID,
		"resource_state": payload.ResourceState,
		"channel_token":  payload.ChannelToken,
		"resource_uri":   payload.ResourceURI,
		"headers":        c.GetReqHeaders(),
	})

	if payload.ChannelID == "" {
		logger.WebhookError("WEBHOOK_REJECTED", "Missing ChannelID", nil, nil)
		return c.SendStatus(fiber.StatusBadRequest)
	}

	// Copy values before goroutine (fasthttp context becomes invalid after handler returns)
	channelID := payload.ChannelID
	resourceID := payload.ResourceID
	resourceState := payload.ResourceState
	channelToken := payload.ChannelToken

	// Process webhook asynchronously with background context
	// IMPORTANT: Cannot use c.Context() in goroutine - it becomes nil after handler returns
	go func() {
		ctx := context.Background()

		logger.Webhook("WEBHOOK_PROCESSING", "Processing webhook asynchronously", map[string]interface{}{
			"resource_state": resourceState,
			"channel_token":  channelToken,
		})

		// Try user-based webhook first
		err := h.driveService.HandleWebhook(
			ctx,
			channelID,
			resourceID,
			resourceState,
			channelToken,
		)

		if err != nil {
			logger.Webhook("WEBHOOK_USER_FAILED", "User webhook failed, trying shared folder", map[string]interface{}{
				"error": err.Error(),
			})

			// Try shared folder webhook
			if h.sharedFolderService != nil {
				err = h.sharedFolderService.HandleWebhook(
					ctx,
					channelID,
					resourceID,
					resourceState,
					channelToken,
				)
				if err != nil {
					logger.WebhookError("WEBHOOK_FAILED", "Both user and shared folder webhook failed", err, map[string]interface{}{
						"channel_id": channelID,
					})
				} else {
					logger.Webhook("WEBHOOK_SUCCESS", "SharedFolder webhook processed successfully", map[string]interface{}{
						"channel_id": channelID,
					})
				}
			} else {
				logger.WebhookError("WEBHOOK_ERROR", "SharedFolderService not available", nil, nil)
			}
		} else {
			logger.Webhook("WEBHOOK_SUCCESS", "User webhook processed successfully", map[string]interface{}{
				"channel_id": channelID,
			})
		}
	}()

	return c.SendStatus(fiber.StatusOK)
}
