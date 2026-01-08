package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Category represents a log category
type Category string

const (
	CategoryAuth      Category = "auth"
	CategoryWebhook   Category = "webhook"
	CategoryWebSocket Category = "websocket"
	CategorySync      Category = "sync"
	CategoryAPI       Category = "api"
	CategoryDB        Category = "db"
	CategoryDrive     Category = "drive"
	CategoryFace      Category = "face"
	CategoryStartup   Category = "startup"
)

// Level represents log level
type Level string

const (
	LevelDebug Level = "DEBUG"
	LevelInfo  Level = "INFO"
	LevelWarn  Level = "WARN"
	LevelError Level = "ERROR"
)

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     Level                  `json:"level"`
	Category  Category               `json:"category"`
	Action    string                 `json:"action"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	Duration  string                 `json:"duration,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// Logger is the main logger struct
type Logger struct {
	mu       sync.Mutex
	logDir   string
	writers  map[Category]*os.File
	console  bool
	minLevel Level
}

var (
	defaultLogger *Logger
	once          sync.Once
)

// Init initializes the default logger
func Init(logDir string, console bool) error {
	var err error
	once.Do(func() {
		defaultLogger, err = NewLogger(logDir, console)
	})
	return err
}

// NewLogger creates a new logger
func NewLogger(logDir string, console bool) (*Logger, error) {
	// Create log directory if not exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	return &Logger{
		logDir:   logDir,
		writers:  make(map[Category]*os.File),
		console:  console,
		minLevel: LevelDebug,
	}, nil
}

// getWriter returns or creates a file writer for the category
func (l *Logger) getWriter(category Category) (io.Writer, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if writer exists and is for today
	today := time.Now().Format("2006-01-02")
	filename := fmt.Sprintf("%s_%s.log", category, today)
	filepath := filepath.Join(l.logDir, filename)

	if writer, exists := l.writers[category]; exists {
		// Check if file is still for today
		if info, err := writer.Stat(); err == nil {
			if info.Name() == filename {
				return writer, nil
			}
		}
		writer.Close()
	}

	// Create new file
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	l.writers[category] = file
	return file, nil
}

// Log writes a log entry
func (l *Logger) Log(entry LogEntry) {
	entry.Timestamp = time.Now()

	// Format as JSON
	jsonData, err := json.Marshal(entry)
	if err != nil {
		fmt.Printf("Error marshaling log entry: %v\n", err)
		return
	}

	// Write to file
	writer, err := l.getWriter(entry.Category)
	if err != nil {
		fmt.Printf("Error getting log writer: %v\n", err)
	} else {
		fmt.Fprintln(writer, string(jsonData))
	}

	// Also write to console if enabled
	if l.console {
		l.printToConsole(entry)
	}
}

// printToConsole prints formatted log to console
func (l *Logger) printToConsole(entry LogEntry) {
	timestamp := entry.Timestamp.Format("15:04:05.000")

	// Color codes for levels
	levelColors := map[Level]string{
		LevelDebug: "\033[36m", // Cyan
		LevelInfo:  "\033[32m", // Green
		LevelWarn:  "\033[33m", // Yellow
		LevelError: "\033[31m", // Red
	}
	reset := "\033[0m"

	color := levelColors[entry.Level]

	fmt.Printf("%s[%s]%s [%s] [%s] %s: %s",
		color,
		entry.Level,
		reset,
		timestamp,
		entry.Category,
		entry.Action,
		entry.Message,
	)

	if entry.UserID != "" {
		fmt.Printf(" (user: %s)", entry.UserID)
	}
	if entry.Duration != "" {
		fmt.Printf(" (duration: %s)", entry.Duration)
	}
	if entry.Error != "" {
		fmt.Printf(" ERROR: %s", entry.Error)
	}
	fmt.Println()

	// Print data if present
	if len(entry.Data) > 0 {
		dataJSON, _ := json.MarshalIndent(entry.Data, "    ", "  ")
		fmt.Printf("    Data: %s\n", string(dataJSON))
	}
}

// Close closes all file writers
func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, writer := range l.writers {
		writer.Close()
	}
	l.writers = make(map[Category]*os.File)
}

// Default returns the default logger
func Default() *Logger {
	if defaultLogger == nil {
		// Initialize with default settings if not initialized
		Init("logs", true)
	}
	return defaultLogger
}

// Helper functions for common log operations

// Auth logs authentication related events
func Auth(action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelInfo,
		Category: CategoryAuth,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// AuthError logs authentication errors
func AuthError(action, message string, err error, data map[string]interface{}) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	Default().Log(LogEntry{
		Level:    LevelError,
		Category: CategoryAuth,
		Action:   action,
		Message:  message,
		Error:    errStr,
		Data:     data,
	})
}

// Webhook logs webhook related events
func Webhook(action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelInfo,
		Category: CategoryWebhook,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// WebhookError logs webhook errors
func WebhookError(action, message string, err error, data map[string]interface{}) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	Default().Log(LogEntry{
		Level:    LevelError,
		Category: CategoryWebhook,
		Action:   action,
		Message:  message,
		Error:    errStr,
		Data:     data,
	})
}

