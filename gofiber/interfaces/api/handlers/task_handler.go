package handlers

import (
	"strconv"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gofiber-template/domain/dto"
	"gofiber-template/domain/services"
	"gofiber-template/pkg/utils"
)

type TaskHandler struct {
	taskService services.TaskService
}

func NewTaskHandler(taskService services.TaskService) *TaskHandler {
	return &TaskHandler{
		taskService: taskService,
	}
}

func (h *TaskHandler) CreateTask(c *fiber.Ctx) error {
	user, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "User not authenticated")
	}

	var req dto.CreateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ValidationErrorResponse(c, "Invalid request body")
	}

	if err := utils.ValidateStruct(&req); err != nil {
		errors := utils.GetValidationErrors(err)
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"success": false,
			"message": "Validation failed",
			"errors":  errors,
		})
	}

	task, err := h.taskService.CreateTask(c.Context(), user.ID, &req)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Task creation failed", err)
	}

	taskResponse := dto.TaskToTaskResponse(task, nil)
	return utils.SuccessResponse(c, "Task created successfully", taskResponse)
}

func (h *TaskHandler) GetTask(c *fiber.Ctx) error {
	taskIDStr := c.Params("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid task ID")
	}

	task, err := h.taskService.GetTask(c.Context(), taskID)
	if err != nil {
		return utils.NotFoundResponse(c, "Task not found")
	}

	taskResponse := dto.TaskToTaskResponse(task, &task.User)
	return utils.SuccessResponse(c, "Task retrieved successfully", taskResponse)
}

func (h *TaskHandler) UpdateTask(c *fiber.Ctx) error {
	taskIDStr := c.Params("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid task ID")
	}

	var req dto.UpdateTaskRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ValidationErrorResponse(c, "Invalid request body")
	}

	task, err := h.taskService.UpdateTask(c.Context(), taskID, &req)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Task update failed", err)
	}

	taskResponse := dto.TaskToTaskResponse(task, &task.User)
	return utils.SuccessResponse(c, "Task updated successfully", taskResponse)
}

func (h *TaskHandler) DeleteTask(c *fiber.Ctx) error {
	taskIDStr := c.Params("id")
	taskID, err := uuid.Parse(taskIDStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid task ID")
	}

	err = h.taskService.DeleteTask(c.Context(), taskID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Task deletion failed", err)
	}

	return utils.SuccessResponse(c, "Task deleted successfully", nil)
}

func (h *TaskHandler) GetUserTasks(c *fiber.Ctx) error {
	user, err := utils.GetUserFromContext(c)
	if err != nil {
		return utils.UnauthorizedResponse(c, "User not authenticated")
	}

	offsetStr := c.Query("offset", "0")
	limitStr := c.Query("limit", "10")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid offset parameter")
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid limit parameter")
	}

	tasks, total, err := h.taskService.GetUserTasks(c.Context(), user.ID, offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to retrieve tasks", err)
	}

	taskResponses := make([]dto.TaskResponse, len(tasks))
	for i, task := range tasks {
		taskResponses[i] = *dto.TaskToTaskResponse(task, &task.User)
	}

	response := &dto.TaskListResponse{
		Tasks: taskResponses,
		Meta: dto.PaginationMeta{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return utils.SuccessResponse(c, "Tasks retrieved successfully", response)
}

func (h *TaskHandler) ListTasks(c *fiber.Ctx) error {
	offsetStr := c.Query("offset", "0")
	limitStr := c.Query("limit", "10")

	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid offset parameter")
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid limit parameter")
	}

	tasks, total, err := h.taskService.ListTasks(c.Context(), offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to retrieve tasks", err)
	}

	taskResponses := make([]dto.TaskResponse, len(tasks))
	for i, task := range tasks {
		taskResponses[i] = *dto.TaskToTaskResponse(task, &task.User)
	}

	response := &dto.TaskListResponse{
		Tasks: taskResponses,
		Meta: dto.PaginationMeta{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return utils.SuccessResponse(c, "Tasks retrieved successfully", response)
}