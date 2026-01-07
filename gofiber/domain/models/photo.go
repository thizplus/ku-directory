package models

import (
	"time"

	"github.com/google/uuid"
)

type FaceProcessingStatus string

const (
	FaceStatusPending    FaceProcessingStatus = "pending"
	FaceStatusProcessing FaceProcessingStatus = "processing"
	FaceStatusCompleted  FaceProcessingStatus = "completed"
	FaceStatusFailed     FaceProcessingStatus = "failed"
)

type Photo struct {
	ID             uuid.UUID  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SharedFolderID uuid.UUID  `gorm:"type:uuid;not null;index"` // Reference to shared folder
	UserID         *uuid.UUID `gorm:"type:uuid;index"`          // Deprecated: kept for migration, nullable

	// Google Drive metadata
	DriveFileID     string `gorm:"uniqueIndex;not null"` // Google Drive file ID
	DriveFolderID   string `gorm:"index"`                // Parent folder ID in Drive
	DriveFolderPath string                               // Full folder path (e.g., "กิจกรรม/รับน้อง 2567")

	// File info (cached from Drive)
	FileName      string
	MimeType      string
	FileSize      int64
	Width         int
	Height        int
	ThumbnailURL  string // Google Drive thumbnail URL
	WebViewURL    string // Google Drive web view URL

	// Timestamps from Drive
	DriveCreatedAt  *time.Time // Original creation time in Drive
	DriveModifiedAt *time.Time // Last modified time in Drive

	// Face processing
	FaceStatus     FaceProcessingStatus `gorm:"default:'pending';index"`
	FaceCount      int                  `gorm:"default:0"` // Number of faces detected
	FaceProcessedAt *time.Time

	CreatedAt time.Time
	UpdatedAt time.Time

	// Relations
	SharedFolder SharedFolder `gorm:"foreignKey:SharedFolderID"`
	User         *User        `gorm:"foreignKey:UserID"` // Deprecated
	Faces        []Face       `gorm:"foreignKey:PhotoID"`
}

func (Photo) TableName() string {
	return "photos"
}
