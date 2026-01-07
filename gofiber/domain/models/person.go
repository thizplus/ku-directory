package models

import (
	"time"

	"github.com/google/uuid"
)

type Person struct {
	ID     uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	// Person info
	Name         string `gorm:"not null"`
	ThumbnailURL string // URL to a representative face thumbnail

	// Stats (cached)
	FaceCount int `gorm:"default:0"` // Number of faces tagged as this person

	CreatedAt time.Time
	UpdatedAt time.Time

	// Relations
	User  User   `gorm:"foreignKey:UserID"`
	Faces []Face `gorm:"foreignKey:PersonID"`
}

func (Person) TableName() string {
	return "persons"
}
