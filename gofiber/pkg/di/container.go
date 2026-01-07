package di

import (
	"context"
	"log"

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
	log.Println("✓ Configuration loaded")
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
	log.Println("✓ Database connected")

	// Run migrations
	if err := postgres.Migrate(db); err != nil {
		return err
	}
	log.Println("✓ Database migrated")

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
		log.Printf("Warning: Redis connection failed: %v", err)
	} else {
		log.Println("✓ Redis connected")
	}

	// Initialize Bunny Storage
	bunnyConfig := storage.BunnyConfig{
		StorageZone: c.Config.Bunny.StorageZone,
		AccessKey:   c.Config.Bunny.AccessKey,
		BaseURL:     c.Config.Bunny.BaseURL,
		CDNUrl:      c.Config.Bunny.CDNUrl,
	}
	c.BunnyStorage = storage.NewBunnyStorage(bunnyConfig)
	log.Println("✓ Bunny Storage initialized")

	// Initialize Google OAuth
	c.GoogleOAuth = oauth.NewGoogleOAuth(c.Config.Google)
	if err := c.GoogleOAuth.ValidateConfig(); err != nil {
		log.Printf("Warning: Google OAuth not configured: %v", err)
	} else {
		log.Println("✓ Google OAuth initialized")
	}

	// Initialize Google Drive Client
	c.GoogleDrive = googledrive.NewDriveClient(c.Config.GoogleDrive)
	if err := c.GoogleDrive.ValidateConfig(); err != nil {
		log.Printf("Warning: Google Drive not configured: %v", err)
	} else {
		log.Println("✓ Google Drive client initialized")
	}

	// Initialize Gemini Client
	if c.Config.Gemini.APIKey != "" {
		geminiClient, err := gemini.NewGeminiClient(c.Config.Gemini.APIKey, c.Config.Gemini.Model)
		if err != nil {
			log.Printf("Warning: Failed to initialize Gemini client: %v", err)
		} else {
			c.GeminiClient = geminiClient
			log.Println("✓ Gemini client initialized")
		}
	} else {
		log.Println("Warning: Gemini API key not configured")
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
	log.Println("✓ Repositories initialized")
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
		log.Println("✓ News service initialized")
	}

	// SharedFolderService will be initialized after workers (needs SyncWorker)

	log.Println("✓ Services initialized")
	return nil
}

func (c *Container) initScheduler() error {
	c.EventScheduler = scheduler.NewEventScheduler()
	c.JobService = serviceimpl.NewJobService(c.JobRepository, c.EventScheduler)

	// Start the scheduler
	c.EventScheduler.Start()
	log.Println("✓ Event scheduler started")

	// Load and schedule existing active jobs
	ctx := context.Background()
	jobs, _, err := c.JobService.ListJobs(ctx, 0, 1000)
	if err != nil {
		log.Printf("Warning: Failed to load existing jobs: %v", err)
		return nil
	}

	activeJobCount := 0
	for _, job := range jobs {
		if job.IsActive {
			err := c.EventScheduler.AddJob(job.ID.String(), job.CronExpr, func() {
				c.JobService.ExecuteJob(ctx, job)
			})
			if err != nil {
				log.Printf("Warning: Failed to schedule job %s: %v", job.Name, err)
			} else {
				activeJobCount++
			}
		}
	}

	if activeJobCount > 0 {
		log.Printf("✓ Scheduled %d active jobs", activeJobCount)
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
		log.Println("Face API is disabled, skipping face worker initialization")
	}

	// Initialize SharedFolderService (needs SyncWorker, SyncJobRepository, and PhotoRepository)
	c.SharedFolderService = serviceimpl.NewSharedFolderService(
		c.SharedFolderRepository,
		c.SyncJobRepository,
		c.PhotoRepository,
		c.GoogleDrive,
		c.SyncWorker,
	)
	log.Println("✓ SharedFolder service initialized")

	return nil
}

// autoSyncOnStartup creates sync jobs for all users with Drive connected
func (c *Container) autoSyncOnStartup() {
	ctx := context.Background()

	// Get all users
	users, err := c.UserRepository.List(ctx, 0, 1000)
	if err != nil {
		log.Printf("Warning: Failed to list users for auto sync: %v", err)
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
				log.Printf("Warning: Failed to create auto sync job for user %s: %v", user.ID, err)
				continue
			}
			syncCount++
			log.Printf("Auto sync job created for user %s (job: %s)", user.ID, job.ID)
		}
	}

	if syncCount > 0 {
		log.Printf("✓ Created %d auto sync jobs on startup", syncCount)
	}
}

func (c *Container) Cleanup() error {
	log.Println("Starting cleanup...")

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
			log.Println("✓ Event scheduler stopped")
		} else {
			log.Println("✓ Event scheduler was already stopped")
		}
	}

	// Close Redis connection
	if c.RedisClient != nil {
		if err := c.RedisClient.Close(); err != nil {
			log.Printf("Warning: Failed to close Redis connection: %v", err)
		} else {
			log.Println("✓ Redis connection closed")
		}
	}

	// Close database connection
	if c.DB != nil {
		sqlDB, err := c.DB.DB()
		if err == nil {
			if err := sqlDB.Close(); err != nil {
				log.Printf("Warning: Failed to close database connection: %v", err)
			} else {
				log.Println("✓ Database connection closed")
			}
		}
	}

	log.Println("✓ Cleanup completed")
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