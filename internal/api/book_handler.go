package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/book"
	"github.com/unalkalkan/TwelveReader/internal/packaging"
	"github.com/unalkalkan/TwelveReader/internal/parser"
	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/internal/segmentation"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/internal/streaming"
	"github.com/unalkalkan/TwelveReader/internal/tts"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// BookHandler handles book-related API endpoints
type BookHandler struct {
	repo            book.Repository
	parserFactory   parser.Factory
	providerReg     *provider.Registry
	ttsOrchestrator *tts.Orchestrator
	packagingService *packaging.Service
	streamingService *streaming.Service
	storage         storage.Adapter
}

// NewBookHandler creates a new book handler
func NewBookHandler(repo book.Repository, parserFactory parser.Factory, providerReg *provider.Registry, storage storage.Adapter) *BookHandler {
	return &BookHandler{
		repo:            repo,
		parserFactory:   parserFactory,
		providerReg:     providerReg,
		ttsOrchestrator: tts.NewOrchestrator(providerReg, repo, storage, 3),
		packagingService: packaging.NewService(repo, storage),
		streamingService: streaming.NewService(repo),
		storage:         storage,
	}
}

// UploadBook handles POST /api/v1/books
func (h *BookHandler) UploadBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse multipart form (max 100MB)
	if err := r.ParseMultipartForm(100 << 20); err != nil {
		respondError(w, "Failed to parse form", http.StatusBadRequest)
		return
	}

	// Get file from form
	file, header, err := r.FormFile("file")
	if err != nil {
		respondError(w, "No file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Get metadata
	title := r.FormValue("title")
	author := r.FormValue("author")
	language := r.FormValue("language")
	if language == "" {
		language = "en"
	}

	// Detect format from filename
	ext := strings.ToLower(filepath.Ext(header.Filename))
	format := strings.TrimPrefix(ext, ".")
	if format == "" {
		respondError(w, "Could not detect file format", http.StatusBadRequest)
		return
	}

	// Validate format
	if _, err := h.parserFactory.GetParser(format); err != nil {
		respondError(w, fmt.Sprintf("Unsupported format: %s", format), http.StatusBadRequest)
		return
	}

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		respondError(w, "Failed to read file", http.StatusInternalServerError)
		return
	}

	// Generate book ID
	bookID := fmt.Sprintf("book_%d", time.Now().UnixNano())

	// Create book metadata
	newBook := &types.Book{
		ID:         bookID,
		Title:      title,
		Author:     author,
		Language:   language,
		UploadedAt: time.Now(),
		Status:     "uploaded",
		OrigFormat: format,
	}

	// Save book metadata
	ctx := r.Context()
	if err := h.repo.SaveBook(ctx, newBook); err != nil {
		respondError(w, "Failed to save book metadata", http.StatusInternalServerError)
		return
	}

	// Save raw file
	if err := h.repo.SaveRawFile(ctx, bookID, data, format); err != nil {
		respondError(w, "Failed to save raw file", http.StatusInternalServerError)
		return
	}

	// Start async processing with proper error handling
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("Panic in book processing for %s: %v", bookID, r)
				h.updateBookError(context.Background(), bookID, fmt.Sprintf("Processing panic: %v", r))
			}
		}()
		h.processBook(bookID, data, format)
	}()

	// Return success
	respondJSON(w, newBook, http.StatusCreated)
}

// processBook handles async book processing
func (h *BookHandler) processBook(bookID string, data []byte, format string) {
	ctx := context.Background()

	// Update status to parsing
	book, _ := h.repo.GetBook(ctx, bookID)
	if book != nil {
		book.Status = "parsing"
		h.repo.UpdateBook(ctx, book)
	}

	// Parse the book
	parser, err := h.parserFactory.GetParser(format)
	if err != nil {
		h.updateBookError(ctx, bookID, fmt.Sprintf("Parser error: %v", err))
		return
	}

	chapters, err := parser.Parse(ctx, data)
	if err != nil {
		h.updateBookError(ctx, bookID, fmt.Sprintf("Parse failed: %v", err))
		return
	}

	// Save chapters
	for i, chapter := range chapters {
		chapter.BookID = bookID
		chapter.Number = i + 1
		if err := h.repo.SaveChapter(ctx, chapter); err != nil {
			log.Printf("Failed to save chapter %s: %v", chapter.ID, err)
		}
	}

	// Update book with chapter count
	if book != nil {
		book.TotalChapters = len(chapters)
		book.Status = "segmenting"
		h.repo.UpdateBook(ctx, book)
	}

	// Segment chapters using LLM
	llmProviders := h.providerReg.ListLLM()
	if len(llmProviders) > 0 {
		llmProvider, err := h.providerReg.GetLLM(llmProviders[0])
		if err == nil && llmProvider != nil {
			segService := segmentation.NewService(llmProvider, 2)
			segments, err := segService.SegmentChapters(ctx, bookID, chapters)
			if err != nil {
				log.Printf("Segmentation failed: %v", err)
			} else {
				// Save segments
				for _, segment := range segments {
					if err := h.repo.SaveSegment(ctx, segment); err != nil {
						log.Printf("Failed to save segment %s: %v", segment.ID, err)
					}
				}

				// Update book with segment count
				if book != nil {
					book.TotalSegments = len(segments)
					book.Status = "voice_mapping"
					h.repo.UpdateBook(ctx, book)
				}
			}
		}
	}

	// If no LLM provider, mark as ready
	if len(llmProviders) == 0 {
		if book != nil {
			book.Status = "ready"
			h.repo.UpdateBook(ctx, book)
		}
	}
}

