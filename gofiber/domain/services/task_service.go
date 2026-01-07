package services

import (
	"context"
	"gofiber-template/domain/dto"
	"gofiber-template/domain/models"

	"github.com/google/uuid"
)

type TaskService interface {
	CreateTask(ctx context.Context, userID uuid.UUID, req *dto.CreateTaskRequest) (*models.Task, error)
	GetTask(ctx context.Context, taskID uuid.UUID) (*models.Task, error)
	GetUserTasks(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*models.Task, int64, error)
	UpdateTask(ctx context.Context, taskID uuid.UUID, req *dto.UpdateTaskRequest) (*models.Task, error)
	DeleteTask(ctx context.Context, taskID uuid.UUID) error
	ListTasks(ctx context.Context, offset, limit int) ([]*models.Task, int64, error)
}
