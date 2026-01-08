package config

import (
	"os"
	"strconv"
	"github.com/joho/godotenv"
)

type Config struct {
	App         AppConfig
	Database    DatabaseConfig
	Redis       RedisConfig
	JWT         JWTConfig
	Admin       AdminConfig
	RateLimit   RateLimitConfig
	Bunny       BunnyConfig
	Google      GoogleOAuthConfig
	GoogleDrive GoogleDriveConfig
	FaceAPI     FaceAPIConfig
	Gemini      GeminiConfig
}

type AdminConfig struct {
	Token string // Separate admin token for log access (falls back to JWT secret if not set)
}

type RateLimitConfig struct {
	Enabled       bool // Enable/disable rate limiting
	MaxRequests   int  // Max requests per window
	WindowSeconds int  // Time window in seconds
	// Stricter limits for sensitive endpoints
	AuthMaxRequests   int // Max auth requests per window (login, register, etc.)
	AuthWindowSeconds int // Auth time window in seconds
}

type AppConfig struct {
	Name string
	Port string
	Env  string
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret string
}

type BunnyConfig struct {
	StorageZone string
	AccessKey   string
	BaseURL     string
	CDNUrl      string
}

type GoogleOAuthConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

type GoogleDriveConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
	WebhookURL   string // URL for Drive push notifications
}

type FaceAPIConfig struct {
	BaseURL string // Base URL of the Python InsightFace service
	Enabled bool   // Enable/disable face processing
}

type GeminiConfig struct {
	APIKey string // Gemini API Key
	Model  string // Model name (e.g., gemini-2.0-flash)
}

func LoadConfig() (*Config, error) {
	// Load .env file if exists (optional for production)
	_ = godotenv.Load() // Ignore error if .env doesn't exist

	redisDB, _ := strconv.Atoi(getEnv("REDIS_DB", "0"))

	config := &Config{
		App: AppConfig{
			Name: getEnv("APP_NAME", "GoFiber Template"),
			Port: getEnv("APP_PORT", "3000"),
			Env:  getEnv("APP_ENV", "development"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnv("DB_PORT", "5432"),
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", ""),
			DBName:   getEnv("DB_NAME", "gofiber_template"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "your-secret-key"),
		},
		Admin: AdminConfig{
			Token: getEnv("ADMIN_TOKEN", ""), // Will fall back to JWT_SECRET in handler if empty
		},
		Bunny: BunnyConfig{
			StorageZone: getEnv("BUNNY_STORAGE_ZONE", ""),
			AccessKey:   getEnv("BUNNY_ACCESS_KEY", ""),
			BaseURL:     getEnv("BUNNY_BASE_URL", "https://storage.bunnycdn.com"),
			CDNUrl:      getEnv("BUNNY_CDN_URL", ""),
		},
		Google: GoogleOAuthConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
			RedirectURL:  getEnv("GOOGLE_REDIRECT_URL", "http://localhost:8080/api/v1/auth/google/callback"),
		},
		GoogleDrive: GoogleDriveConfig{
			ClientID:     getEnv("GOOGLE_CLIENT_ID", ""),     // Same as Google OAuth
			ClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""), // Same as Google OAuth
			RedirectURL:  getEnv("GOOGLE_DRIVE_REDIRECT_URL", "http://localhost:8080/api/v1/drive/callback"),
			WebhookURL:   getEnv("GOOGLE_DRIVE_WEBHOOK_URL", ""),
		},
		FaceAPI: FaceAPIConfig{
			BaseURL: getEnv("FACE_API_URL", "http://localhost:5000"),
			Enabled: getEnv("FACE_API_ENABLED", "true") == "true",
		},
		Gemini: GeminiConfig{
			APIKey: getEnv("GEMINI_API_KEY", ""),
			Model:  getEnv("GEMINI_MODEL", "gemini-2.0-flash"),
		},
		RateLimit: RateLimitConfig{
			Enabled:           getEnv("RATE_LIMIT_ENABLED", "true") == "true",
			MaxRequests:       getEnvInt("RATE_LIMIT_MAX_REQUESTS", 100),
			WindowSeconds:     getEnvInt("RATE_LIMIT_WINDOW_SECONDS", 60),
			AuthMaxRequests:   getEnvInt("RATE_LIMIT_AUTH_MAX_REQUESTS", 10),
			AuthWindowSeconds: getEnvInt("RATE_LIMIT_AUTH_WINDOW_SECONDS", 60),
		},
	}

	return config, nil
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	intValue, err := strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}