package dto

import (
	"time"

	"github.com/google/uuid"
)

// PhotoResponse is the DTO for photo API responses
type PhotoResponse struct {
	ID              uuid.UUID `json:"id"`
	SharedFolderID  uuid.UUID `json:"shared_folder_id"`
	DriveFileID     string    `json:"drive_file_id"`
	FileName        string    `json:"file_name"`
	MimeType        string    `json:"mime_type"`
	ThumbnailURL    string    `json:"thumbnail_url"`
	WebViewURL      string    `json:"web_view_url"`
	DriveFolderPath string    `json:"drive_folder_path"`
	FaceStatus      string    `json:"face_status"`
	FaceCount       int       `json:"face_count"`
	CreatedAt       time.Time `json:"created_at"`
}

// PhotoListResponse is the DTO for paginated photo list
type PhotoListResponse struct {
	Photos []PhotoResponse `json:"photos"`
	Total  int64           `json:"total"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
}
