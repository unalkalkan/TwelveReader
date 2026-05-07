package book

import (
	"bytes"
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

func TestPersonaProfileRepository(t *testing.T) {
	tempDir := t.TempDir()
	storageAdapter, err := storage.NewLocalAdapter(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage adapter: %v", err)
	}
	defer storageAdapter.Close()

	repo := NewRepository(storageAdapter)
	ctx := context.Background()

	t.Run("SaveAndGetPersonaProfiles", func(t *testing.T) {
		profiles := []*types.PersonaProfile{
			{
				BookID:           "book_persona_1",
				PersonaID:        "narrator",
				DisplayName:      "Narrator",
				VoiceDescription: "neutral storytelling voice",
				SegmentCount:     5,
				UpdatedAt:        time.Now().Truncate(time.Millisecond),
			},
			{
				BookID:           "book_persona_1",
				PersonaID:        "alice",
				DisplayName:      "Alice",
				VoiceDescription: "young female voice",
				SegmentCount:     3,
				UpdatedAt:        time.Now().Truncate(time.Millisecond),
			},
		}

		err := repo.SavePersonaProfiles(ctx, "book_persona_1", profiles)
		if err != nil {
			t.Fatalf("Failed to save persona profiles: %v", err)
		}

		retrieved, err := repo.GetPersonaProfiles(ctx, "book_persona_1")
		if err != nil {
			t.Fatalf("Failed to get persona profiles: %v", err)
		}

		if len(retrieved) != 2 {
			t.Fatalf("Expected 2 profiles, got %d", len(retrieved))
		}

		foundNarrator := false
		foundAlice := false
		for _, p := range retrieved {
			if p.PersonaID == "narrator" {
				foundNarrator = true
				if p.DisplayName != "Narrator" {
					t.Errorf("Narrator display name mismatch: got %s", p.DisplayName)
				}
				if p.VoiceDescription != "neutral storytelling voice" {
					t.Errorf("Narrator voice description mismatch: got %s", p.VoiceDescription)
				}
				if p.SegmentCount != 5 {
					t.Errorf("Narrator segment count mismatch: got %d", p.SegmentCount)
				}
			}
			if p.PersonaID == "alice" {
				foundAlice = true
				if p.SegmentCount != 3 {
					t.Errorf("Alice segment count mismatch: got %d", p.SegmentCount)
				}
			}
		}
		if !foundNarrator {
			t.Error("Narrator profile not found")
		}
		if !foundAlice {
			t.Error("Alice profile not found")
		}
	})

	t.Run("GetPersonaProfilesNonExistent", func(t *testing.T) {
		profiles, err := repo.GetPersonaProfiles(ctx, "nonexistent_book_xyz")
		if err != nil {
			t.Fatalf("Expected no error for non-existent book, got: %v", err)
		}
		if len(profiles) != 0 {
			t.Errorf("Expected empty profiles for non-existent book, got %d", len(profiles))
		}
	})

	t.Run("GetPersonaProfilesInvalidJSONReturnsError", func(t *testing.T) {
		if err := storageAdapter.Put(ctx, "books/book_bad_personas/personas.json", bytes.NewReader([]byte("not-json"))); err != nil {
			t.Fatalf("Failed to seed invalid personas file: %v", err)
		}

		_, err := repo.GetPersonaProfiles(ctx, "book_bad_personas")
		if err == nil {
			t.Fatal("Expected invalid personas JSON to return an error")
		}
	})

	t.Run("UpdatePersonaProfilesFromSegments_New", func(t *testing.T) {
		segments := []*types.Segment{
			{
				ID:               "seg_a1",
				BookID:           "book_update_new",
				Person:           "narrator",
				VoiceDescription: "calm neutral voice",
			},
			{
				ID:               "seg_a2",
				BookID:           "book_update_new",
				Person:           "alice",
				VoiceDescription: "cheerful young female",
			},
			{
				ID:               "seg_a3",
				BookID:           "book_update_new",
				Person:           "narrator",
				VoiceDescription: "calm neutral voice",
			},
		}

		err := repo.UpdatePersonaProfilesFromSegments(ctx, "book_update_new", segments)
		if err != nil {
			t.Fatalf("Failed to update persona profiles from segments: %v", err)
		}

		profiles, err := repo.GetPersonaProfiles(ctx, "book_update_new")
		if err != nil {
			t.Fatalf("Failed to get persona profiles: %v", err)
		}

		if len(profiles) != 2 {
			t.Fatalf("Expected 2 profiles, got %d", len(profiles))
		}

		for _, p := range profiles {
			if p.BookID != "book_update_new" {
				t.Errorf("BookID mismatch: got %s", p.BookID)
			}
			switch p.PersonaID {
			case "narrator":
				if p.SegmentCount != 2 {
					t.Errorf("Narrator segment count: got %d, want 2", p.SegmentCount)
				}
				if p.VoiceDescription != "calm neutral voice" {
					t.Errorf("Narrator voice description: got %s", p.VoiceDescription)
				}
				if p.DisplayName != "narrator" {
					t.Errorf("Narrator display name: got %s", p.DisplayName)
				}
			case "alice":
				if p.SegmentCount != 1 {
					t.Errorf("Alice segment count: got %d, want 1", p.SegmentCount)
				}
			}
		}
	})

	t.Run("UpdatePersonaProfilesFromSegments_MergePreservesVoice", func(t *testing.T) {
		existing := []*types.PersonaProfile{
			{
				BookID:           "book_merge",
				PersonaID:        "bob",
				DisplayName:      "Bob",
				VoiceDescription: "deep male voice",
				SegmentCount:     4,
				UpdatedAt:        time.Now().Truncate(time.Millisecond),
			},
			{
				BookID:           "book_merge",
				PersonaID:        "narrator",
				DisplayName:      "Narrator",
				VoiceDescription: "",
				SegmentCount:     1,
				UpdatedAt:        time.Now().Truncate(time.Millisecond),
			},
		}

		err := repo.SavePersonaProfiles(ctx, "book_merge", existing)
		if err != nil {
			t.Fatalf("Failed to save initial profiles: %v", err)
		}

		segments := []*types.Segment{
			{
				ID:               "seg_m1",
				BookID:           "book_merge",
				Person:           "bob",
				VoiceDescription: "different voice",
			},
			{
				ID:               "seg_m2",
				BookID:           "book_merge",
				Person:           "narrator",
				VoiceDescription: "soft neutral voice",
			},
			{
				ID:               "seg_m3",
				BookID:           "book_merge",
				Person:           "carol",
				VoiceDescription: "warm elderly female",
			},
		}

		err = repo.UpdatePersonaProfilesFromSegments(ctx, "book_merge", segments)
		if err != nil {
			t.Fatalf("Failed to merge persona profiles: %v", err)
		}

		profiles, err := repo.GetPersonaProfiles(ctx, "book_merge")
		if err != nil {
			t.Fatalf("Failed to get merged profiles: %v", err)
		}

		if len(profiles) != 3 {
			t.Fatalf("Expected 3 profiles, got %d", len(profiles))
		}

		profileMap := make(map[string]*types.PersonaProfile)
		for i := range profiles {
			profileMap[profiles[i].PersonaID] = profiles[i]
		}

		bob := profileMap["bob"]
		if bob == nil {
			t.Fatal("bob profile not found")
		}
		if bob.VoiceDescription != "deep male voice" {
			t.Errorf("Bob voice description should be preserved: got %s", bob.VoiceDescription)
		}
		if bob.SegmentCount != 5 {
			t.Errorf("Bob segment count: got %d, want 5", bob.SegmentCount)
		}

		narrator := profileMap["narrator"]
		if narrator == nil {
			t.Fatal("narrator profile not found")
		}
		if narrator.VoiceDescription != "soft neutral voice" {
			t.Errorf("Narrator voice description should be updated from empty: got %s", narrator.VoiceDescription)
		}
		if narrator.SegmentCount != 2 {
			t.Errorf("Narrator segment count: got %d, want 2", narrator.SegmentCount)
		}

		carol := profileMap["carol"]
		if carol == nil {
			t.Fatal("carol profile not found")
		}
		if carol.SegmentCount != 1 {
			t.Errorf("Carol segment count: got %d, want 1", carol.SegmentCount)
		}
	})

	t.Run("UpdatePersonaProfilesFromSegments_IgnoreEmptyPersonaID", func(t *testing.T) {
		segments := []*types.Segment{
			{
				ID:               "seg_e1",
				BookID:           "book_empty_persona",
				Person:           "narrator",
				VoiceDescription: "neutral",
			},
			{
				ID:               "seg_e2",
				BookID:           "book_empty_persona",
				Person:           "",
				VoiceDescription: "should be ignored",
			},
			{
				ID:               "seg_e3",
				BookID:           "book_empty_persona",
				Person:           "alice",
				VoiceDescription: "young female",
			},
		}

		err := repo.UpdatePersonaProfilesFromSegments(ctx, "book_empty_persona", segments)
		if err != nil {
			t.Fatalf("Failed to update profiles: %v", err)
		}

		profiles, err := repo.GetPersonaProfiles(ctx, "book_empty_persona")
		if err != nil {
			t.Fatalf("Failed to get profiles: %v", err)
		}

		if len(profiles) != 2 {
			t.Fatalf("Expected 2 profiles (empty persona ignored), got %d", len(profiles))
		}

		for _, p := range profiles {
			if p.PersonaID == "" {
				t.Error("Empty persona ID should not appear in profiles")
			}
		}
	})
}



func TestDefaultVoiceRepository(t *testing.T) {
	tempDir := t.TempDir()
	storageAdapter, err := storage.NewLocalAdapter(tempDir)
	if err != nil {
		t.Fatalf("Failed to create storage adapter: %v", err)
	}
	defer storageAdapter.Close()

	repo := NewRepository(storageAdapter)
	ctx := context.Background()

	t.Run("GetMissingDefaultVoiceReturnsNil", func(t *testing.T) {
		setting, err := repo.GetDefaultVoice(ctx)
		if err != nil {
			t.Fatalf("Expected missing default voice to be non-fatal, got: %v", err)
		}
		if setting != nil {
			t.Fatalf("Expected nil missing default voice, got %#v", setting)
		}
	})

	t.Run("SaveAndGetDefaultVoice", func(t *testing.T) {
		setting := &types.DefaultVoice{
			Provider:         "stub-tts",
			VoiceID:          "stub-voice-1",
			Language:         "en",
			VoiceDescription: "A stub voice for testing",
			UpdatedAt:        time.Now().UTC().Truncate(time.Millisecond),
		}
		if err := repo.SaveDefaultVoice(ctx, setting); err != nil {
			t.Fatalf("Failed to save default voice: %v", err)
		}

		retrieved, err := repo.GetDefaultVoice(ctx)
		if err != nil {
			t.Fatalf("Failed to get default voice: %v", err)
		}
		if retrieved == nil {
			t.Fatal("Expected default voice, got nil")
		}
		if retrieved.Provider != setting.Provider || retrieved.VoiceID != setting.VoiceID {
			t.Fatalf("Default voice mismatch: got %s/%s want %s/%s", retrieved.Provider, retrieved.VoiceID, setting.Provider, setting.VoiceID)
		}
		if retrieved.Language != setting.Language {
			t.Fatalf("Language mismatch: got %s want %s", retrieved.Language, setting.Language)
		}
		if retrieved.VoiceDescription != setting.VoiceDescription {
			t.Fatalf("VoiceDescription mismatch: got %s want %s", retrieved.VoiceDescription, setting.VoiceDescription)
		}
	})

	t.Run("PersistsAcrossRepositoryInstances", func(t *testing.T) {
		freshAdapter, err := storage.NewLocalAdapter(tempDir)
		if err != nil {
			t.Fatalf("Failed to reopen storage adapter: %v", err)
		}
		defer freshAdapter.Close()
		freshRepo := NewRepository(freshAdapter)

		retrieved, err := freshRepo.GetDefaultVoice(ctx)
		if err != nil {
			t.Fatalf("Failed to get default voice from fresh repository: %v", err)
		}
		if retrieved == nil || retrieved.Provider != "stub-tts" || retrieved.VoiceID != "stub-voice-1" {
			t.Fatalf("Default voice did not persist across repository instances: %#v", retrieved)
		}
	})
}
