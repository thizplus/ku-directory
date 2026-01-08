package di

import (
	"context"

	"gorm.io/gorm"

	"gofiber-template/application/serviceimpl"
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"gofiber-template/infrastructure/faceapi"
	"gofiber-template/infrastructure/gemini"
	"gofiber-template/infrastructure/googledrive"
	"gofiber-template/infrastructure/oauth"
	"gofiber-template/infrastructure/postgres"
	"gofiber-template/infrastructure/redis"
	"gofiber-template/infrastructure/storage"
	"gofiber-template/infrastructure/worker"
	"gofiber-template/interfaces/api/handlers"
	"gofiber-template/pkg/config"
	"gofiber-template/pkg/logger"
	"gofiber-template/pkg/scheduler"
)

type Container struct {
	// Configuration
	Config *config.Config

	// Infrastructure
	DB             *gorm.DB
	RedisClient    *redis.RedisClient
	BunnyStorage   storage.BunnyStorage
	EventScheduler scheduler.EventScheduler
	GoogleOAuth    *oauth.GoogleOAuth
	GoogleDrive    *googledrive.DriveClient

	// Repositories
	UserRepository         repositories.UserRepository
	TaskRepository         repositories.TaskRepository
	FileRepository         repositories.FileRepository
	JobRepository          repositories.JobRepository
	PhotoRepository        repositories.PhotoRepository
	SyncJobRepository      repositories.SyncJobRepository
	FaceRepository         repositories.FaceRepository
	PersonRepository       repositories.PersonRepository
	SharedFolderRepository repositories.SharedFolderRepository

	// Services
	UserService         services.UserService
	TaskService         services.TaskService
	FileService         services.FileService
	JobService          services.JobService
	AuthService         services.AuthService
	DriveService        services.DriveService
	FaceService         services.FaceService
	NewsService         services.NewsService
	SharedFolderService services.SharedFolderService

	// Workers
	SyncWorker *worker.SyncWorker
	FaceWorker *worker.FaceWorker

	// Clients
	FaceClient   *faceapi.FaceClient
	GeminiClient *gemini.GeminiClient
}

func NewContainer() *Container {
	return &Container{}
}

func (c *Container) Initialize() error {
	if err := c.initConfig(); err != nil {
		return err
	}

	if err := c.initInfrastructure(); err != nil {
		return err
	}

	if err := c.initRepositories(); err != nil {
		return err
	}

	if err := c.initServices(); err != nil {
		return err
	}

	if err := c.initScheduler(); err != nil {
		return err
	}

	if err := c.initWorkers(); err != nil {
		return err
	}

	return nil
}

func (c *Container) initConfig() error {
	cfg, err := config.LoadConfig()
	if err != nil {
		return err
	}
	c.Config = cfg
	logger.Startup("config_loaded", "Configuration loaded", nil)
	return nil
}

