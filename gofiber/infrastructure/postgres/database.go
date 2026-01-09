package postgres

import (
	"fmt"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"gofiber-template/domain/models"
)

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func NewDatabase(config DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=UTC",
		config.Host, config.User, config.Password, config.DBName, config.Port, config.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	return db, nil
}

func Migrate(db *gorm.DB) error {
	// Enable pgvector extension for face embeddings
	if err := db.Exec("CREATE EXTENSION IF NOT EXISTS vector").Error; err != nil {
		return fmt.Errorf("failed to enable pgvector extension: %v", err)
	}

	// Run auto migrations first to create tables
	if err := db.AutoMigrate(
		// Core models
		&models.User{},
		&models.Task{},
		&models.File{},
		&models.Job{},

		// Shared folder models (must come before Photo/Face)
		&models.SharedFolder{},
		&models.UserFolderAccess{},

		// KU Directory models
		&models.Photo{},
		&models.Face{},
		&models.Person{},
		&models.News{},
		&models.NewsPhoto{},
		&models.SyncJob{},
		&models.DriveWebhookLog{},
		&models.ActivityLog{},
	); err != nil {
		return fmt.Errorf("failed to run auto migrations: %v", err)
	}

	// Run manual migrations for SharedFolder refactor
	// These handle changes that AutoMigrate cannot do (dropping constraints, making columns nullable)
	if err := runSharedFolderMigrations(db); err != nil {
		return fmt.Errorf("failed to run shared folder migrations: %v", err)
	}

	return nil
}

// runSharedFolderMigrations handles the migration from user-centric to server-centric sync
func runSharedFolderMigrations(db *gorm.DB) error {
	migrations := []string{
		// Photos table: Add shared_folder_id and make user_id nullable
		`ALTER TABLE photos ADD COLUMN IF NOT EXISTS shared_folder_id uuid`,
		`DO $$ BEGIN
			ALTER TABLE photos ALTER COLUMN user_id DROP NOT NULL;
		EXCEPTION WHEN others THEN NULL; END $$`,

		// Photos: Drop old user-based constraints and indexes
		`DROP INDEX IF EXISTS idx_photos_user_drive_file`,
		`DROP INDEX IF EXISTS idx_photos_user_id`,
		`DO $$ BEGIN
			ALTER TABLE photos DROP CONSTRAINT IF EXISTS fk_users_photos;
		EXCEPTION WHEN others THEN NULL; END $$`,

		// Photos: Add new shared_folder constraints
		`CREATE INDEX IF NOT EXISTS idx_photos_shared_folder_id ON photos(shared_folder_id)`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_photos_folder_drive_file ON photos(shared_folder_id, drive_file_id)`,
		`DO $$ BEGIN
			ALTER TABLE photos ADD CONSTRAINT fk_photos_shared_folder
				FOREIGN KEY (shared_folder_id) REFERENCES shared_folders(id);
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		// Faces table: Add shared_folder_id and make user_id nullable
		`ALTER TABLE faces ADD COLUMN IF NOT EXISTS shared_folder_id uuid`,
		`DO $$ BEGIN
			ALTER TABLE faces ALTER COLUMN user_id DROP NOT NULL;
		EXCEPTION WHEN others THEN NULL; END $$`,

		// Faces: Drop old user-based constraints
		`DROP INDEX IF EXISTS idx_faces_user_id`,
		`DO $$ BEGIN
			ALTER TABLE faces DROP CONSTRAINT IF EXISTS fk_faces_user;
		EXCEPTION WHEN others THEN NULL; END $$`,

		// Faces: Add new shared_folder constraints
		`CREATE INDEX IF NOT EXISTS idx_faces_shared_folder_id ON faces(shared_folder_id)`,
		`DO $$ BEGIN
			ALTER TABLE faces ADD CONSTRAINT fk_faces_shared_folder
				FOREIGN KEY (shared_folder_id) REFERENCES shared_folders(id);
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	}

	for _, sql := range migrations {
		if err := db.Exec(sql).Error; err != nil {
			return fmt.Errorf("migration failed: %s, error: %v", sql[:50], err)
		}
	}

	return nil
}