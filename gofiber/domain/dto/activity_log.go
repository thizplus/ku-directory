package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"gofiber-template/domain/models"
)

// ActivityLogResponse represents an activity log entry
type ActivityLogResponse struct {
	ID             uuid.UUID `json:"id"`
	SharedFolderID uuid.UUID `json:"sharedFolderId"`
	ActivityType   string    `json:"activityType"`
	Message        string    `json:"message"`
	Details        any       `json:"details,omitempty"`
	RawData        any       `json:"rawData,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

// ActivityLogListRequest represents a request to list activity logs
type ActivityLogListRequest struct {
	FolderID     uuid.UUID `query:"folderId" validate:"required"`
	ActivityType string    `query:"activityType"`
	Page         int       `query:"page"`
	Limit        int       `query:"limit"`
}

// ActivityLogToResponse converts a model to response DTO
func ActivityLogToResponse(log *models.ActivityLog) *ActivityLogResponse {
	resp := &ActivityLogResponse{
		ID:             log.ID,
		SharedFolderID: log.SharedFolderID,
		ActivityType:   string(log.ActivityType),
		Message:        log.Message,
		CreatedAt:      log.CreatedAt,
	}

	// Parse JSON details if present
	if log.Details != "" {
		var details map[string]interface{}
		if err := json.Unmarshal([]byte(log.Details), &details); err == nil {
			resp.Details = details
		}
	}

	// Parse JSON raw data if present
	if log.RawData != "" {
		var rawData map[string]interface{}
		if err := json.Unmarshal([]byte(log.RawData), &rawData); err == nil {
			resp.RawData = rawData
		}
	}

	return resp
}

// ActivityLogsToResponse converts a slice of models to response DTOs
func ActivityLogsToResponse(logs []models.ActivityLog) []*ActivityLogResponse {
	result := make([]*ActivityLogResponse, len(logs))
	for i, log := range logs {
		result[i] = ActivityLogToResponse(&log)
	}
	return result
}