func (c *Container) initInfrastructure() error {
	// Initialize Database
	dbConfig := postgres.DatabaseConfig{
		Host:     c.Config.Database.Host,
		Port:     c.Config.Database.Port,
		User:     c.Config.Database.User,
		Password: c.Config.Database.Password,
		DBName:   c.Config.Database.DBName,
		SSLMode:  c.Config.Database.SSLMode,
	}

	db, err := postgres.NewDatabase(dbConfig)
	if err != nil {
		return err
	}
	c.DB = db
	logger.Startup("db_connected", "Database connected", nil)

	// Run migrations
	if err := postgres.Migrate(db); err != nil {
		return err
	}
	logger.Startup("db_migrated", "Database migrated", nil)

	// Initialize Redis
	redisConfig := redis.RedisConfig{
		Host:     c.Config.Redis.Host,
		Port:     c.Config.Redis.Port,
		Password: c.Config.Redis.Password,
		DB:       c.Config.Redis.DB,
	}
	c.RedisClient = redis.NewRedisClient(redisConfig)

	// Test Redis connection
	if err := c.RedisClient.Ping(context.Background()); err != nil {
		logger.StartupWarn("redis_connection_failed", "Redis connection failed", map[string]interface{}{"error": err.Error()})
	} else {
		logger.Startup("redis_connected", "Redis connected", nil)
	}

	// Initialize Bunny Storage
	bunnyConfig := storage.BunnyConfig{
		StorageZone: c.Config.Bunny.StorageZone,
		AccessKey:   c.Config.Bunny.AccessKey,
		BaseURL:     c.Config.Bunny.BaseURL,
		CDNUrl:      c.Config.Bunny.CDNUrl,
	}
	c.BunnyStorage = storage.NewBunnyStorage(bunnyConfig)
	logger.Startup("bunny_storage_initialized", "Bunny Storage initialized", nil)

	// Initialize Google OAuth
	c.GoogleOAuth = oauth.NewGoogleOAuth(c.Config.Google)
	if err := c.GoogleOAuth.ValidateConfig(); err != nil {
		logger.StartupWarn("google_oauth_not_configured", "Google OAuth not configured", map[string]interface{}{"error": err.Error()})
	} else {
		logger.Startup("google_oauth_initialized", "Google OAuth initialized", nil)
	}

	// Initialize Google Drive Client
	c.GoogleDrive = googledrive.NewDriveClient(c.Config.GoogleDrive)
	if err := c.GoogleDrive.ValidateConfig(); err != nil {
		logger.StartupWarn("google_drive_not_configured", "Google Drive not configured", map[string]interface{}{"error": err.Error()})
	} else {
		logger.Startup("google_drive_initialized", "Google Drive client initialized", nil)
	}

	// Initialize Gemini Client
	if c.Config.Gemini.APIKey != "" {
		geminiClient, err := gemini.NewGeminiClient(c.Config.Gemini.APIKey, c.Config.Gemini.Model)
		if err != nil {
			logger.StartupWarn("gemini_init_failed", "Failed to initialize Gemini client", map[string]interface{}{"error": err.Error()})
		} else {
			c.GeminiClient = geminiClient
			logger.Startup("gemini_initialized", "Gemini client initialized", nil)
		}
	} else {
		logger.StartupWarn("gemini_not_configured", "Gemini API key not configured", nil)
	}

	return nil
}

func (c *Container) initRepositories() error {
	c.UserRepository = postgres.NewUserRepository(c.DB)
	c.TaskRepository = postgres.NewTaskRepository(c.DB)
	c.FileRepository = postgres.NewFileRepository(c.DB)
	c.JobRepository = postgres.NewJobRepository(c.DB)
	c.PhotoRepository = postgres.NewPhotoRepository(c.DB)
	c.SyncJobRepository = postgres.NewSyncJobRepository(c.DB)
	c.FaceRepository = postgres.NewFaceRepository(c.DB)
	c.PersonRepository = postgres.NewPersonRepository(c.DB)
	c.SharedFolderRepository = postgres.NewSharedFolderRepository(c.DB)
	logger.Startup("repositories_initialized", "Repositories initialized", nil)
	return nil
}

func (c *Container) initServices() error {
	c.UserService = serviceimpl.NewUserService(c.UserRepository, c.Config.JWT.Secret)
	c.TaskService = serviceimpl.NewTaskService(c.TaskRepository, c.UserRepository)
	c.FileService = serviceimpl.NewFileService(c.FileRepository, c.UserRepository, c.BunnyStorage)
	c.AuthService = serviceimpl.NewAuthService(c.UserRepository, c.GoogleOAuth, c.Config.JWT.Secret)
	c.DriveService = serviceimpl.NewDriveService(c.GoogleDrive, c.UserRepository, c.PhotoRepository, c.SyncJobRepository)

	// Initialize Face Client (needed for FaceService)
	if c.Config.FaceAPI.Enabled {
		c.FaceClient = faceapi.NewFaceClient(c.Config.FaceAPI.BaseURL)
		c.FaceService = serviceimpl.NewFaceService(c.FaceRepository, c.PhotoRepository, c.PersonRepository, c.UserRepository, c.SharedFolderRepository, c.FaceClient)
	}

	// Initialize News Service (requires Google Drive - Gemini credentials are per-user)
	if c.GoogleDrive != nil {
		c.NewsService = serviceimpl.NewNewsServiceWithDrive(c.GoogleDrive, c.PhotoRepository, c.UserRepository, c.SharedFolderRepository)
		logger.Startup("news_service_initialized", "News service initialized", nil)
	}

	// SharedFolderService will be initialized after workers (needs SyncWorker)

	logger.Startup("services_initialized", "Services initialized", nil)
	return nil
}

