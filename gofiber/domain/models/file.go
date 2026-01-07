package models

import (
	"time"
	"github.com/google/uuid"
)

type File struct {
	ID        uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	FileName  string    `gorm:"not null"`
	FileSize  int64
	MimeType  string
	URL       string    `gorm:"not null"`
	CDNPath   string
	UserID    uuid.UUID `gorm:"not null"`
	User      User      `gorm:"foreignKey:UserID"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (File) TableName() string {
	return "files"
}