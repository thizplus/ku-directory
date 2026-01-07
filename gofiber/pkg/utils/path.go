package utils

import (
	"errors"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	ErrInvalidPath     = errors.New("invalid path format")
	ErrUnsafePath      = errors.New("unsafe path detected")
	ErrPathTooLong     = errors.New("path is too long")
	ErrEmptyPath       = errors.New("path cannot be empty")
	ErrInvalidCharacter = errors.New("path contains invalid characters")
)

const (
	MaxPathLength = 500
)

// ValidateAndSanitizePath validates and sanitizes a custom file path
func ValidateAndSanitizePath(customPath string) (string, error) {
	if customPath == "" {
		return "", ErrEmptyPath
	}

	// Check path length
	if len(customPath) > MaxPathLength {
		return "", ErrPathTooLong
	}

	// Remove leading/trailing whitespace
	customPath = strings.TrimSpace(customPath)

	// Check for directory traversal attempts
	if strings.Contains(customPath, "..") {
		return "", ErrUnsafePath
	}

	// Check for absolute paths (Windows and Unix)
	if filepath.IsAbs(customPath) {
		return "", ErrUnsafePath
	}

	// Check for dangerous characters
	dangerousChars := regexp.MustCompile(`[<>:"|?*\x00-\x1f\x7f]`)
	if dangerousChars.MatchString(customPath) {
		return "", ErrInvalidCharacter
	}

	// Check for reserved Windows names
	reservedNames := []string{
		"CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
	}

	pathParts := strings.Split(customPath, "/")
	for _, part := range pathParts {
		upperPart := strings.ToUpper(strings.TrimSuffix(part, filepath.Ext(part)))
		for _, reserved := range reservedNames {
			if upperPart == reserved {
				return "", ErrInvalidPath
			}
		}
	}

	// Normalize path separators to forward slashes
	customPath = strings.ReplaceAll(customPath, "\\", "/")

	// Remove duplicate slashes
	customPath = regexp.MustCompile(`/+`).ReplaceAllString(customPath, "/")

	// Remove leading slash
	customPath = strings.TrimPrefix(customPath, "/")

	// Remove trailing slash
	customPath = strings.TrimSuffix(customPath, "/")

	if customPath == "" {
		return "", ErrEmptyPath
	}

	return customPath, nil
}

// GenerateStructuredPath creates a structured path from category, entityId, and fileType
func GenerateStructuredPath(userID, category, entityID, fileType string) string {
	if category != "" && entityID != "" && fileType != "" {
		return filepath.Join("uploads", userID, category, entityID, fileType)
	} else if category != "" && fileType != "" {
		return filepath.Join("uploads", userID, category, fileType)
	} else if category != "" {
		return filepath.Join("uploads", userID, category)
	}
	return filepath.Join("uploads", userID)
}

// SanitizeFileName sanitizes a filename to ensure it's safe for storage
func SanitizeFileName(filename string) string {
	// Remove path components
	filename = filepath.Base(filename)

	// Replace dangerous characters with underscore
	dangerousChars := regexp.MustCompile(`[<>:"|?*\x00-\x1f\x7f]`)
	filename = dangerousChars.ReplaceAllString(filename, "_")

	// Trim whitespace
	filename = strings.TrimSpace(filename)

	// Ensure filename is not empty
	if filename == "" || filename == "." || filename == ".." {
		filename = "file"
	}

	return filename
}