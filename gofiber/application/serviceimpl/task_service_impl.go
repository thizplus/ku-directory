package serviceimpl

import (
	"context"
	"errors"
	"gofiber-template/domain/dto"
	"gofiber-template/domain/models"
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"time"

	"github.com/google/uuid"
)

type TaskServiceImpl struct {
	taskRepo repositories.TaskRepository
	userRepo repositories.UserRepository
}

func NewTaskService(taskRepo repositories.TaskRepository, userRepo repositories.UserRepository) services.TaskService {
	return &TaskServiceImpl{
		taskRepo: taskRepo,
		userRepo: userRepo,
	}
}

func (s *TaskServiceImpl) CreateTask(ctx context.Context, userID uuid.UUID, req *dto.CreateTaskRequest) (*models.Task, error) {
	_, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	task := &models.Task{
		ID:          uuid.New(),
		Title:       req.Title,
		Description: req.Description,
		Status:      "pending",
		Priority:    req.Priority,
		DueDate:     req.DueDate,
		UserID:      userID,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if task.Priority == 0 {
		task.Priority = 1
	}

	err = s.taskRepo.Create(ctx, task)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskServiceImpl) GetTask(ctx context.Context, taskID uuid.UUID) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, errors.New("task not found")
	}
	return task, nil
}

func (s *TaskServiceImpl) GetUserTasks(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*models.Task, int64, error) {
	tasks, err := s.taskRepo.GetByUserID(ctx, userID, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.taskRepo.CountByUserID(ctx, userID)
	if err != nil {
		return nil, 0, err
	}

	return tasks, count, nil
}

func (s *TaskServiceImpl) UpdateTask(ctx context.Context, taskID uuid.UUID, req *dto.UpdateTaskRequest) (*models.Task, error) {
	task, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, errors.New("task not found")
	}

	if req.Title != "" {
		task.Title = req.Title
	}
	if req.Description != "" {
		task.Description = req.Description
	}
	if req.Status != "" {
		task.Status = req.Status
	}
	if req.Priority > 0 {
		task.Priority = req.Priority
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}

	task.UpdatedAt = time.Now()

	err = s.taskRepo.Update(ctx, taskID, task)
	if err != nil {
		return nil, err
	}

	return task, nil
}

func (s *TaskServiceImpl) DeleteTask(ctx context.Context, taskID uuid.UUID) error {
	_, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return errors.New("task not found")
	}

	return s.taskRepo.Delete(ctx, taskID)
}

func (s *TaskServiceImpl) ListTasks(ctx context.Context, offset, limit int) ([]*models.Task, int64, error) {
	tasks, err := s.taskRepo.List(ctx, offset, limit)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.taskRepo.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return tasks, count, nil
}
