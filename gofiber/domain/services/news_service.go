package services

import (
	"context"

	"github.com/google/uuid"

	"gofiber-template/infrastructure/gemini"
)

// NewsGenerateRequest contains the request parameters for news generation
type NewsGenerateRequest struct {
	PhotoIDs []uuid.UUID `json:"photo_ids"`
	Headings []string    `json:"headings"` // 4 custom headings
	Tone     string      `json:"tone"`     // formal, friendly, news
	Length   string      `json:"length"`   // short, medium, long
}

// NewsService handles news generation operations
type NewsService interface {
	// GenerateNews generates a news article from selected photos
	GenerateNews(ctx context.Context, userID uuid.UUID, req *NewsGenerateRequest) (*gemini.NewsArticle, error)
}
