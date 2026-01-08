package main

import (
	"fmt"
	"html/template"
	"os"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v2"
	"github.com/yokeTH/gofiber-scalar/scalar/v2"
	_ "gofiber-template/docs"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/interfaces/api/middleware"
	"gofiber-template/interfaces/api/routes"
	"gofiber-template/pkg/di"
	"gofiber-template/pkg/logger"
)

// @title KU Directory API
// @version 1.0
// @description ระบบจัดการรูปภาพและโฟลเดอร์ Google Drive สำหรับมหาวิทยาลัยเกษตรศาสตร์
// @description
// @description ## ความสามารถหลัก
// @description - **การจัดการโฟลเดอร์** - เชื่อมต่อและซิงค์กับ Google Drive
// @description - **การค้นหาใบหน้า** - ค้นหารูปภาพจากใบหน้าด้วย AI
// @description - **การสร้างข่าว** - สร้างข่าวอัตโนมัติจากรูปภาพด้วย Gemini AI
// @description - **การจัดการผู้ใช้** - ระบบสมาชิกและสิทธิ์การเข้าถึง
// @description
// @description ## การยืนยันตัวตน
// @description ใช้ JWT Token ในการยืนยันตัวตน โดยส่ง Token ใน Header `Authorization: Bearer <token>`

// @contact.name ทีมพัฒนา KU Directory
// @contact.email support@ku-directory.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description ใส่ "Bearer" ตามด้วยช่องว่างและ JWT Token เช่น "Bearer eyJhbGc..."

// @securityDefinitions.apikey AdminToken
// @in header
// @name X-Admin-Token
// @description Token สำหรับผู้ดูแลระบบ ใช้เข้าถึง Log และฟังก์ชันพิเศษ

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
	app.Use(middleware.RateLimiter(&container.GetConfig().RateLimit))

	// Log rate limit config
	if container.GetConfig().RateLimit.Enabled {
		logger.Startup("rate_limit_enabled", "Rate limiting enabled", map[string]interface{}{
			"max_requests":        container.GetConfig().RateLimit.MaxRequests,
			"window_seconds":      container.GetConfig().RateLimit.WindowSeconds,
			"auth_max_requests":   container.GetConfig().RateLimit.AuthMaxRequests,
			"auth_window_seconds": container.GetConfig().RateLimit.AuthWindowSeconds,
		})
	} else {
		logger.StartupWarn("rate_limit_disabled", "Rate limiting is disabled", nil)
	}

	// Create handlers from services
	services := container.GetHandlerServices()
	repos := container.GetHandlerRepositories()
	h := handlers.NewHandlers(services, repos, container.GetConfig())

	// Create health handler (for detailed health check)
	healthHandler := handlers.NewHealthHandler(
		container.DB,
		container.RedisClient,
		container.FaceClient,
		container.PhotoRepository,
	)

	// Setup routes
	routes.SetupRoutes(app, h, healthHandler, container.GetConfig())

	// Setup Scalar API Documentation (modern alternative to Swagger UI)
	// Custom CSS for Google Fonts (Google Sans + Roboto)
	customCSS := template.CSS(`
		@import url('https://fonts.googleapis.com/css2?family=Google+Sans:wght@400;500;700&display=swap');
		@import url('https://fonts.googleapis.com/css2?family=Roboto:wght@300;400;500;700&display=swap');
		@import url('https://fonts.googleapis.com/css2?family=Roboto+Mono:wght@400;500&display=swap');
		:root {
			--scalar-font: 'Roboto', 'Google Sans', -apple-system, BlinkMacSystemFont, sans-serif;
			--scalar-font-code: 'Roboto Mono', monospace;
		}
	`)
	app.Get("/docs/*", scalar.New(scalar.Config{
		Title:       "KU Directory API",
		Theme:       scalar.ThemeDeepSpace,
		CustomStyle: customCSS,
	}))

	// Start server
	port := container.GetConfig().App.Port
	logger.Startup("server_starting", "Server starting", map[string]interface{}{
		"port":        port,
		"environment": container.GetConfig().App.Env,
		"health":      fmt.Sprintf("http://localhost:%s/health", port),
		"api":         fmt.Sprintf("http://localhost:%s/api/v1", port),
		"docs":        fmt.Sprintf("http://localhost:%s/docs", port),
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