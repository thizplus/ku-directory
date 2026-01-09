package handlers

import (
	"gofiber-template/domain/repositories"
	"gofiber-template/domain/services"
	"gofiber-template/pkg/config"
)

// Services contains all the services needed for handlers
type Services struct {
	UserService         services.UserService
	TaskService         services.TaskService
	FileService         services.FileService
	JobService          services.JobService
	AuthService         services.AuthService
	DriveService        services.DriveService
	FaceService         services.FaceService
	NewsService         services.NewsService
	SharedFolderService services.SharedFolderService
	ActivityLogService  services.ActivityLogService
}

// Repositories contains repositories needed for some handlers
type Repositories struct {
	PhotoRepository        repositories.PhotoRepository
	SharedFolderRepository repositories.SharedFolderRepository
	UserRepository         repositories.UserRepository
}

// Handlers contains all HTTP handlers
type Handlers struct {
	UserHandler         *UserHandler
	TaskHandler         *TaskHandler
	FileHandler         *FileHandler
	JobHandler          *JobHandler
	AuthHandler         *AuthHandler
	DriveHandler        *DriveHandler
	FaceHandler         *FaceHandler
	NewsHandler         *NewsHandler
	SharedFolderHandler *SharedFolderHandler
	LogHandler          *LogHandler
	ActivityLogHandler  *ActivityLogHandler

	// Short accessors for routes
	User         *UserHandler
	Task         *TaskHandler
	File         *FileHandler
	Job          *JobHandler
	Auth         *AuthHandler
	Drive        *DriveHandler
	Face         *FaceHandler
	News         *NewsHandler
	SharedFolder *SharedFolderHandler
	Log          *LogHandler
	ActivityLog  *ActivityLogHandler
}

// NewHandlers creates a new instance of Handlers with all dependencies
func NewHandlers(services *Services, repos *Repositories, cfg *config.Config) *Handlers {
	userHandler := NewUserHandler(services.UserService)
	taskHandler := NewTaskHandler(services.TaskService)
	fileHandler := NewFileHandler(services.FileService)
	jobHandler := NewJobHandler(services.JobService)
	authHandler := NewAuthHandler(services.AuthService)
	driveHandler := NewDriveHandler(services.DriveService)
	faceHandler := NewFaceHandler(services.FaceService)
	newsHandler := NewNewsHandler(services.NewsService)
	logHandler := NewLogHandler(cfg)

	var sharedFolderHandler *SharedFolderHandler
	var activityLogHandler *ActivityLogHandler
	if services.SharedFolderService != nil && repos != nil {
		sharedFolderHandler = NewSharedFolderHandler(
			services.SharedFolderService,
			repos.PhotoRepository,
			repos.SharedFolderRepository,
			repos.UserRepository,
		)
		// Wire shared folder service to drive handler for webhook support
		driveHandler.SetSharedFolderService(services.SharedFolderService)
	}

	if services.ActivityLogService != nil && repos != nil {
		activityLogHandler = NewActivityLogHandler(
			services.ActivityLogService,
			repos.SharedFolderRepository,
		)
	}

	return &Handlers{
		UserHandler:         userHandler,
		TaskHandler:         taskHandler,
		FileHandler:         fileHandler,
		JobHandler:          jobHandler,
		AuthHandler:         authHandler,
		DriveHandler:        driveHandler,
		FaceHandler:         faceHandler,
		NewsHandler:         newsHandler,
		SharedFolderHandler: sharedFolderHandler,
		LogHandler:          logHandler,
		ActivityLogHandler:  activityLogHandler,

		// Short accessors
		User:         userHandler,
		Task:         taskHandler,
		File:         fileHandler,
		Job:          jobHandler,
		Auth:         authHandler,
		Drive:        driveHandler,
		Face:         faceHandler,
		News:         newsHandler,
		SharedFolder: sharedFolderHandler,
		Log:          logHandler,
		ActivityLog:  activityLogHandler,
	}
}