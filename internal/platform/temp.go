package platform

import (
	"fmt"
	"os"
	"path/filepath"
)

// InTempDir returns a path within the tabctl temporary directory
func InTempDir(filename string) string {
	tempDir := GetTempDir()
	return filepath.Join(tempDir, filename)
}

// CreateTempFile creates a temporary file with the given prefix
func CreateTempFile(prefix string) (*os.File, error) {
	tempDir := GetTempDir()
	if err := EnsureDir(tempDir); err != nil {
		return nil, err
	}

	return os.CreateTemp(tempDir, prefix)
}

// CreateTempDir creates a temporary directory with the given prefix
func CreateTempDir(prefix string) (string, error) {
	tempDir := GetTempDir()
	if err := EnsureDir(tempDir); err != nil {
		return "", err
	}

	return os.MkdirTemp(tempDir, prefix)
}

// GetDefaultSQLitePath returns the default SQLite database path
func GetDefaultSQLitePath() string {
	return InTempDir("tabs.sqlite")
}

// GetDefaultTSVPath returns the default TSV file path
func GetDefaultTSVPath() string {
	return InTempDir("tabs.tsv")
}

// CleanupTempDir removes all files from the temporary directory
func CleanupTempDir() error {
	tempDir := GetTempDir()
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to clean
	}

	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		path := filepath.Join(tempDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			// Continue cleaning other files even if one fails
			fmt.Printf("Warning: failed to remove %s: %v\n", path, err)
		}
	}

	return nil
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(filename string) (int64, error) {
	info, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// FileExists checks if a file exists
func FileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}

// WriteToFile writes content to a file, creating directories as needed
func WriteToFile(filename, content string) error {
	dir := filepath.Dir(filename)
	if err := EnsureDir(dir); err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(content), 0644)
}

// ReadFromFile reads content from a file
func ReadFromFile(filename string) (string, error) {
	content, err := os.ReadFile(filename)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// AppendToFile appends content to a file
func AppendToFile(filename, content string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}