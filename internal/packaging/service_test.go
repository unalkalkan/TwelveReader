package packaging

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/book"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func TestService_PackageBook(t *testing.T) {
	ctx := context.Background()

	// Setup test storage
	storageAdapter, err := storage.NewLocalAdapter("/tmp/test-packaging")
	if err != nil {
		t.Fatalf("Failed to create storage adapter: %v", err)
	}
	defer storageAdapter.Close()

	// Setup test repository
	repo := book.NewRepository(storageAdapter)

	// Create a test book
	testBook := &types.Book{
		ID:            "book_pkg_001",
		Title:         "Test Package Book",
		Author:        "Test Author",
		Language:      "en",
		UploadedAt:    time.Now(),
		Status:        "synthesized",
		TotalChapters: 1,
		TotalSegments: 2,
	}
	if err := repo.SaveBook(ctx, testBook); err != nil {
		t.Fatalf("Failed to save book: %v", err)
	}

	// Create test chapter
	testChapter := &types.Chapter{
		ID:      "chapter_001",
		BookID:  "book_pkg_001",
		Number:  1,
		Title:   "Chapter One",
		TOCPath: []string{"Chapter 1"},
		Paragraphs: []string{
			"First paragraph.",
			"Second paragraph.",
		},
	}
	if err := repo.SaveChapter(ctx, testChapter); err != nil {
		t.Fatalf("Failed to save chapter: %v", err)
	}

	// Create test segments with timestamps
	segments := []*types.Segment{
		{
			ID:               "seg_001",
			BookID:           "book_pkg_001",
			Chapter:          "chapter_001",
			TOCPath:          []string{"Chapter 1"},
			Text:             "First segment.",
			Language:         "en",
			Person:           "narrator",
			VoiceDescription: "neutral",
			VoiceID:          "voice_1",
			Timestamps: &types.TimestampData{
				Precision: "word",
				Items: []types.TimestampItem{
					{Word: "First", Start: 0.0, End: 0.3},
					{Word: "segment", Start: 0.3, End: 0.8},
				},
			},
			Processing: &types.ProcessingInfo{
				SegmenterVersion: "v1",
				TTSProvider:      "test-tts",
				GeneratedAt:      time.Now(),
			},
		},
		{
			ID:               "seg_002",
			BookID:           "book_pkg_001",
			Chapter:          "chapter_001",
			TOCPath:          []string{"Chapter 1"},
			Text:             "Second segment.",
			Language:         "en",
			Person:           "narrator",
			VoiceDescription: "neutral",
			VoiceID:          "voice_1",
			Timestamps: &types.TimestampData{
				Precision: "word",
				Items: []types.TimestampItem{
					{Word: "Second", Start: 0.0, End: 0.4},
					{Word: "segment", Start: 0.4, End: 0.9},
				},
			},
			Processing: &types.ProcessingInfo{
				SegmenterVersion: "v1",
				TTSProvider:      "test-tts",
				GeneratedAt:      time.Now(),
			},
		},
	}

	for _, seg := range segments {
		if err := repo.SaveSegment(ctx, seg); err != nil {
			t.Fatalf("Failed to save segment: %v", err)
		}

		// Create mock audio file
		audioPath := "books/book_pkg_001/audio/" + seg.ID + ".wav"
		if err := storageAdapter.Put(ctx, audioPath, bytes.NewReader([]byte("MOCK_AUDIO"))); err != nil {
			t.Fatalf("Failed to save audio: %v", err)
		}
	}

	// Create voice map
	voiceMap := &types.VoiceMap{
		BookID: "book_pkg_001",
		Persons: []types.PersonVoice{
			{ID: "narrator", ProviderVoice: "voice_1"},
		},
	}
	if err := repo.SaveVoiceMap(ctx, voiceMap); err != nil {
		t.Fatalf("Failed to save voice map: %v", err)
	}

	// Create packaging service
	service := NewService(repo, storageAdapter)

	// Package the book
	zipReader, err := service.PackageBook(ctx, "book_pkg_001")
	if err != nil {
		t.Fatalf("Failed to package book: %v", err)
	}

	// Read ZIP into memory
	zipData, err := io.ReadAll(zipReader)
	if err != nil {
		t.Fatalf("Failed to read ZIP: %v", err)
	}

	// Verify ZIP contents
	reader := bytes.NewReader(zipData)
	zipFile, err := zip.NewReader(reader, int64(len(zipData)))
	if err != nil {
		t.Fatalf("Failed to open ZIP: %v", err)
	}

	// Check for required files
	requiredFiles := map[string]bool{
		"manifest.json":  false,
		"toc.json":       false,
		"voice-map.json": false,
	}

	for _, f := range zipFile.File {
		if _, ok := requiredFiles[f.Name]; ok {
			requiredFiles[f.Name] = true
		}

		// Verify manifest.json
		if f.Name == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open manifest: %v", err)
			}
			defer rc.Close()

			var manifest Manifest
			if err := json.NewDecoder(rc).Decode(&manifest); err != nil {
				t.Fatalf("Failed to decode manifest: %v", err)
			}

			if manifest.BookID != "book_pkg_001" {
				t.Errorf("Expected BookID 'book_pkg_001', got '%s'", manifest.BookID)
			}
			if manifest.Title != "Test Package Book" {
				t.Errorf("Expected title 'Test Package Book', got '%s'", manifest.Title)
			}
		}

		// Verify toc.json
		if f.Name == "toc.json" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("Failed to open toc: %v", err)
			}
			defer rc.Close()

			var toc TOC
			if err := json.NewDecoder(rc).Decode(&toc); err != nil {
				t.Fatalf("Failed to decode TOC: %v", err)
			}

			if len(toc.Chapters) != 1 {
				t.Errorf("Expected 1 chapter in TOC, got %d", len(toc.Chapters))
			}
		}
	}

	// Check all required files are present
	for file, found := range requiredFiles {
		if !found {
			t.Errorf("Required file '%s' not found in ZIP", file)
		}
	}
}

