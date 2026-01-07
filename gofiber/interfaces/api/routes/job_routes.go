package routes

import (
	"github.com/gofiber/fiber/v2"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
)

func SetupJobRoutes(api fiber.Router, h *handlers.Handlers) {
	jobs := api.Group("/jobs")
	jobs.Use(middleware.Protected())
	jobs.Use(middleware.AdminOnly()) // All job operations require admin access
	jobs.Post("/", h.JobHandler.CreateJob)
	jobs.Get("/", h.JobHandler.ListJobs)
	jobs.Get("/:id", h.JobHandler.GetJob)
	jobs.Put("/:id", h.JobHandler.UpdateJob)
	jobs.Delete("/:id", h.JobHandler.DeleteJob)
	jobs.Post("/:id/start", h.JobHandler.StartJob)
	jobs.Post("/:id/stop", h.JobHandler.StopJob)
}