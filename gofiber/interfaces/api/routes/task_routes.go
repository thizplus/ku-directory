package routes

import (
	"github.com/gofiber/fiber/v2"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
)

func SetupTaskRoutes(api fiber.Router, h *handlers.Handlers) {
	tasks := api.Group("/tasks")
	tasks.Use(middleware.Protected())
	tasks.Post("/", h.TaskHandler.CreateTask)
	tasks.Get("/", middleware.AdminOnly(), h.TaskHandler.ListTasks)
	tasks.Get("/my", h.TaskHandler.GetUserTasks)
	tasks.Get("/:id", h.TaskHandler.GetTask)
	tasks.Put("/:id", middleware.OwnerOnly(), h.TaskHandler.UpdateTask)
	tasks.Delete("/:id", middleware.OwnerOnly(), h.TaskHandler.DeleteTask)
}