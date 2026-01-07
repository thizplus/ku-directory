package serviceimpl

import (
	"context"
	"errors"
	"fmt"
	"gofiber-template/domain/dto"
	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"gofiber-template/infrastructure/storage"
	"gofiber-template/pkg/utils"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

type FileServiceImpl struct {
	fileRepo repositories.FileRepository
	userRepo repositories.UserRepository
	storage  storage.BunnyStorage
}

func NewFileService(fileRepo repositories.FileRepository, userRepo repositories.UserRepository, storage storage.BunnyStorage) services.FileService {
	return &FileServiceImpl{
		fileRepo: fileRepo,
		userRepo: userRepo,
		storage:  storage,
	}
}

func (s *FileServiceImpl) UploadFile(ctx context.Context, userID uuid.UUID, fileHeader *multipart.FileHeader, options *dto.UploadFileRequest) (*models.File, error) {
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Sanitize the filename
	sanitizedFileName := utils.SanitizeFileName(fileHeader.Filename)
	fileExt := filepath.Ext(sanitizedFileName)
	uniqueFileName := fmt.Sprintf("%s%s", uuid.New().String(), fileExt)

	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = s.getMimeTypeFromExtension(fileExt)
	}

	// Determine the path based on whether custom path is provided
	var cdnPath string
	if options != nil && options.CustomPath != "" {
		// Use custom path approach
		validatedPath, err := utils.ValidateAndSanitizePath(options.CustomPath)
		if err != nil {
			return nil, fmt.Errorf("invalid custom path: %w", err)
		}
		cdnPath = filepath.Join(validatedPath, uniqueFileName)
	} else {
		// Use structured path approach
		category := ""
		entityID := ""
		fileType := ""

		if options != nil {
			category = options.Category
			entityID = options.EntityID
			fileType = options.FileType
		}

		structuredPath := utils.GenerateStructuredPath(userID.String(), category, entityID, fileType)
		cdnPath = filepath.Join(structuredPath, uniqueFileName)
	}

	// Normalize path separators for storage
	cdnPath = strings.ReplaceAll(cdnPath, "\\", "/")

	url, err := s.storage.UploadFile(file, cdnPath, mimeType)
	if err != nil {
		return nil, err
	}

	fileModel := &models.File{
		ID:        uuid.New(),
		FileName:  sanitizedFileName,
		FileSize:  fileHeader.Size,
		MimeType:  mimeType,
		URL:       url,
		CDNPath:   cdnPath,
		UserID:    userID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	err = s.fileRepo.Create(ctx, fileModel)
	if err != nil {
		s.storage.DeleteFile(cdnPath)
		return nil, err
	}

	return fileModel, nil
}

func (s *FileServiceImpl) GetFile(ctx context.Context, fileID uuid.UUID) (*models.File, error) {
	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return nil, errors.New("file not found")
	}
	return file, nil
}

func (s *FileServiceImpl) GetUserFiles(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*models.File, int64, error) {
	files, err := s.fileRepo.GetByUserID(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.fileRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return files, count, nil
}

func (s *FileServiceImpl) DeleteFile(ctx context.Context, fileID uuid.UUID) error {
	file, err := s.fileRepo.GetByID(ctx, fileID)
	if err != nil {
		return errors.New("file not found")
	}

	err = s.storage.DeleteFile(file.CDNPath)
	if err != nil {
		return err
	}

	return s.fileRepo.Delete(ctx, fileID)
}

func (s *FileServiceImpl) ListFiles(ctx context.Context, offset, limit int) ([]*models.File, int64, error) {
	files, err := s.fileRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.fileRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return files, count, nil
}

func (s *FileServiceImpl) getMimeTypeFromExtension(ext string) string {
	ext = strings.ToLower(ext)
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".pdf":  "application/pdf",
		".txt":  "text/plain",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".zip":  "application/zip",
	}

	if mimeType, exists := mimeTypes[ext]; exists {
		return mimeType
	}
	return "application/octet-stream"
}
