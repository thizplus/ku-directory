package dto

import (
	"time"
	"github.com/google/uuid"
)

type UploadFileRequest struct {
	// Path-related fields (from form data)
	CustomPath string `json:"customPath" validate:"omitempty,min=1,max=500"`
	Category   string `json:"category" validate:"omitempty,min=1,max=50"`
	EntityID   string `json:"entityId" validate:"omitempty,uuid"`
	FileType   string `json:"fileType" validate:"omitempty,min=1,max=50"`
}

type FileResponse struct {
	ID        uuid.UUID `json:"id"`
	FileName  string    `json:"fileName"`
	FileSize  int64     `json:"fileSize"`
	MimeType  string    `json:"mimeType"`
	URL       string    `json:"url"`
	CDNPath   string    `json:"cdnPath"`
	UserID    uuid.UUID `json:"userId"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type FileListResponse struct {
	Files []FileResponse `json:"files"`
	Meta  PaginationMeta `json:"meta"`
}

type UploadResponse struct {
	FileID   uuid.UUID `json:"fileId"`
	FileName string    `json:"fileName"`
	URL      string    `json:"url"`
	CDNPath  string    `json:"cdnPath"`
	FileSize int64     `json:"fileSize"`
	MimeType string    `json:"mimeType"`
	PathType string    `json:"pathType"` // "custom" or "structured"
}

type FileFilterRequest struct {
	MimeType string `query:"mimeType" validate:"omitempty,min=1,max=100"`
	UserID   string `query:"userId" validate:"omitempty,uuid"`
	Limit    int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Offset   int    `query:"offset" validate:"omitempty,min=0"`
}

type DeleteFileRequest struct {
	FileID uuid.UUID `json:"fileId" validate:"required" param:"id"`
}

type DeleteFileResponse struct {
	Message string `json:"message"`
	FileID  uuid.UUID `json:"fileId"`
}