// updateBookError updates book with error status
func (h *BookHandler) updateBookError(ctx context.Context, bookID, errorMsg string) {
	book, err := h.repo.GetBook(ctx, bookID)
	if err == nil && book != nil {
		book.Status = "error"
		book.Error = errorMsg
		h.repo.UpdateBook(ctx, book)
	}
}

// GetBook handles GET /api/v1/books/:id
func (h *BookHandler) GetBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract book ID from path
	bookID := extractIDFromPath(r.URL.Path, "/api/v1/books/")
	if bookID == "" {
		respondError(w, "Book ID required", http.StatusBadRequest)
		return
	}

	// Get book from repository
	book, err := h.repo.GetBook(r.Context(), bookID)
	if err != nil {
		respondError(w, "Book not found", http.StatusNotFound)
		return
	}

	respondJSON(w, book, http.StatusOK)
}

// GetBookStatus handles GET /api/v1/books/:id/status
func (h *BookHandler) GetBookStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract book ID from path
	path := r.URL.Path
	bookID := extractIDFromPath(path, "/api/v1/books/")
	if bookID == "" {
		respondError(w, "Book ID required", http.StatusBadRequest)
		return
	}

	// Get book
	book, err := h.repo.GetBook(r.Context(), bookID)
	if err != nil {
		respondError(w, "Book not found", http.StatusNotFound)
		return
	}

	// Build status response
	status := &types.ProcessingStatus{
		BookID:         book.ID,
		Status:         book.Status,
		Stage:          book.Status,
		TotalChapters:  book.TotalChapters,
		ParsedChapters: book.TotalChapters,
		TotalSegments:  book.TotalSegments,
		Error:          book.Error,
		UpdatedAt:      time.Now(),
	}

	// Calculate progress
	switch book.Status {
	case "uploaded":
		status.Progress = 10
	case "parsing":
		status.Progress = 30
	case "segmenting":
		status.Progress = 60
	case "voice_mapping":
		status.Progress = 80
	case "ready":
		status.Progress = 100
	case "error":
		status.Progress = 0
	}

	respondJSON(w, status, http.StatusOK)
}

// ListSegments handles GET /api/v1/books/:id/segments
func (h *BookHandler) ListSegments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract book ID from path
	bookID := extractIDFromPath(r.URL.Path, "/api/v1/books/")
	if bookID == "" {
		respondError(w, "Book ID required", http.StatusBadRequest)
		return
	}

	// Get segments
	segments, err := h.repo.ListSegments(r.Context(), bookID)
	if err != nil {
		respondError(w, "Failed to list segments", http.StatusInternalServerError)
		return
	}

	respondJSON(w, segments, http.StatusOK)
}

// SetVoiceMap handles POST /api/v1/books/:id/voice-map
func (h *BookHandler) SetVoiceMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract book ID from path
	bookID := extractIDFromPath(r.URL.Path, "/api/v1/books/")
	if bookID == "" {
		respondError(w, "Book ID required", http.StatusBadRequest)
		return
	}

	// Parse request body
	var voiceMap types.VoiceMap
	if err := json.NewDecoder(r.Body).Decode(&voiceMap); err != nil {
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	voiceMap.BookID = bookID

	// Save voice map
	if err := h.repo.SaveVoiceMap(r.Context(), &voiceMap); err != nil {
		respondError(w, "Failed to save voice map", http.StatusInternalServerError)
		return
	}

	// Update book status to ready and trigger TTS synthesis
	book, err := h.repo.GetBook(r.Context(), bookID)
	if err == nil && book != nil {
		book.Status = "ready"
		h.repo.UpdateBook(r.Context(), book)
		
		// Start TTS synthesis asynchronously
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Panic in TTS synthesis for %s: %v", bookID, r)
				}
			}()
			
			// Get first available TTS provider
			ttsProviders := h.providerReg.ListTTS()
			if len(ttsProviders) > 0 {
				if err := h.ttsOrchestrator.SynthesizeBook(context.Background(), bookID, ttsProviders[0]); err != nil {
					log.Printf("TTS synthesis failed for book %s: %v", bookID, err)
				}
			} else {
				log.Printf("No TTS providers available for book %s", bookID)
			}
		}()
	}

	respondJSON(w, voiceMap, http.StatusOK)
}

