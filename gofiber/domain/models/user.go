package models

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID         uuid.UUID  `gorm:"primaryKey;type:uuid;default:gen_random_uuid()"`
	Email      string     `gorm:"uniqueIndex;not null"`
	Username   string     `gorm:"uniqueIndex;not null"`
	Password   string     // Optional for OAuth users
	FirstName  string
	LastName   string
	Avatar     string
	Role       string     `gorm:"default:'user'"`
	IsActive   bool       `gorm:"default:true"`
	Provider   string     `gorm:"default:'local'"` // local, google
	ProviderID string     // OAuth provider's user ID
	LastLogin  *time.Time

	// Public access
	PublicSlug string `gorm:"uniqueIndex"` // URL slug for public access (e.g., /p/john-doe)
	IsPublic   bool   `gorm:"default:false"`

	// Google Drive integration
	DriveAccessToken  string     // Google Drive access token (encrypted)
	DriveRefreshToken string     // Google Drive refresh token (encrypted)
	DriveTokenExpiry  *time.Time // Token expiry time
	DriveRootFolderID   string // Root folder ID to sync from
	DriveRootFolderName string // Root folder name (for path-based queries)
	DriveWebhookToken   string // Token for webhook verification
	DrivePageToken    string     // Start page token for change tracking

	// Gemini AI integration
	GeminiAPIKey string `gorm:"column:gemini_api_key"` // User's Gemini API key
	GeminiModel  string `gorm:"column:gemini_model"`   // Gemini model to use (e.g., gemini-2.0-flash)

	CreatedAt time.Time
	UpdatedAt time.Time

	// Relations
	Photos  []Photo  `gorm:"foreignKey:UserID"`
	Persons []Person `gorm:"foreignKey:UserID"`
	News    []News   `gorm:"foreignKey:UserID"`
}

func (User) TableName() string {
	return "users"
}