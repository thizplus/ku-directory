package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/pgvector/pgvector-go"
)

type Face struct {
	ID             uuid.UUID  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	SharedFolderID uuid.UUID  `gorm:"type:uuid;not null;index"` // For faster queries
	PhotoID        uuid.UUID  `gorm:"type:uuid;not null;index"`
	UserID         *uuid.UUID `gorm:"type:uuid;index"` // Deprecated: kept for migration

	// Face embedding vector (512 dimensions for InsightFace)
	Embedding pgvector.Vector `gorm:"type:vector(512);not null"`

	// Bounding box (x, y, width, height as percentage of image)
	BboxX      float64 `gorm:"not null"`
	BboxY      float64 `gorm:"not null"`
	BboxWidth  float64 `gorm:"not null"`
	BboxHeight float64 `gorm:"not null"`

	// Detection confidence (0-1)
	Confidence float64 `gorm:"not null"`

	// Person identification (optional - set when user tags the face)
	PersonID *uuid.UUID `gorm:"type:uuid;index"`

	CreatedAt time.Time
	UpdatedAt time.Time

	// Relations
	SharedFolder SharedFolder `gorm:"foreignKey:SharedFolderID"`
	Photo        Photo        `gorm:"foreignKey:PhotoID"`
	User         *User        `gorm:"foreignKey:UserID"` // Deprecated
	Person       *Person      `gorm:"foreignKey:PersonID"`
}

func (Face) TableName() string {
	return "faces"
}
