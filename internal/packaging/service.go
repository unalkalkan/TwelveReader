package packaging

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/book"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/internal/util"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// Service handles book packaging into ZIP archives
type Service struct {
	bookRepo book.Repository
	storage  storage.Adapter
}

// NewService creates a new packaging service
func NewService(bookRepo book.Repository, storage storage.Adapter) *Service {
	return &Service{
		bookRepo: bookRepo,
		storage:  storage,
	}
}

// Manifest represents the top-level book manifest
type Manifest struct {
	BookID       string    `json:"book_id"`
	Title        string    `json:"title"`
	Author       string    `json:"author"`
	Language     string    `json:"language"`
	TotalDuration float64  `json:"total_duration_seconds"`
	CreatedAt    time.Time `json:"created_at"`
	Version      string    `json:"version"`
}

// TOC represents the table of contents
type TOC struct {
	Chapters []TOCChapter `json:"chapters"`
}

// TOCChapter represents a chapter in the TOC
type TOCChapter struct {
	ID        string   `json:"id"`
	Title     string   `json:"title"`
	TOCPath   []string `json:"toc_path"`
	Segments  []string `json:"segments"` // Segment IDs
	StartTime float64  `json:"start_time_seconds"`
	Duration  float64  `json:"duration_seconds"`
}

// PackageBook creates a ZIP archive for a book
func (s *Service) PackageBook(ctx context.Context, bookID string) (io.Reader, error) {
	// Get book metadata
	book, err := s.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get book: %w", err)
	}

	// Check if book is synthesized
	if book.Status != "synthesized" {
		return nil, fmt.Errorf("book is not synthesized (status: %s)", book.Status)
	}

	// Get segments
	segments, err := s.bookRepo.ListSegments(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to list segments: %w", err)
	}

	// Get chapters
	chapters, err := s.bookRepo.ListChapters(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to list chapters: %w", err)
	}

	// Get voice map
	voiceMap, err := s.bookRepo.GetVoiceMap(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to get voice map: %w", err)
	}

	// Create ZIP in memory
	buf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(buf)

	// Generate manifest
	manifest := s.generateManifest(book, segments)
	if err := s.addJSONFile(zipWriter, "manifest.json", manifest); err != nil {
		return nil, fmt.Errorf("failed to add manifest: %w", err)
	}

	// Generate TOC
	toc := s.generateTOC(chapters, segments)
	if err := s.addJSONFile(zipWriter, "toc.json", toc); err != nil {
		return nil, fmt.Errorf("failed to add toc: %w", err)
	}

	// Add voice map
	if err := s.addJSONFile(zipWriter, "voice-map.json", voiceMap); err != nil {
		return nil, fmt.Errorf("failed to add voice-map: %w", err)
	}

	// Add segments (metadata + audio)
	for i, segment := range segments {
		// Shard segments into directories (100 per folder)
		shardDir := fmt.Sprintf("segments/%03d", i/100)

		// Add segment metadata
		metadataPath := filepath.Join(shardDir, fmt.Sprintf("%s.json", segment.ID))
		if err := s.addJSONFile(zipWriter, metadataPath, segment); err != nil {
			return nil, fmt.Errorf("failed to add segment metadata %s: %w", segment.ID, err)
		}

		// Add audio file if it exists
		var audioPath string
		var audioReader io.ReadCloser
		var err error
		
		// Try different audio formats
		for _, format := range util.AudioFormats() {
			audioPath = util.GetAudioPath(bookID, segment.ID, format)
			audioReader, err = s.storage.Get(ctx, audioPath)
			if err == nil {
				break
			}
		}

		if err == nil {
			audioZipPath := filepath.Join(shardDir, filepath.Base(audioPath))
			if err := s.addFileFromReader(zipWriter, audioZipPath, audioReader); err != nil {
				audioReader.Close()
				return nil, fmt.Errorf("failed to add audio %s: %w", segment.ID, err)
			}
			audioReader.Close()
		}
	}

	// Close ZIP writer
	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close zip: %w", err)
	}

	return bytes.NewReader(buf.Bytes()), nil
}

// generateManifest creates the manifest file
func (s *Service) generateManifest(book *types.Book, segments []*types.Segment) *Manifest {
	// Calculate total duration
	var totalDuration float64
	for _, seg := range segments {
		if seg.Timestamps != nil && len(seg.Timestamps.Items) > 0 {
			lastItem := seg.Timestamps.Items[len(seg.Timestamps.Items)-1]
			totalDuration += lastItem.End
		}
	}

	return &Manifest{
		BookID:        book.ID,
		Title:         book.Title,
		Author:        book.Author,
		Language:      book.Language,
		TotalDuration: totalDuration,
		CreatedAt:     time.Now(),
		Version:       "1.0",
	}
}

// generateTOC creates the table of contents
func (s *Service) generateTOC(chapters []*types.Chapter, segments []*types.Segment) *TOC {
	toc := &TOC{
		Chapters: make([]TOCChapter, 0, len(chapters)),
	}

	// Group segments by chapter
	segmentsByChapter := make(map[string][]*types.Segment)
	for _, seg := range segments {
		segmentsByChapter[seg.Chapter] = append(segmentsByChapter[seg.Chapter], seg)
	}

	// Build TOC chapters
	currentTime := 0.0
	for _, chapter := range chapters {
		chapterSegs := segmentsByChapter[chapter.ID]
		segIDs := make([]string, len(chapterSegs))
		chapterDuration := 0.0

		for i, seg := range chapterSegs {
			segIDs[i] = seg.ID
			if seg.Timestamps != nil && len(seg.Timestamps.Items) > 0 {
				lastItem := seg.Timestamps.Items[len(seg.Timestamps.Items)-1]
				chapterDuration += lastItem.End
			}
		}

		tocChapter := TOCChapter{
			ID:        chapter.ID,
			Title:     chapter.Title,
			TOCPath:   chapter.TOCPath,
			Segments:  segIDs,
			StartTime: currentTime,
			Duration:  chapterDuration,
		}

		toc.Chapters = append(toc.Chapters, tocChapter)
		currentTime += chapterDuration
	}

	return toc
}

// addJSONFile adds a JSON file to the ZIP
func (s *Service) addJSONFile(zipWriter *zip.Writer, path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	writer, err := zipWriter.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}

	if _, err := writer.Write(jsonData); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}

	return nil
}

// addFileFromReader adds a file from an io.Reader to the ZIP
func (s *Service) addFileFromReader(zipWriter *zip.Writer, path string, reader io.Reader) error {
	writer, err := zipWriter.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create zip entry: %w", err)
	}

	if _, err := io.Copy(writer, reader); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	return nil
}