func (c *Container) initScheduler() error {
	c.EventScheduler = scheduler.NewEventScheduler()
	c.JobService = serviceimpl.NewJobService(c.JobRepository, c.EventScheduler)

	// Start the scheduler
	c.EventScheduler.Start()
	logger.Startup("scheduler_started", "Event scheduler started", nil)

	// Load and schedule existing active jobs
	ctx := context.Background()
	jobs, _, err := c.JobService.ListJobs(ctx, 0, 1000)
	if err != nil {
		logger.StartupWarn("jobs_load_failed", "Failed to load existing jobs", map[string]interface{}{"error": err.Error()})
		return nil
	}

	activeJobCount := 0
	for _, job := range jobs {
		if job.IsActive {
			err := c.EventScheduler.AddJob(job.ID.String(), job.CronExpr, func() {
				c.JobService.ExecuteJob(ctx, job)
			})
			if err != nil {
				logger.StartupWarn("job_schedule_failed", "Failed to schedule job", map[string]interface{}{"job_name": job.Name, "error": err.Error()})
			} else {
				activeJobCount++
			}
		}
	}

	if activeJobCount > 0 {
		logger.Startup("jobs_scheduled", "Scheduled active jobs", map[string]interface{}{"count": activeJobCount})
	}

	return nil
}

func (c *Container) initWorkers() error {
	// Initialize Sync Worker
	c.SyncWorker = worker.NewSyncWorker(
		c.GoogleDrive,
		c.SharedFolderRepository,
		c.PhotoRepository,
		c.SyncJobRepository,
	)

	// Start the sync worker
	c.SyncWorker.Start()

	// NOTE: Disabled auto sync on startup - users should manually trigger sync when needed
	// c.autoSyncOnStartup()

	// Initialize Face Worker (if enabled and FaceClient is available)
	if c.Config.FaceAPI.Enabled && c.FaceClient != nil {
		c.FaceWorker = worker.NewFaceWorker(
			c.FaceClient,
			c.GoogleDrive,
			c.UserRepository,
			c.PhotoRepository,
			c.FaceRepository,
			c.SharedFolderRepository,
		)

		// Start the face worker
		c.FaceWorker.Start()
	} else if !c.Config.FaceAPI.Enabled {
		logger.Startup("face_api_disabled", "Face API is disabled, skipping face worker initialization", nil)
	}

	// Initialize SharedFolderService (needs SyncWorker, SyncJobRepository, PhotoRepository, and UserRepository)
	c.SharedFolderService = serviceimpl.NewSharedFolderService(
		c.SharedFolderRepository,
		c.SyncJobRepository,
		c.PhotoRepository,
		c.UserRepository,
		c.GoogleDrive,
		c.SyncWorker,
	)
	logger.Startup("shared_folder_service_initialized", "SharedFolder service initialized", nil)

	// Schedule webhook renewal job (runs every 6 hours)
	c.scheduleWebhookRenewal()

	return nil
}

// scheduleWebhookRenewal sets up a scheduled job to renew expiring webhooks
func (c *Container) scheduleWebhookRenewal() {
	if c.EventScheduler == nil || c.SharedFolderService == nil {
		logger.StartupWarn("webhook_renewal_skip", "Scheduler or SharedFolderService not available, skipping webhook renewal job", nil)
		return
	}

	// Run every 6 hours: "0 */6 * * *"
	err := c.EventScheduler.AddJob("webhook-renewal", "0 */6 * * *", func() {
		ctx := context.Background()
		renewed, failed, err := c.SharedFolderService.RenewExpiringWebhooks(ctx)
		if err != nil {
			logger.SchedulerError("webhook_renewal_job_error", "Webhook renewal job failed", err, nil)
			return
		}
		if renewed > 0 || failed > 0 {
			logger.Scheduler("webhook_renewal_job_done", "Webhook renewal job completed", map[string]interface{}{
				"renewed": renewed,
				"failed":  failed,
			})
		}
	})

	if err != nil {
		logger.StartupWarn("webhook_renewal_schedule_failed", "Failed to schedule webhook renewal job", map[string]interface{}{"error": err.Error()})
	} else {
		logger.Startup("webhook_renewal_scheduled", "Webhook renewal job scheduled (every 6 hours)", nil)
	}
}

