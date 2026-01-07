package services

import (
	"context"
	"gofiber-template/domain/dto"
	"gofiber-template/domain/models"
	"github.com/google/uuid"
)

type UserService interface {
	Register(ctx context.Context, req *dto.CreateUserRequest) (*models.User, error)
	Login(ctx context.Context, req *dto.LoginRequest) (string, *models.User, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.User, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req *dto.UpdateUserRequest) (*models.User, error)
	UpdateGeminiSettings(ctx context.Context, userID uuid.UUID, req *dto.GeminiSettingsRequest) (*models.User, error)
	DeleteUser(ctx context.Context, userID uuid.UUID) error
	ListUsers(ctx context.Context, offset, limit int) ([]*models.User, int64, error)
	GenerateJWT(user *models.User) (string, error)
	ValidateJWT(token string) (*models.User, error)
}