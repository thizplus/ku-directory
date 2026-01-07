package dto

import (
	"time"
	"github.com/google/uuid"
)

type CreateTaskRequest struct {
	Title       string     `json:"title" validate:"required,min=1,max=200"`
	Description string     `json:"description" validate:"omitempty,max=1000"`
	Priority    int        `json:"priority" validate:"omitempty,min=1,max=5"`
	DueDate     *time.Time `json:"dueDate" validate:"omitempty"`
}

type UpdateTaskRequest struct {
	Title       string     `json:"title" validate:"omitempty,min=1,max=200"`
	Description string     `json:"description" validate:"omitempty,max=1000"`
	Status      string     `json:"status" validate:"omitempty,oneof=pending in_progress completed cancelled"`
	Priority    int        `json:"priority" validate:"omitempty,min=1,max=5"`
	DueDate     *time.Time `json:"dueDate" validate:"omitempty"`
}

type TaskResponse struct {
	ID          uuid.UUID    `json:"id"`
	Title       string       `json:"title"`
	Description string       `json:"description"`
	Status      string       `json:"status"`
	Priority    int          `json:"priority"`
	DueDate     *time.Time   `json:"dueDate"`
	UserID      uuid.UUID    `json:"userId"`
	User        UserResponse `json:"user"`
	CreatedAt   time.Time    `json:"createdAt"`
	UpdatedAt   time.Time    `json:"updatedAt"`
}

type TaskListResponse struct {
	Tasks []TaskResponse `json:"tasks"`
	Meta  PaginationMeta `json:"meta"`
}

type TaskFilterRequest struct {
	Status   string `query:"status" validate:"omitempty,oneof=pending in_progress completed cancelled"`
	Priority int    `query:"priority" validate:"omitempty,min=1,max=5"`
	UserID   string `query:"userId" validate:"omitempty,uuid"`
	Limit    int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Offset   int    `query:"offset" validate:"omitempty,min=0"`
}