func TestService_PackageBook_NotSynthesized(t *testing.T) {
	ctx := context.Background()

	// Setup test storage
	storageAdapter, err := storage.NewLocalAdapter("/tmp/test-packaging-not-synth")
	if err != nil {
		t.Fatalf("Failed to create storage adapter: %v", err)
	}
	defer storageAdapter.Close()

	// Setup test repository
	repo := book.NewRepository(storageAdapter)

	// Create a test book with wrong status
	testBook := &types.Book{
		ID:     "book_pkg_002",
		Status: "ready",
	}
	if err := repo.SaveBook(ctx, testBook); err != nil {
		t.Fatalf("Failed to save book: %v", err)
	}

	// Create packaging service
	service := NewService(repo, storageAdapter)

	// Try to package book - should fail
	_, pkgErr := service.PackageBook(ctx, "book_pkg_002")
	if pkgErr == nil {
		t.Fatal("Expected error when packaging non-synthesized book")
	}
}

func TestGenerateManifest(t *testing.T) {
	service := &Service{}

	testBook := &types.Book{
		ID:       "book_test",
		Title:    "Test Book",
		Author:   "Test Author",
		Language: "en",
	}

	segments := []*types.Segment{
		{
			Timestamps: &types.TimestampData{
				Precision: "word",
				Items: []types.TimestampItem{
					{Word: "test", Start: 0.0, End: 1.0},
				},
			},
		},
		{
			Timestamps: &types.TimestampData{
				Precision: "word",
				Items: []types.TimestampItem{
					{Word: "another", Start: 0.0, End: 2.0},
				},
			},
		},
	}

	manifest := service.generateManifest(testBook, segments)

	if manifest.BookID != "book_test" {
		t.Errorf("Expected BookID 'book_test', got '%s'", manifest.BookID)
	}

	if manifest.TotalDuration != 3.0 {
		t.Errorf("Expected total duration 3.0, got %f", manifest.TotalDuration)
	}
}