// GetVoiceMap handles GET /api/v1/books/:id/voice-map
func (h *BookHandler) GetVoiceMap(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract book ID from path
	bookID := extractIDFromPath(r.URL.Path, "/api/v1/books/")
	if bookID == "" {
		respondError(w, "Book ID required", http.StatusBadRequest)
		return
	}

	// Get voice map
	voiceMap, err := h.repo.GetVoiceMap(r.Context(), bookID)
	if err != nil {
		respondError(w, "Voice map not found", http.StatusNotFound)
		return
	}

	respondJSON(w, voiceMap, http.StatusOK)
}

// StreamSegments handles GET /api/v1/books/:id/stream
func (h *BookHandler) StreamSegments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract book ID from path
	bookID := extractIDFromPath(r.URL.Path, "/api/v1/books/")
	if bookID == "" {
		respondError(w, "Book ID required", http.StatusBadRequest)
		return
	}

	// Get optional "after" parameter for resumption
	afterSegmentID := r.URL.Query().Get("after")

	// Get stream items
	items, err := h.streamingService.StreamSegments(r.Context(), bookID, afterSegmentID)
	if err != nil {
		respondError(w, "Failed to stream segments", http.StatusInternalServerError)
		return
	}

	// Encode as NDJSON
	ndjson, err := streaming.EncodeNDJSON(items)
	if err != nil {
		respondError(w, "Failed to encode stream", http.StatusInternalServerError)
		return
	}

	// Return NDJSON response
	w.Header().Set("Content-Type", "application/x-ndjson")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(ndjson))
}

// DownloadBook handles GET /api/v1/books/:id/download
func (h *BookHandler) DownloadBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract book ID from path
	bookID := extractIDFromPath(r.URL.Path, "/api/v1/books/")
	if bookID == "" {
		respondError(w, "Book ID required", http.StatusBadRequest)
		return
	}

	// Get book metadata
	book, err := h.repo.GetBook(r.Context(), bookID)
	if err != nil {
		respondError(w, "Book not found", http.StatusNotFound)
		return
	}

	// Package the book
	zipReader, err := h.packagingService.PackageBook(r.Context(), bookID)
	if err != nil {
		respondError(w, fmt.Sprintf("Failed to package book: %v", err), http.StatusInternalServerError)
		return
	}

	// Set headers for ZIP download
	filename := fmt.Sprintf("book-%s.zip", bookID)
	if book.Title != "" {
		// Sanitize title for filename
		safeTitle := strings.ReplaceAll(book.Title, " ", "_")
		safeTitle = strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
				return r
			}
			return -1
		}, safeTitle)
		if safeTitle != "" {
			filename = fmt.Sprintf("%s.zip", safeTitle)
		}
	}

	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.WriteHeader(http.StatusOK)

	// Copy ZIP data to response
	io.Copy(w, zipReader)
}

// GetAudio handles GET /api/v1/books/:id/audio/:segmentId
func (h *BookHandler) GetAudio(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract book ID from path
	bookID := extractIDFromPath(r.URL.Path, "/api/v1/books/")
	if bookID == "" {
		respondError(w, "Book ID required", http.StatusBadRequest)
		return
	}

	// Extract segment ID (after /audio/)
	parts := strings.Split(r.URL.Path, "/audio/")
	if len(parts) < 2 {
		respondError(w, "Segment ID required", http.StatusBadRequest)
		return
	}
	segmentID := parts[1]

	// Try different audio formats
	var audioReader io.ReadCloser
	var err error
	var format string

	for _, audioFormat := range []string{"wav", "mp3", "ogg", "flac"} {
		audioPath := filepath.Join("books", bookID, "audio", fmt.Sprintf("%s.%s", segmentID, audioFormat))
		audioReader, err = h.storage.Get(r.Context(), audioPath)
		if err == nil {
			format = audioFormat
			break
		}
	}

	if err != nil {
		respondError(w, "Audio file not found", http.StatusNotFound)
		return
	}
	defer audioReader.Close()

	// Set content type based on format
	contentType := "audio/wav"
	switch format {
	case "mp3":
		contentType = "audio/mpeg"
	case "ogg":
		contentType = "audio/ogg"
	case "flac":
		contentType = "audio/flac"
	}

	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(http.StatusOK)

	// Stream audio data
	io.Copy(w, audioReader)
}

// Helper functions

func extractIDFromPath(path, prefix string) string {
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func respondJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}
