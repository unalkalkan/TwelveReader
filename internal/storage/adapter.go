package storage

import (
	"context"
	"io"
)

// Adapter defines the interface for storage backends
type Adapter interface {
	// Put stores data at the given path
	Put(ctx context.Context, path string, data io.Reader) error

	// Get retrieves data from the given path
	Get(ctx context.Context, path string) (io.ReadCloser, error)

	// Delete removes data at the given path
	Delete(ctx context.Context, path string) error

	// Exists checks if data exists at the given path
	Exists(ctx context.Context, path string) (bool, error)

	// List returns paths matching the given prefix
	List(ctx context.Context, prefix string) ([]string, error)

	// Close cleans up any resources
	Close() error
}

// Metadata represents file metadata
type Metadata struct {
	Path         string
	Size         int64
	LastModified int64
	ContentType  string
}
