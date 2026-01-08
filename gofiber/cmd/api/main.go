package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	swagger "github.com/swaggo/fiber-swagger"
	"gofiber-template/docs"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
	"gofiber-template/interfaces/api/routes"
	"gofiber-template/pkg/di"
	"gofiber-template/pkg/logger"
)

// @title KU Directory API
// @version 1.0
// @description API สำหรับระบบจัดการรูปภาพและโฟลเดอร์ Google Drive
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@example.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

// @securityDefinitions.apikey AdminToken
// @in header
// @name X-Admin-Token
// @description Admin token for log access

func main() {
	// Initialize logger
	if err := logger.Init("logs", true); err != nil {
		fmt.Printf("Warning: Failed to initialize logger: %v\n", err)
	}
	logger.Startup("logger_init", "Logger initialized - logs will be written to ./logs/", nil)

	// Initialize DI container
	container := di.NewContainer()

	// Initialize all dependencies
	if err := container.Initialize(); err != nil {
		logger.StartupError("container_init_failed", "Failed to initialize container", err, nil)
		os.Exit(1)
	}

	// Setup graceful shutdown
	setupGracefulShutdown(container)

	// Create Fiber app
	app := fiber.New(fiber.Config{
		ErrorHandler: middleware.ErrorHandler(),
		AppName:      container.GetConfig().App.Name,
	})

	// Setup middleware
	app.Use(middleware.LoggerMiddleware())
	app.Use(middleware.CorsMiddleware())

	// Create handlers from services
	services := container.GetHandlerServices()
	repos := container.GetHandlerRepositories()
	h := handlers.NewHandlers(services, repos, container.GetConfig())

	// Setup routes
	routes.SetupRoutes(app, h)

	// Setup Swagger - use empty host so it works on any domain
	docs.SwaggerInfo.Host = ""
	app.Get("/swagger/*", swagger.WrapHandler)

	// Start server
	port := container.GetConfig().App.Port
	logger.Startup("server_starting", "Server starting", map[string]interface{}{
		"port":        port,
		"environment": container.GetConfig().App.Env,
		"health":      fmt.Sprintf("http://localhost:%s/health", port),
		"api":         fmt.Sprintf("http://localhost:%s/api/v1", port),
		"swagger":     fmt.Sprintf("http://localhost:%s/swagger/index.html", port),
		"websocket":   fmt.Sprintf("ws://localhost:%s/ws", port),
		"logs_api":    fmt.Sprintf("http://localhost:%s/api/v1/admin/logs", port),
	})

	if err := app.Listen(":" + port); err != nil {
		logger.StartupError("server_failed", "Server failed to start", err, nil)
		os.Exit(1)
	}
}

func setupGracefulShutdown(container *di.Container) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		logger.Startup("shutdown_started", "Gracefully shutting down", nil)

		if err := container.Cleanup(); err != nil {
			logger.StartupError("cleanup_failed", "Error during cleanup", err, nil)
		}

		logger.Startup("shutdown_complete", "Shutdown complete", nil)
		os.Exit(0)
	}()
}