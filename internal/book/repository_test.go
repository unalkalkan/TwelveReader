package book

import (
	"context"
	"testing"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func TestBookRepository(t *testing.T) {
	// Create a temporary storage adapter
	tempDir := t.TempDir()
	storageAdapter, err := storage.NewLocalAdapter(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage adapter: %v", err)
	}
	defer storageAdapter.Close()

	repo := NewRepository(storageAdapter)
	ctx := context.Background()

	t.Run("SaveAndGetBook", func(t *testing.T) {
		book := &types.Book{
			ID:         "book_123",
			Title:      "Test Book",
			Author:     "Test Author",
			Language:   "en",
			UploadedAt: time.Now(),
			Status:     "uploaded",
			OrigFormat: "txt",
		}

		// Save book
		err := repo.SaveBook(ctx, book)
		if err != nil {
			t.Fatalf("Failed to save book: %v", err)
		}

		// Get book
		retrieved, err := repo.GetBook(ctx, "book_123")
		if err != nil {
			t.Fatalf("Failed to get book: %v", err)
		}

		if retrieved.ID != book.ID {
			t.Errorf("Book ID mismatch: got %s, want %s", retrieved.ID, book.ID)
		}
		if retrieved.Title != book.Title {
			t.Errorf("Book title mismatch: got %s, want %s", retrieved.Title, book.Title)
		}
	})

	t.Run("UpdateBook", func(t *testing.T) {
		book := &types.Book{
			ID:         "book_456",
			Title:      "Original Title",
			Author:     "Test Author",
			Language:   "en",
			UploadedAt: time.Now(),
			Status:     "uploaded",
			OrigFormat: "txt",
		}

		// Save book
		err := repo.SaveBook(ctx, book)
		if err != nil {
			t.Fatalf("Failed to save book: %v", err)
		}

		// Update book
		book.Title = "Updated Title"
		book.Status = "ready"
		err = repo.UpdateBook(ctx, book)
		if err != nil {
			t.Fatalf("Failed to update book: %v", err)
		}

		// Get updated book
		retrieved, err := repo.GetBook(ctx, "book_456")
		if err != nil {
			t.Fatalf("Failed to get book: %v", err)
		}

		if retrieved.Title != "Updated Title" {
			t.Errorf("Book title not updated: got %s, want %s", retrieved.Title, "Updated Title")
		}
		if retrieved.Status != "ready" {
			t.Errorf("Book status not updated: got %s, want %s", retrieved.Status, "ready")
		}
	})

	t.Run("SaveAndGetChapter", func(t *testing.T) {
		chapter := &types.Chapter{
			ID:         "chapter_001",
			BookID:     "book_123",
			Number:     1,
			Title:      "Chapter One",
			TOCPath:    []string{"Part I", "Chapter One"},
			Paragraphs: []string{"First paragraph", "Second paragraph"},
		}

		// Save chapter
		err := repo.SaveChapter(ctx, chapter)
		if err != nil {
			t.Fatalf("Failed to save chapter: %v", err)
		}

		// Get chapter
		retrieved, err := repo.GetChapter(ctx, "book_123", "chapter_001")
		if err != nil {
			t.Fatalf("Failed to get chapter: %v", err)
		}

		if retrieved.ID != chapter.ID {
			t.Errorf("Chapter ID mismatch: got %s, want %s", retrieved.ID, chapter.ID)
		}
		if len(retrieved.Paragraphs) != 2 {
			t.Errorf("Paragraph count mismatch: got %d, want 2", len(retrieved.Paragraphs))
		}
	})

	t.Run("SaveAndGetSegment", func(t *testing.T) {
		segment := &types.Segment{
			ID:               "seg_00001",
			BookID:           "book_123",
			Chapter:          "chapter_001",
			TOCPath:          []string{"Chapter One"},
			Text:             "Test segment text",
			Language:         "en",
			Person:           "narrator",
			VoiceDescription: "neutral",
			Processing: &types.ProcessingInfo{
				SegmenterVersion: "v1",
				GeneratedAt:      time.Now(),
			},
		}

		// Save segment
		err := repo.SaveSegment(ctx, segment)
		if err != nil {
			t.Fatalf("Failed to save segment: %v", err)
		}

		// Get segment
		retrieved, err := repo.GetSegment(ctx, "book_123", "seg_00001")
		if err != nil {
			t.Fatalf("Failed to get segment: %v", err)
		}

		if retrieved.ID != segment.ID {
			t.Errorf("Segment ID mismatch: got %s, want %s", retrieved.ID, segment.ID)
		}
		if retrieved.Text != segment.Text {
			t.Errorf("Segment text mismatch: got %s, want %s", retrieved.Text, segment.Text)
		}
	})

	t.Run("SaveAndGetVoiceMap", func(t *testing.T) {
		voiceMap := &types.VoiceMap{
			BookID: "book_123",
			Persons: []types.PersonVoice{
				{ID: "narrator", ProviderVoice: "voice_1"},
				{ID: "alice", ProviderVoice: "voice_2"},
			},
		}

		// Save voice map
		err := repo.SaveVoiceMap(ctx, voiceMap)
		if err != nil {
			t.Fatalf("Failed to save voice map: %v", err)
		}

		// Get voice map
		retrieved, err := repo.GetVoiceMap(ctx, "book_123")
		if err != nil {
			t.Fatalf("Failed to get voice map: %v", err)
		}

		if retrieved.BookID != voiceMap.BookID {
			t.Errorf("Voice map BookID mismatch: got %s, want %s", retrieved.BookID, voiceMap.BookID)
		}
		if len(retrieved.Persons) != 2 {
			t.Errorf("Persons count mismatch: got %d, want 2", len(retrieved.Persons))
		}
	})

	t.Run("ListChapters", func(t *testing.T) {
		// Save multiple chapters
		for i := 1; i <= 3; i++ {
			chapter := &types.Chapter{
				ID:         "chapter_" + string(rune('0'+i)),
				BookID:     "book_789",
				Number:     i,
				Title:      "Chapter " + string(rune('0'+i)),
				Paragraphs: []string{"Paragraph 1"},
			}
			err := repo.SaveChapter(ctx, chapter)
			if err != nil {
				t.Fatalf("Failed to save chapter %d: %v", i, err)
			}
		}

		// List chapters
		chapters, err := repo.ListChapters(ctx, "book_789")
		if err != nil {
			t.Fatalf("Failed to list chapters: %v", err)
		}

		if len(chapters) < 3 {
			t.Errorf("Expected at least 3 chapters, got %d", len(chapters))
		}
	})

	t.Run("GetNonExistentBook", func(t *testing.T) {
		_, err := repo.GetBook(ctx, "nonexistent_book")
		if err == nil {
			t.Error("Expected error for non-existent book")
		}
	})
}