// autoSyncOnStartup creates sync jobs for all users with Drive connected
func (c *Container) autoSyncOnStartup() {
	ctx := context.Background()

	// Get all users
	users, err := c.UserRepository.List(ctx, 0, 1000)
	if err != nil {
		logger.StartupWarn("auto_sync_list_users_failed", "Failed to list users for auto sync", map[string]interface{}{"error": err.Error()})
		return
	}

	syncCount := 0
	for _, user := range users {
		// Check if user has Drive connected with root folder set
		if user.DriveRefreshToken != "" && user.DriveRootFolderID != "" {
			// Check if there's already a pending/running sync job for this user
			existingJob, _ := c.SyncJobRepository.GetLatestByUserAndType(ctx, user.ID, "drive_sync")
			if existingJob != nil && (existingJob.Status == "pending" || existingJob.Status == "running") {
				continue // Skip if already has a pending/running job
			}

			// Create new sync job
			job, err := c.DriveService.StartSync(ctx, user.ID)
			if err != nil {
				logger.StartupWarn("auto_sync_job_failed", "Failed to create auto sync job for user", map[string]interface{}{"user_id": user.ID.String(), "error": err.Error()})
				continue
			}
			syncCount++
			logger.Startup("auto_sync_job_created", "Auto sync job created", map[string]interface{}{"user_id": user.ID.String(), "job_id": job.ID.String()})
		}
	}

	if syncCount > 0 {
		logger.Startup("auto_sync_jobs_created", "Created auto sync jobs on startup", map[string]interface{}{"count": syncCount})
	}
}

func (c *Container) Cleanup() error {
	logger.Startup("cleanup_started", "Starting cleanup...", nil)

	// Stop face worker
	if c.FaceWorker != nil {
		if c.FaceWorker.IsRunning() {
			c.FaceWorker.Stop()
		}
	}

	// Stop sync worker
	if c.SyncWorker != nil {
		if c.SyncWorker.IsRunning() {
			c.SyncWorker.Stop()
		}
	}

	// Stop scheduler
	if c.EventScheduler != nil {
		if c.EventScheduler.IsRunning() {
			c.EventScheduler.Stop()
			logger.Startup("scheduler_stopped", "Event scheduler stopped", nil)
		} else {
			logger.Startup("scheduler_already_stopped", "Event scheduler was already stopped", nil)
		}
	}

	// Close Redis connection
	if c.RedisClient != nil {
		if err := c.RedisClient.Close(); err != nil {
			logger.StartupWarn("redis_close_failed", "Failed to close Redis connection", map[string]interface{}{"error": err.Error()})
		} else {
			logger.Startup("redis_closed", "Redis connection closed", nil)
		}
	}

	// Close database connection
	if c.DB != nil {
		sqlDB, err := c.DB.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				logger.StartupWarn("db_close_failed", "Failed to close database connection", map[string]interface{}{"error": err.Error()})
			} else {
				logger.Startup("db_closed", "Database connection closed", nil)
			}
		}
	}

	logger.Startup("cleanup_completed", "Cleanup completed", nil)
	return nil
}

func (c *Container) GetServices() (services.UserService, services.TaskService, services.FileService, services.JobService) {
	return c.UserService, c.TaskService, c.FileService, c.JobService
}

func (c *Container) GetConfig() *config.Config {
	return c.Config
}

func (c *Container) GetHandlerServices() *handlers.Services {
	return &handlers.Services{
		UserService:         c.UserService,
		TaskService:         c.TaskService,
		FileService:         c.FileService,
		JobService:          c.JobService,
		AuthService:         c.AuthService,
		DriveService:        c.DriveService,
		FaceService:         c.FaceService,
		NewsService:         c.NewsService,
		SharedFolderService: c.SharedFolderService,
	}
}

func (c *Container) GetHandlerRepositories() *handlers.Repositories {
	return &handlers.Repositories{
		PhotoRepository:        c.PhotoRepository,
		SharedFolderRepository: c.SharedFolderRepository,
		UserRepository:         c.UserRepository,
	}
}