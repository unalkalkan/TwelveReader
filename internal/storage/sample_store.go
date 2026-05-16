package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
)

// SampleStore stores reusable voice preview audio samples.
type SampleStore interface {
	Put(ctx context.Context, path string, data []byte) error
	Get(ctx context.Context, path string) ([]byte, error)
	Exists(ctx context.Context, path string) (bool, error)
}

// AdapterSampleStore persists samples through the configured storage adapter.
type AdapterSampleStore struct {
	adapter Adapter
}

// NewAdapterSampleStore creates a sample store backed by the app storage adapter.
func NewAdapterSampleStore(adapter Adapter) *AdapterSampleStore {
	return &AdapterSampleStore{adapter: adapter}
}

func (s *AdapterSampleStore) Put(ctx context.Context, path string, data []byte) error {
	if s == nil || s.adapter == nil {
		return fmt.Errorf("sample storage adapter is nil")
	}
	return s.adapter.Put(ctx, path, bytes.NewReader(data))
}

func (s *AdapterSampleStore) Get(ctx context.Context, path string) ([]byte, error) {
	if s == nil || s.adapter == nil {
		return nil, fmt.Errorf("sample storage adapter is nil")
	}
	r, err := s.adapter.Get(ctx, path)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

func (s *AdapterSampleStore) Exists(ctx context.Context, path string) (bool, error) {
	if s == nil || s.adapter == nil {
		return false, fmt.Errorf("sample storage adapter is nil")
	}
	return s.adapter.Exists(ctx, path)
}

// MemorySampleStore is an in-memory implementation for tests.
type MemorySampleStore struct {
	mu   sync.RWMutex
	data map[string][]byte
}

// NewMemorySampleStore creates an in-memory sample store.
func NewMemorySampleStore() *MemorySampleStore {
	return &MemorySampleStore{data: make(map[string][]byte)}
}

func (s *MemorySampleStore) Put(ctx context.Context, path string, data []byte) error {
	if s == nil {
		return fmt.Errorf("memory sample store is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	copyData := append([]byte(nil), data...)
	s.data[path] = copyData
	return nil
}

func (s *MemorySampleStore) Get(ctx context.Context, path string) ([]byte, error) {
	if s == nil {
		return nil, fmt.Errorf("memory sample store is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.data[path]
	if !ok {
		return nil, fmt.Errorf("sample not found: %s", path)
	}
	return append([]byte(nil), data...), nil
}

func (s *MemorySampleStore) Exists(ctx context.Context, path string) (bool, error) {
	if s == nil {
		return false, fmt.Errorf("memory sample store is nil")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[path]
	return ok, nil
}
