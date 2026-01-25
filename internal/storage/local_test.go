package storage

import (
	"bytes"
	"context"
	"io"
	"testing"
)

func TestLocalAdapter(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewLocalAdapter(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create local adapter: %v", err)
	}
	defer adapter.Close()

	ctx := context.Background()
	testPath := "test/file.txt"
	testData := []byte("Hello, World!")

	// Test Put
	t.Run("Put", func(t *testing.T) {
		err := adapter.Put(ctx, testPath, bytes.NewReader(testData))
		if err != nil {
			t.Fatalf("Failed to put data: %v", err)
		}
	})

	// Test Exists
	t.Run("Exists", func(t *testing.T) {
		exists, err := adapter.Exists(ctx, testPath)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}
		if !exists {
			t.Error("File should exist after Put")
		}
	})

	// Test Get
	t.Run("Get", func(t *testing.T) {
		reader, err := adapter.Get(ctx, testPath)
		if err != nil {
			t.Fatalf("Failed to get data: %v", err)
		}
		defer reader.Close()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Fatalf("Failed to read data: %v", err)
		}

		if !bytes.Equal(data, testData) {
			t.Errorf("Expected %s, got %s", testData, data)
		}
	})

	// Test List
	t.Run("List", func(t *testing.T) {
		// Put another file
		adapter.Put(ctx, "test/file2.txt", bytes.NewReader([]byte("test2")))

		paths, err := adapter.List(ctx, "test/")
		if err != nil {
			t.Fatalf("Failed to list files: %v", err)
		}

		if len(paths) < 1 {
			t.Errorf("Expected at least 1 file, got %d", len(paths))
		}
	})

	// Test Delete
	t.Run("Delete", func(t *testing.T) {
		err := adapter.Delete(ctx, testPath)
		if err != nil {
			t.Fatalf("Failed to delete data: %v", err)
		}

		exists, err := adapter.Exists(ctx, testPath)
		if err != nil {
			t.Fatalf("Failed to check existence: %v", err)
		}
		if exists {
			t.Error("File should not exist after Delete")
		}
	})

	// Test Get non-existent file
	t.Run("GetNonExistent", func(t *testing.T) {
		_, err := adapter.Get(ctx, "non-existent.txt")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
}

func TestLocalAdapterConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	adapter, err := NewLocalAdapter(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create local adapter: %v", err)
	}
	defer adapter.Close()

	ctx := context.Background()

	// Test concurrent writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(idx int) {
			path := bytes.NewBufferString("test/file")
			path.WriteString(string(rune('0' + idx)))
			path.WriteString(".txt")
			data := []byte("test data")
			err := adapter.Put(ctx, path.String(), bytes.NewReader(data))
			if err != nil {
				t.Errorf("Failed to put data: %v", err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}
