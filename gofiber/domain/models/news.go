package models

import (
	"time"

	"github.com/google/uuid"
)

type NewsStatus string

const (
	NewsStatusDraft     NewsStatus = "draft"
	NewsStatusPublished NewsStatus = "published"
	NewsStatusArchived  NewsStatus = "archived"
)

type News struct {
	ID     uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index"`

	// Content
	Title   string `gorm:"not null"`
	Content string `gorm:"type:text"` // HTML or Markdown content
	Summary string // Short summary for preview

	// AI generation info
	AIModel    string // Model used (e.g., "gemini-pro")
	AIPrompt   string `gorm:"type:text"` // Prompt used to generate

	// Status
	Status      NewsStatus `gorm:"default:'draft';index"`
	PublishedAt *time.Time

	// SEO
	Slug        string `gorm:"uniqueIndex"` // URL-friendly slug
	MetaTitle   string
	MetaDesc    string

	CreatedAt time.Time
	UpdatedAt time.Time

	// Relations
	User   User         `gorm:"foreignKey:UserID"`
	Photos []NewsPhoto  `gorm:"foreignKey:NewsID"`
}

func (News) TableName() string {
	return "news"
}

// NewsPhoto is a many-to-many relation between News and Photo
type NewsPhoto struct {
	ID      uuid.UUID `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	NewsID  uuid.UUID `gorm:"type:uuid;not null;index"`
	PhotoID uuid.UUID `gorm:"type:uuid;not null;index"`

	// Order of photo in the news article
	SortOrder int `gorm:"default:0"`

	// Optional caption for this photo in this news
	Caption string

	CreatedAt time.Time

	// Relations
	News  News  `gorm:"foreignKey:NewsID"`
	Photo Photo `gorm:"foreignKey:PhotoID"`
}

func (NewsPhoto) TableName() string {
	return "news_photos"
}
