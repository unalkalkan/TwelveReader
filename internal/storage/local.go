package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// LocalAdapter implements the Adapter interface for local filesystem
type LocalAdapter struct {
	basePath string
}

// NewLocalAdapter creates a new local filesystem adapter
func NewLocalAdapter(basePath string) (*LocalAdapter, error) {
	// Ensure base path exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base path: %w", err)
	}

	return &LocalAdapter{
		basePath: basePath,
	}, nil
}

// Put stores data at the given path
func (l *LocalAdapter) Put(ctx context.Context, path string, data io.Reader) error {
	fullPath := l.fullPath(path)

	// Create parent directories
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Copy data
	if _, err := io.Copy(file, data); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

// Get retrieves data from the given path
func (l *LocalAdapter) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	fullPath := l.fullPath(path)

	file, err := os.Open(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	return file, nil
}

// Delete removes data at the given path
func (l *LocalAdapter) Delete(ctx context.Context, path string) error {
	fullPath := l.fullPath(path)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// Exists checks if data exists at the given path
func (l *LocalAdapter) Exists(ctx context.Context, path string) (bool, error) {
	fullPath := l.fullPath(path)

	_, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence: %w", err)
	}

	return true, nil
}

// List returns paths matching the given prefix
func (l *LocalAdapter) List(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := l.fullPath(prefix)
	var paths []string

	// Walk the directory tree
	err := filepath.Walk(l.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if path matches prefix
		if strings.HasPrefix(path, fullPrefix) {
			// Convert to relative path
			relPath, err := filepath.Rel(l.basePath, path)
			if err != nil {
				return err
			}
			paths = append(paths, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return paths, nil
}

// Close cleans up any resources
func (l *LocalAdapter) Close() error {
	// No cleanup needed for local adapter
	return nil
}

// fullPath returns the full filesystem path
func (l *LocalAdapter) fullPath(path string) string {
	return filepath.Join(l.basePath, path)
}
