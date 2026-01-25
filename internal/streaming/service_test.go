package streaming

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/book"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func TestService_StreamSegments(t *testing.T) {
	ctx := context.Background()

	// Setup test storage
	storageAdapter, err := storage.NewLocalAdapter("/tmp/test-streaming")
	if err != nil {
		t.Fatalf("Failed to create storage adapter: %v", err)
	}
	defer storageAdapter.Close()

	// Setup test repository
	repo := book.NewRepository(storageAdapter)

	// Create test segments
	segments := []*types.Segment{
		{
			ID:               "seg_001",
			BookID:           "book_stream_001",
			Chapter:          "chapter_001",
			TOCPath:          []string{"Chapter 1"},
			Text:             "First segment.",
			Language:         "en",
			Person:           "narrator",
			VoiceDescription: "neutral",
			Processing: &types.ProcessingInfo{
				SegmenterVersion: "v1",
				GeneratedAt:      time.Now(),
			},
		},
		{
			ID:               "seg_002",
			BookID:           "book_stream_001",
			Chapter:          "chapter_001",
			TOCPath:          []string{"Chapter 1"},
			Text:             "Second segment.",
			Language:         "en",
			Person:           "narrator",
			VoiceDescription: "neutral",
			Processing: &types.ProcessingInfo{
				SegmenterVersion: "v1",
				GeneratedAt:      time.Now(),
			},
		},
		{
			ID:               "seg_003",
			BookID:           "book_stream_001",
			Chapter:          "chapter_001",
			TOCPath:          []string{"Chapter 1"},
			Text:             "Third segment.",
			Language:         "en",
			Person:           "narrator",
			VoiceDescription: "neutral",
			Processing: &types.ProcessingInfo{
				SegmenterVersion: "v1",
				GeneratedAt:      time.Now(),
			},
		},
	}

	for _, seg := range segments {
		if err := repo.SaveSegment(ctx, seg); err != nil {
			t.Fatalf("Failed to save segment: %v", err)
		}
	}

	// Create streaming service
	service := NewService(repo)

	t.Run("StreamAll", func(t *testing.T) {
		items, err := service.StreamSegments(ctx, "book_stream_001", "")
		if err != nil {
			t.Fatalf("Failed to stream segments: %v", err)
		}

		if len(items) != 3 {
			t.Errorf("Expected 3 segments, got %d", len(items))
		}

		// Verify audio URL is set
		for _, item := range items {
			if item.AudioURL == "" {
				t.Errorf("Expected AudioURL to be set for segment %s", item.ID)
			}
		}
	})

	t.Run("StreamAfter", func(t *testing.T) {
		items, err := service.StreamSegments(ctx, "book_stream_001", "seg_001")
		if err != nil {
			t.Fatalf("Failed to stream segments: %v", err)
		}

		if len(items) != 2 {
			t.Errorf("Expected 2 segments after seg_001, got %d", len(items))
		}

		if len(items) > 0 && items[0].ID != "seg_002" {
			t.Errorf("Expected first segment to be seg_002, got %s", items[0].ID)
		}
	})

	t.Run("StreamAfterLast", func(t *testing.T) {
		items, err := service.StreamSegments(ctx, "book_stream_001", "seg_003")
		if err != nil {
			t.Fatalf("Failed to stream segments: %v", err)
		}

		if len(items) != 0 {
			t.Errorf("Expected 0 segments after seg_003, got %d", len(items))
		}
	})
}

func TestEncodeNDJSON(t *testing.T) {
	items := []StreamItem{
		{
			Segment: &types.Segment{
				ID:       "seg_001",
				BookID:   "book_test",
				Text:     "Test segment 1",
				Language: "en",
				Person:   "narrator",
			},
			AudioURL: "/api/v1/books/book_test/audio/seg_001",
		},
		{
			Segment: &types.Segment{
				ID:       "seg_002",
				BookID:   "book_test",
				Text:     "Test segment 2",
				Language: "en",
				Person:   "narrator",
			},
			AudioURL: "/api/v1/books/book_test/audio/seg_002",
		},
	}

	ndjson, err := EncodeNDJSON(items)
	if err != nil {
		t.Fatalf("Failed to encode NDJSON: %v", err)
	}

	// Verify format
	lines := strings.Split(strings.TrimSpace(ndjson), "\n")
	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

	// Verify each line is valid JSON
	for i, line := range lines {
		var item StreamItem
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			t.Errorf("Line %d is not valid JSON: %v", i, err)
		}

		// Verify audio URL
		if item.AudioURL == "" {
			t.Errorf("Line %d missing AudioURL", i)
		}
	}
}

func TestGetAudioURL(t *testing.T) {
	service := &Service{}

	url := service.getAudioURL("book_123", "seg_456")

	expectedPrefix := "/api/v1/books/book_123/audio/seg_456"
	if !strings.Contains(url, expectedPrefix) {
		t.Errorf("Expected URL to contain '%s', got '%s'", expectedPrefix, url)
	}
}