// WebSocket logs WebSocket related events
func WebSocket(action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelInfo,
		Category: CategoryWebSocket,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// Sync logs sync related events
func Sync(action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelInfo,
		Category: CategorySync,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// SyncError logs sync errors
func SyncError(action, message string, err error, data map[string]interface{}) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	Default().Log(LogEntry{
		Level:    LevelError,
		Category: CategorySync,
		Action:   action,
		Message:  message,
		Error:    errStr,
		Data:     data,
	})
}

// API logs API request/response events
func API(action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelInfo,
		Category: CategoryAPI,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// DB logs database operations
func DB(action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelDebug,
		Category: CategoryDB,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// Drive logs Google Drive operations
func Drive(action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelInfo,
		Category: CategoryDrive,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// DriveError logs Google Drive errors
func DriveError(action, message string, err error, data map[string]interface{}) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	Default().Log(LogEntry{
		Level:    LevelError,
		Category: CategoryDrive,
		Action:   action,
		Message:  message,
		Error:    errStr,
		Data:     data,
	})
}

// Face logs face recognition operations
func Face(action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelInfo,
		Category: CategoryFace,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// FaceError logs face recognition errors
func FaceError(action, message string, err error, data map[string]interface{}) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	Default().Log(LogEntry{
		Level:    LevelError,
		Category: CategoryFace,
		Action:   action,
		Message:  message,
		Error:    errStr,
		Data:     data,
	})
}

// Startup logs startup/initialization events
func Startup(action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelInfo,
		Category: CategoryStartup,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// StartupError logs startup errors
func StartupError(action, message string, err error, data map[string]interface{}) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	Default().Log(LogEntry{
		Level:    LevelError,
		Category: CategoryStartup,
		Action:   action,
		Message:  message,
		Error:    errStr,
		Data:     data,
	})
}

// StartupWarn logs startup warnings
func StartupWarn(action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelWarn,
		Category: CategoryStartup,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// Info logs info level message
func Info(category Category, action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelInfo,
		Category: category,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// Error logs error level message
func Error(category Category, action, message string, err error, data map[string]interface{}) {
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	Default().Log(LogEntry{
		Level:    LevelError,
		Category: category,
		Action:   action,
		Message:  message,
		Error:    errStr,
		Data:     data,
	})
}

// Debug logs debug level message
func Debug(category Category, action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelDebug,
		Category: category,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// Warn logs warning level message
func Warn(category Category, action, message string, data map[string]interface{}) {
	Default().Log(LogEntry{
		Level:    LevelWarn,
		Category: category,
		Action:   action,
		Message:  message,
		Data:     data,
	})
}

// ReadLogsOptions options for reading logs
type ReadLogsOptions struct {
	Category Category // Filter by category (empty = all)
	Level    Level    // Filter by level (empty = all)
	Lines    int      // Number of lines to return (default 100)
	Search   string   // Search in message/action
}

// ReadLogs reads log entries from files
func ReadLogs(opts ReadLogsOptions) ([]LogEntry, error) {
	return Default().ReadLogs(opts)
}

// ReadLogs reads log entries from the logger's log directory
func (l *Logger) ReadLogs(opts ReadLogsOptions) ([]LogEntry, error) {
	if opts.Lines <= 0 {
		opts.Lines = 100
	}
	if opts.Lines > 1000 {
		opts.Lines = 1000 // Max limit
	}

	var entries []LogEntry

	// Get today's date for log files
	today := time.Now().Format("2006-01-02")

	// Determine which categories to read
	categories := []Category{CategoryAuth, CategoryWebhook, CategoryWebSocket, CategorySync, CategoryAPI, CategoryDB, CategoryDrive, CategoryFace, CategoryStartup}
	if opts.Category != "" {
		categories = []Category{opts.Category}
	}

	// Read from each category file
	for _, cat := range categories {
		filename := fmt.Sprintf("%s_%s.log", cat, today)
		filePath := filepath.Join(l.logDir, filename)

		// Read file if exists
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue // Skip if file doesn't exist
		}

		// Parse each line as JSON
		lines := splitLines(string(data))
		for _, line := range lines {
			if line == "" {
				continue
			}

			var entry LogEntry
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				continue
			}

			// Apply filters
			if opts.Level != "" && entry.Level != opts.Level {
				continue
			}
			if opts.Search != "" {
				if !containsIgnoreCase(entry.Message, opts.Search) &&
					!containsIgnoreCase(entry.Action, opts.Search) &&
					!containsIgnoreCase(entry.Error, opts.Search) {
					continue
				}
			}

			entries = append(entries, entry)
		}
	}

	// Sort by timestamp descending (newest first)
	sortEntriesByTime(entries)

	// Limit results
	if len(entries) > opts.Lines {
		entries = entries[:opts.Lines]
	}

	return entries, nil
}

// GetLogDir returns the log directory path
func GetLogDir() string {
	return Default().logDir
}

// ListLogFiles returns list of log files
func ListLogFiles() ([]string, error) {
	return Default().ListLogFiles()
}

// ListLogFiles returns list of log files in the log directory
func (l *Logger) ListLogFiles() ([]string, error) {
	var files []string

	entries, err := os.ReadDir(l.logDir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".log" {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// Helper functions

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func containsIgnoreCase(s, substr string) bool {
	s = toLower(s)
	substr = toLower(substr)
	return contains(s, substr)
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func sortEntriesByTime(entries []LogEntry) {
	// Simple bubble sort (good enough for small arrays)
	n := len(entries)
	for i := 0; i < n-1; i++ {
		for j := 0; j < n-i-1; j++ {
			if entries[j].Timestamp.Before(entries[j+1].Timestamp) {
				entries[j], entries[j+1] = entries[j+1], entries[j]
			}
		}
	}
}
