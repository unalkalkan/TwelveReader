package tts

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/book"
	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func TestOrchestrator_SynthesizeBook(t *testing.T) {
	ctx := context.Background()

	// Setup test storage
	storageAdapter, err := storage.NewLocalAdapter("/tmp/test-tts-orchestrator")
	if err != nil {
		t.Fatalf("Failed to create storage adapter: %v", err)
	}
	defer storageAdapter.Close()

	// Setup test repository
	repo := book.NewRepository(storageAdapter)

	// Create a test book
	testBook := &types.Book{
		ID:            "book_test_001",
		Title:         "Test Book",
		Author:        "Test Author",
		Language:      "en",
		UploadedAt:    time.Now(),
		Status:        "ready",
		TotalChapters: 1,
		TotalSegments: 2,
	}
	if err := repo.SaveBook(ctx, testBook); err != nil {
		t.Fatalf("Failed to save book: %v", err)
	}

	// Create test segments
	segments := []*types.Segment{
		{
			ID:               "seg_001",
			BookID:           "book_test_001",
			Chapter:          "chapter_001",
			TOCPath:          []string{"Chapter 1"},
			Text:             "Hello world.",
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
			BookID:           "book_test_001",
			Chapter:          "chapter_001",
			TOCPath:          []string{"Chapter 1"},
			Text:             "This is a test.",
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

	// Create voice map
	voiceMap := &types.VoiceMap{
		BookID: "book_test_001",
		Persons: []types.PersonVoice{
			{ID: "narrator", ProviderVoice: "voice_1"},
		},
	}
	if err := repo.SaveVoiceMap(ctx, voiceMap); err != nil {
		t.Fatalf("Failed to save voice map: %v", err)
	}

	// Setup provider registry with stub TTS
	registry := provider.NewRegistry()
	ttsConfig := types.TTSProviderConfig{
		Name:             "test-tts",
		Enabled:          true,
		MaxSegmentSize:   500,
		Concurrency:      3,
		TimestampPrec:    "word",
	}
	registry.RegisterTTS(provider.NewStubTTSProvider(ttsConfig))

	// Create orchestrator
	orchestrator := NewOrchestrator(registry, repo, storageAdapter, 2)

	// Synthesize book
	if err := orchestrator.SynthesizeBook(ctx, "book_test_001", "test-tts"); err != nil {
		t.Fatalf("Failed to synthesize book: %v", err)
	}

	// Verify book status
	updatedBook, err := repo.GetBook(ctx, "book_test_001")
	if err != nil {
		t.Fatalf("Failed to get book: %v", err)
	}

	if updatedBook.Status != "synthesized" {
		t.Errorf("Expected book status 'synthesized', got '%s'", updatedBook.Status)
	}

	// Verify segments were updated
	updatedSegments, err := repo.ListSegments(ctx, "book_test_001")
	if err != nil {
		t.Fatalf("Failed to list segments: %v", err)
	}

	for _, seg := range updatedSegments {
		if seg.VoiceID == "" {
			t.Errorf("Expected VoiceID to be set for segment %s", seg.ID)
		}
		if seg.Processing == nil || seg.Processing.TTSProvider == "" {
			t.Errorf("Expected TTS provider to be set for segment %s", seg.ID)
		}
		if seg.Timestamps == nil {
			t.Errorf("Expected timestamps to be set for segment %s", seg.ID)
		}
	}
}

func TestOrchestrator_SynthesizeBook_NotReady(t *testing.T) {
	ctx := context.Background()

	// Setup test storage
	storageAdapter, err := storage.NewLocalAdapter("/tmp/test-tts-orchestrator-not-ready")
	if err != nil {
		t.Fatalf("Failed to create storage adapter: %v", err)
	}
	defer storageAdapter.Close()

	// Setup test repository
	repo := book.NewRepository(storageAdapter)

	// Create a test book with wrong status
	testBook := &types.Book{
		ID:     "book_test_002",
		Status: "parsing",
	}
	if err := repo.SaveBook(ctx, testBook); err != nil {
		t.Fatalf("Failed to save book: %v", err)
	}

	// Setup provider registry
	registry := provider.NewRegistry()

	// Create orchestrator
	orchestrator := NewOrchestrator(registry, repo, storageAdapter, 2)

	// Try to synthesize book - should fail
	synthErr := orchestrator.SynthesizeBook(ctx, "book_test_002", "test-tts")
	if synthErr == nil {
		t.Fatal("Expected error when synthesizing non-ready book")
	}

	if !strings.Contains(synthErr.Error(), "not ready for synthesis") {
		t.Errorf("Expected 'not ready for synthesis' error, got: %v", synthErr)
	}
}

// mockTTSProvider is a mock TTS provider for testing
type mockTTSProvider struct {
	name      string
	shouldFail bool
}

func (m *mockTTSProvider) Name() string {
	return m.name
}

func (m *mockTTSProvider) Synthesize(ctx context.Context, req provider.TTSRequest) (*provider.TTSResponse, error) {
	if m.shouldFail {
		return nil, io.ErrUnexpectedEOF
	}

	return &provider.TTSResponse{
		AudioData: []byte("MOCK_AUDIO_DATA"),
		Format:    "wav",
		Timestamps: []provider.WordTimestamp{
			{Word: "test", Start: 0.0, End: 0.5},
		},
	}, nil
}

func (m *mockTTSProvider) Close() error {
	return nil
}
