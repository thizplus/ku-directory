package handlers

import (
	"strconv"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gofiber-template/domain/dto"
	"gofiber-template/domain/services"
	"gofiber-template/pkg/utils"
)

type JobHandler struct {
	jobService services.JobService
}

func NewJobHandler(jobService services.JobService) *JobHandler {
	return &JobHandler{
		jobService: jobService,
	}
}

func (h *JobHandler) CreateJob(c *fiber.Ctx) error {
	var req dto.CreateJobRequest
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

	job, err := h.jobService.CreateJob(c.Context(), &req)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Job creation failed", err)
	}

	jobResponse := dto.JobToJobResponse(job)
	return utils.SuccessResponse(c, "Job created successfully", jobResponse)
}

func (h *JobHandler) GetJob(c *fiber.Ctx) error {
	jobIDStr := c.Params("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid job ID")
	}

	job, err := h.jobService.GetJob(c.Context(), jobID)
	if err != nil {
		return utils.NotFoundResponse(c, "Job not found")
	}

	jobResponse := dto.JobToJobResponse(job)
	return utils.SuccessResponse(c, "Job retrieved successfully", jobResponse)
}

func (h *JobHandler) UpdateJob(c *fiber.Ctx) error {
	jobIDStr := c.Params("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid job ID")
	}

	var req dto.UpdateJobRequest
	if err := c.BodyParser(&req); err != nil {
		return utils.ValidationErrorResponse(c, "Invalid request body")
	}

	job, err := h.jobService.UpdateJob(c.Context(), jobID, &req)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Job update failed", err)
	}

	jobResponse := dto.JobToJobResponse(job)
	return utils.SuccessResponse(c, "Job updated successfully", jobResponse)
}

func (h *JobHandler) DeleteJob(c *fiber.Ctx) error {
	jobIDStr := c.Params("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid job ID")
	}

	err = h.jobService.DeleteJob(c.Context(), jobID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Job deletion failed", err)
	}

	return utils.SuccessResponse(c, "Job deleted successfully", nil)
}

func (h *JobHandler) StartJob(c *fiber.Ctx) error {
	jobIDStr := c.Params("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid job ID")
	}

	err = h.jobService.StartJob(c.Context(), jobID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Failed to start job", err)
	}

	return utils.SuccessResponse(c, "Job started successfully", nil)
}

func (h *JobHandler) StopJob(c *fiber.Ctx) error {
	jobIDStr := c.Params("id")
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		return utils.ValidationErrorResponse(c, "Invalid job ID")
	}

	err = h.jobService.StopJob(c.Context(), jobID)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusBadRequest, "Failed to stop job", err)
	}

	return utils.SuccessResponse(c, "Job stopped successfully", nil)
}

func (h *JobHandler) ListJobs(c *fiber.Ctx) error {
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

	jobs, total, err := h.jobService.ListJobs(c.Context(), offset, limit)
	if err != nil {
		return utils.ErrorResponse(c, fiber.StatusInternalServerError, "Failed to retrieve jobs", err)
	}

	jobResponses := make([]dto.JobResponse, len(jobs))
	for i, job := range jobs {
		jobResponses[i] = *dto.JobToJobResponse(job)
	}

	response := &dto.JobListResponse{
		Jobs: jobResponses,
		Meta: dto.PaginationMeta{
			Total:  total,
			Offset: offset,
			Limit:  limit,
		},
	}

	return utils.SuccessResponse(c, "Jobs retrieved successfully", response)
}