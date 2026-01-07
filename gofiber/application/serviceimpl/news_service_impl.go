package serviceimpl

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"gofiber-template/infrastructure/gemini"
	"gofiber-template/infrastructure/googledrive"
)

type NewsServiceImpl struct {
	driveClient      *googledrive.DriveClient
	photoRepo        repositories.PhotoRepository
	userRepo         repositories.UserRepository
	sharedFolderRepo repositories.SharedFolderRepository
}

func NewNewsService(
	photoRepo repositories.PhotoRepository,
	userRepo repositories.UserRepository,
	sharedFolderRepo repositories.SharedFolderRepository,
) services.NewsService {
	return &NewsServiceImpl{
		photoRepo:        photoRepo,
		userRepo:         userRepo,
		sharedFolderRepo: sharedFolderRepo,
	}
}

// NewNewsServiceWithDrive creates a news service with Google Drive support
func NewNewsServiceWithDrive(
	driveClient *googledrive.DriveClient,
	photoRepo repositories.PhotoRepository,
	userRepo repositories.UserRepository,
	sharedFolderRepo repositories.SharedFolderRepository,
) services.NewsService {
	return &NewsServiceImpl{
		driveClient:      driveClient,
		photoRepo:        photoRepo,
		userRepo:         userRepo,
		sharedFolderRepo: sharedFolderRepo,
	}
}

// GenerateNews generates a news article from selected photos or headings only
func (s *NewsServiceImpl) GenerateNews(ctx context.Context, userID uuid.UUID, req *services.NewsGenerateRequest) (*gemini.NewsArticle, error) {
	// Get user to check Gemini credentials
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Check if user has Gemini API key configured
	if user.GeminiAPIKey == "" {
		return nil, fmt.Errorf("Gemini API key not configured. Please add your API key in Settings")
	}

	// Get model (default to gemini-2.0-flash if not set)
	geminiModel := user.GeminiModel
	if geminiModel == "" {
		geminiModel = "gemini-2.0-flash"
	}

	// Create Gemini client with user's credentials
	geminiClient, err := gemini.NewGeminiClient(user.GeminiAPIKey, geminiModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	var images [][]byte
	var mimeTypes []string
	folderName := ""

	// If photos are provided, download them
	if len(req.PhotoIDs) > 0 {
		// Get photos by IDs
		photos, err := s.photoRepo.GetByIDs(ctx, req.PhotoIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get photos: %w", err)
		}

		// Verify access to photos - check if user has access to the folder
		for _, photo := range photos {
			hasAccess, err := s.sharedFolderRepo.HasUserAccess(ctx, userID, photo.SharedFolderID)
			if err != nil {
				return nil, fmt.Errorf("failed to verify access: %w", err)
			}
			if !hasAccess {
				return nil, fmt.Errorf("unauthorized access to photo")
			}
		}

		// Check if user has Drive connected for downloading images
		if user.DriveRefreshToken == "" {
			return nil, fmt.Errorf("user has not connected Google Drive")
		}

		// Get folder name from first photo's path
		if len(photos) > 0 && photos[0].DriveFolderPath != "" {
			path := photos[0].DriveFolderPath
			for i := len(path) - 1; i >= 0; i-- {
				if path[i] == '/' {
					folderName = path[i+1:]
					break
				}
			}
			if folderName == "" {
				folderName = path
			}
		}

		// Download images from Google Drive (limit to 5 images to avoid token limits)
		maxImages := 5
		if len(photos) < maxImages {
			maxImages = len(photos)
		}

		for i := 0; i < maxImages; i++ {
			photo := photos[i]
			if photo.DriveFileID == "" {
				continue
			}

			// Download thumbnail via Google Drive API
			imgData, contentType, err := s.driveClient.DownloadThumbnail(
				ctx,
				user.DriveAccessToken,
				user.DriveRefreshToken,
				time.Now().Add(-1*time.Hour), // Force token refresh
				photo.DriveFileID,
				800, // Medium resolution
			)
			if err != nil {
				// Skip failed downloads, continue with others
				fmt.Printf("Warning: Failed to download image %s: %v\n", photo.FileName, err)
				continue
			}

			images = append(images, imgData)
			mimeTypes = append(mimeTypes, contentType)
		}
	}

	// Prepare request for Gemini (works with or without images)
	geminiReq := &gemini.GenerateNewsRequest{
		FolderName: folderName,
		Images:     images,
		MimeTypes:  mimeTypes,
		Headings:   req.Headings,
		Tone:       req.Tone,
		Length:     req.Length,
	}

	// Generate news using Gemini
	article, err := geminiClient.GenerateNews(ctx, geminiReq)
	if err != nil {
		return nil, fmt.Errorf("failed to generate news: %w", err)
	}

	return article, nil
}
