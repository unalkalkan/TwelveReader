package book

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// Repository handles book metadata persistence
type Repository interface {
	// SaveBook stores book metadata
	SaveBook(ctx context.Context, book *types.Book) error

	// GetBook retrieves book metadata by ID
	GetBook(ctx context.Context, bookID string) (*types.Book, error)

	// UpdateBook updates book metadata
	UpdateBook(ctx context.Context, book *types.Book) error

	// ListBooks returns all books
	ListBooks(ctx context.Context) ([]*types.Book, error)

	// SaveChapter stores chapter data
	SaveChapter(ctx context.Context, chapter *types.Chapter) error

	// GetChapter retrieves chapter by ID
	GetChapter(ctx context.Context, bookID, chapterID string) (*types.Chapter, error)

	// ListChapters returns all chapters for a book
	ListChapters(ctx context.Context, bookID string) ([]*types.Chapter, error)

	// SaveSegment stores segment metadata
	SaveSegment(ctx context.Context, segment *types.Segment) error

	// GetSegment retrieves segment by ID
	GetSegment(ctx context.Context, bookID, segmentID string) (*types.Segment, error)

	// ListSegments returns all segments for a book
	ListSegments(ctx context.Context, bookID string) ([]*types.Segment, error)

	// SaveVoiceMap stores voice mapping
	SaveVoiceMap(ctx context.Context, voiceMap *types.VoiceMap) error

	// GetVoiceMap retrieves voice mapping for a book
	GetVoiceMap(ctx context.Context, bookID string) (*types.VoiceMap, error)

	// SaveRawFile stores the uploaded raw file
	SaveRawFile(ctx context.Context, bookID string, data []byte, format string) error

	// GetRawFile retrieves the uploaded raw file
	GetRawFile(ctx context.Context, bookID string) ([]byte, string, error)
}

// StorageRepository implements Repository using a storage adapter
type StorageRepository struct {
	storage storage.Adapter
}

// NewRepository creates a new book repository
func NewRepository(storageAdapter storage.Adapter) Repository {
	return &StorageRepository{
		storage: storageAdapter,
	}
}

// SaveBook stores book metadata
func (r *StorageRepository) SaveBook(ctx context.Context, book *types.Book) error {
	data, err := json.Marshal(book)
	if err != nil {
		return fmt.Errorf("failed to marshal book: %w", err)
	}

	path := filepath.Join("books", book.ID, "metadata.json")
	return r.storage.Put(ctx, path, bytesReader(data))
}

// GetBook retrieves book metadata by ID
func (r *StorageRepository) GetBook(ctx context.Context, bookID string) (*types.Book, error) {
	path := filepath.Join("books", bookID, "metadata.json")
	reader, err := r.storage.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get book metadata: %w", err)
	}
	defer reader.Close()

	var book types.Book
	if err := json.NewDecoder(reader).Decode(&book); err != nil {
		return nil, fmt.Errorf("failed to decode book metadata: %w", err)
	}

	return &book, nil
}

// UpdateBook updates book metadata
func (r *StorageRepository) UpdateBook(ctx context.Context, book *types.Book) error {
	return r.SaveBook(ctx, book)
}

// ListBooks returns all books
func (r *StorageRepository) ListBooks(ctx context.Context) ([]*types.Book, error) {
	paths, err := r.storage.List(ctx, "books/")
	if err != nil {
		return nil, fmt.Errorf("failed to list books: %w", err)
	}

	books := make([]*types.Book, 0)
	for _, path := range paths {
		// Only process metadata.json files
		if filepath.Base(path) != "metadata.json" {
			continue
		}

		reader, err := r.storage.Get(ctx, path)
		if err != nil {
			continue // Skip books that can't be read
		}

		var book types.Book
		if err := json.NewDecoder(reader).Decode(&book); err != nil {
			reader.Close()
			continue
		}
		reader.Close()

		books = append(books, &book)
	}

	return books, nil
}

// SaveChapter stores chapter data
func (r *StorageRepository) SaveChapter(ctx context.Context, chapter *types.Chapter) error {
	data, err := json.Marshal(chapter)
	if err != nil {
		return fmt.Errorf("failed to marshal chapter: %w", err)
	}

	path := filepath.Join("books", chapter.BookID, "chapters", fmt.Sprintf("%s.json", chapter.ID))
	return r.storage.Put(ctx, path, bytesReader(data))
}

// GetChapter retrieves chapter by ID
func (r *StorageRepository) GetChapter(ctx context.Context, bookID, chapterID string) (*types.Chapter, error) {
	path := filepath.Join("books", bookID, "chapters", fmt.Sprintf("%s.json", chapterID))
	reader, err := r.storage.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get chapter: %w", err)
	}
	defer reader.Close()

	var chapter types.Chapter
	if err := json.NewDecoder(reader).Decode(&chapter); err != nil {
		return nil, fmt.Errorf("failed to decode chapter: %w", err)
	}

	return &chapter, nil
}

