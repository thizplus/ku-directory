package dto

import (
	"time"
	"github.com/google/uuid"
)

type CreateJobRequest struct {
	Name     string `json:"name" validate:"required,min=1,max=100"`
	CronExpr string `json:"cronExpr" validate:"required,min=5,max=50"`
	Payload  string `json:"payload" validate:"omitempty,json"`
}

type UpdateJobRequest struct {
	Name     string `json:"name" validate:"omitempty,min=1,max=100"`
	CronExpr string `json:"cronExpr" validate:"omitempty,min=5,max=50"`
	Payload  string `json:"payload" validate:"omitempty,json"`
	IsActive bool   `json:"isActive"`
}

type JobResponse struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	CronExpr  string     `json:"cronExpr"`
	Payload   string     `json:"payload"`
	Status    string     `json:"status"`
	LastRun   *time.Time `json:"lastRun"`
	NextRun   *time.Time `json:"nextRun"`
	IsActive  bool       `json:"isActive"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

type JobListResponse struct {
	Jobs []JobResponse  `json:"jobs"`
	Meta PaginationMeta `json:"meta"`
}

type JobFilterRequest struct {
	Status   string `query:"status" validate:"omitempty,oneof=active inactive running completed failed"`
	IsActive *bool  `query:"isActive"`
	Limit    int    `query:"limit" validate:"omitempty,min=1,max=100"`
	Offset   int    `query:"offset" validate:"omitempty,min=0"`
}

type JobExecutionRequest struct {
	JobID uuid.UUID `json:"jobId" validate:"required"`
}

type JobExecutionResponse struct {
	JobID     uuid.UUID `json:"jobId"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	ExecutedAt time.Time `json:"executedAt"`
}