// ListChapters returns all chapters for a book
func (r *StorageRepository) ListChapters(ctx context.Context, bookID string) ([]*types.Chapter, error) {
	prefix := filepath.Join("books", bookID, "chapters/")
	paths, err := r.storage.List(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list chapters: %w", err)
	}

	chapters := make([]*types.Chapter, 0, len(paths))
	for _, path := range paths {
		reader, err := r.storage.Get(ctx, path)
		if err != nil {
			continue
		}

		var chapter types.Chapter
		if err := json.NewDecoder(reader).Decode(&chapter); err != nil {
			reader.Close()
			continue
		}
		reader.Close()

		chapters = append(chapters, &chapter)
	}

	return chapters, nil
}

// SaveSegment stores segment metadata
func (r *StorageRepository) SaveSegment(ctx context.Context, segment *types.Segment) error {
	data, err := json.Marshal(segment)
	if err != nil {
		return fmt.Errorf("failed to marshal segment: %w", err)
	}

	path := filepath.Join("books", segment.BookID, "segments", fmt.Sprintf("%s.json", segment.ID))
	return r.storage.Put(ctx, path, bytesReader(data))
}

// GetSegment retrieves segment by ID
func (r *StorageRepository) GetSegment(ctx context.Context, bookID, segmentID string) (*types.Segment, error) {
	path := filepath.Join("books", bookID, "segments", fmt.Sprintf("%s.json", segmentID))
	reader, err := r.storage.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get segment: %w", err)
	}
	defer reader.Close()

	var segment types.Segment
	if err := json.NewDecoder(reader).Decode(&segment); err != nil {
		return nil, fmt.Errorf("failed to decode segment: %w", err)
	}

	return &segment, nil
}

// ListSegments returns all segments for a book
func (r *StorageRepository) ListSegments(ctx context.Context, bookID string) ([]*types.Segment, error) {
	prefix := filepath.Join("books", bookID, "segments/")
	paths, err := r.storage.List(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list segments: %w", err)
	}

	segments := make([]*types.Segment, 0, len(paths))
	for _, path := range paths {
		reader, err := r.storage.Get(ctx, path)
		if err != nil {
			continue
		}

		var segment types.Segment
		if err := json.NewDecoder(reader).Decode(&segment); err != nil {
			reader.Close()
			continue
		}
		reader.Close()

		segments = append(segments, &segment)
	}

	return segments, nil
}

// SaveVoiceMap stores voice mapping
func (r *StorageRepository) SaveVoiceMap(ctx context.Context, voiceMap *types.VoiceMap) error {
	data, err := json.Marshal(voiceMap)
	if err != nil {
		return fmt.Errorf("failed to marshal voice map: %w", err)
	}

	path := filepath.Join("books", voiceMap.BookID, "voice-map.json")
	return r.storage.Put(ctx, path, bytesReader(data))
}

// GetVoiceMap retrieves voice mapping for a book
func (r *StorageRepository) GetVoiceMap(ctx context.Context, bookID string) (*types.VoiceMap, error) {
	path := filepath.Join("books", bookID, "voice-map.json")
	reader, err := r.storage.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to get voice map: %w", err)
	}
	defer reader.Close()

	var voiceMap types.VoiceMap
	if err := json.NewDecoder(reader).Decode(&voiceMap); err != nil {
		return nil, fmt.Errorf("failed to decode voice map: %w", err)
	}

	return &voiceMap, nil
}

// SaveRawFile stores the uploaded raw file
func (r *StorageRepository) SaveRawFile(ctx context.Context, bookID string, data []byte, format string) error {
	path := filepath.Join("books", bookID, fmt.Sprintf("raw.%s", format))
	return r.storage.Put(ctx, path, bytesReader(data))
}

// GetRawFile retrieves the uploaded raw file
func (r *StorageRepository) GetRawFile(ctx context.Context, bookID string) ([]byte, string, error) {
	// Try different formats
	formats := []string{"pdf", "epub", "txt"}
	for _, format := range formats {
		path := filepath.Join("books", bookID, fmt.Sprintf("raw.%s", format))
		exists, err := r.storage.Exists(ctx, path)
		if err != nil || !exists {
			continue
		}

		reader, err := r.storage.Get(ctx, path)
		if err != nil {
			continue
		}
		defer reader.Close()

		// Read all data
		data := make([]byte, 0)
		buf := make([]byte, 32*1024)
		for {
			n, err := reader.Read(buf)
			if n > 0 {
				data = append(data, buf[:n]...)
			}
			if err != nil {
				break
			}
		}

		return data, format, nil
	}

	return nil, "", fmt.Errorf("raw file not found")
}

// bytesReader wraps a byte slice in a bytes.Reader for storage adapter
func bytesReader(data []byte) *bytesReaderWrapper {
	return &bytesReaderWrapper{data: data, pos: 0}
}

type bytesReaderWrapper struct {
	data []byte
	pos  int
}

func (b *bytesReaderWrapper) Read(p []byte) (n int, err error) {
	if b.pos >= len(b.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, b.data[b.pos:])
	b.pos += n
	return n, nil
}
