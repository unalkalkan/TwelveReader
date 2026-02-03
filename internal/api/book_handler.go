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
	"github.com/unalkalkan/TwelveReader/internal/pipeline"
	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/internal/streaming"
	"github.com/unalkalkan/TwelveReader/internal/tts"
	"github.com/unalkalkan/TwelveReader/internal/util"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// BookHandler handles book-related API endpoints
type BookHandler struct {
	repo               book.Repository
	parserFactory      parser.Factory
	providerReg        *provider.Registry
	ttsOrchestrator    *tts.Orchestrator
	hybridOrchestrator *pipeline.HybridOrchestrator
	packagingService   *packaging.Service
	streamingService   *streaming.Service
	storage            storage.Adapter
}

// NewBookHandler creates a new book handler
func NewBookHandler(repo book.Repository, parserFactory parser.Factory, providerReg *provider.Registry, storage storage.Adapter) *BookHandler {
	// Get first available LLM provider for hybrid orchestrator
	var llmProvider provider.LLMProvider
	llmProviders := providerReg.ListLLM()
	if len(llmProviders) > 0 {
		llmProvider, _ = providerReg.GetLLM(llmProviders[0])
	}

	return &BookHandler{
		repo:            repo,
		parserFactory:   parserFactory,
		providerReg:     providerReg,
		ttsOrchestrator: tts.NewOrchestrator(providerReg, repo, storage, 3),
		hybridOrchestrator: pipeline.NewHybridOrchestrator(
			pipeline.DefaultPipelineConfig(),
			repo,
			storage,
			llmProvider,
			providerReg,
		),
		packagingService: packaging.NewService(repo, storage),
		streamingService: streaming.NewService(repo),
		storage:          storage,
	}
}

// ListBooks handles GET /api/v1/books
func (h *BookHandler) ListBooks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	books, err := h.repo.ListBooks(r.Context())
	if err != nil {
		log.Printf("Failed to list books: %v", err)
		respondError(w, "Failed to list books", http.StatusInternalServerError)
		return
	}

	respondJSON(w, books, http.StatusOK)
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
				log.Printf("[PANIC] Book processing for %s: %v", bookID, r)
				h.updateBookError(context.Background(), bookID, fmt.Sprintf("Processing panic: %v", r))
			}
		}()
		h.processBook(bookID, data, format)
	}()

	// Return success
	respondJSON(w, newBook, http.StatusCreated)
}

// processBook handles async book processing using the hybrid pipeline
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

	// Save chapters and count total paragraphs
	totalParagraphs := 0
	for i, chapter := range chapters {
		chapter.BookID = bookID
		chapter.Number = i + 1
		totalParagraphs += len(chapter.Paragraphs)
		if err := h.repo.SaveChapter(ctx, chapter); err != nil {
			log.Printf("Failed to save chapter %s: %v", chapter.ID, err)
		}
	}

	// Update book with chapter count and total paragraphs
	if book != nil {
		book.TotalChapters = len(chapters)
		book.TotalParagraphs = totalParagraphs
		book.Status = "segmenting"
		h.repo.UpdateBook(ctx, book)
	}

	// Start hybrid pipeline with progress tracking
	progressCallback := func(status *pipeline.PipelineStatus) {
		book, err := h.repo.GetBook(ctx, bookID)
		if err != nil {
			log.Printf("Failed to get book for progress update: %v", err)
			return
		}
		if book == nil {
			return
		}

		// Update book status based on pipeline progress
		for _, stage := range status.Stages {
			switch stage.Stage {
			case "segmenting":
				book.SegmentedParagraphs = stage.Current
				if stage.Total > 0 {
					book.TotalParagraphs = stage.Total
				}
				if stage.Status == "in_progress" {
					book.Status = "segmenting"
				} else if stage.Status == "waiting_for_mapping" {
					book.Status = "voice_mapping"
					book.WaitingForMapping = true
				} else if stage.Status == "completed" {
					// Segmentation complete, update total segments
					if stage.Total > 0 {
						book.TotalSegments = stage.Total
					}
				}
			case "synthesizing":
				book.SynthesizedSegments = stage.Current
				if stage.Total > 0 {
					book.TotalSegments = stage.Total
				}
				if stage.Status == "in_progress" {
					book.Status = "synthesizing"
				}
			case "ready":
				if stage.Status == "completed" {
					book.Status = "synthesized"
				}
			}
		}

		// Try to get persona information from orchestrator
		if personaDiscovery, err := h.hybridOrchestrator.GetPersonaDiscovery(bookID); err == nil {
			book.DiscoveredPersonas = personaDiscovery.Discovered
			book.UnmappedPersonas = personaDiscovery.Unmapped
			book.PendingSegmentCount = personaDiscovery.PendingSegments
		}

		if err := h.repo.UpdateBook(ctx, book); err != nil {
			log.Printf("Failed to update book progress: %v", err)
		}
	}

	// Start the hybrid pipeline
	if err := h.hybridOrchestrator.StartPipeline(ctx, bookID, chapters, progressCallback); err != nil {
		log.Printf("Failed to start hybrid pipeline for book %s: %v", bookID, err)
		h.updateBookError(ctx, bookID, fmt.Sprintf("Pipeline error: %v", err))
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

	// Build status response with real progress
	status := &types.ProcessingStatus{
		BookID:              book.ID,
		Status:              book.Status,
		Stage:               book.Status,
		TotalChapters:       book.TotalChapters,
		ParsedChapters:      book.TotalChapters,
		TotalSegments:       book.TotalSegments,
		TotalParagraphs:     book.TotalParagraphs,
		SegmentedParagraphs: book.SegmentedParagraphs,
		SynthesizedSegments: book.SynthesizedSegments,
		Error:               book.Error,
		UpdatedAt:           time.Now(),
	}

	// Calculate progress based on current stage
	switch book.Status {
	case "uploaded":
		status.Progress = 0
	case "parsing":
		status.Progress = 0 // Progress within parsing not tracked
	case "segmenting":
		// Calculate actual segmentation progress
		if book.TotalParagraphs > 0 {
			status.Progress = float64(book.SegmentedParagraphs) / float64(book.TotalParagraphs) * 100
		} else {
			status.Progress = 0
		}
	case "voice_mapping":
		status.Progress = 100 // Waiting for user input
	case "ready":
		status.Progress = 100 // Ready for synthesis
	case "synthesizing":
		// Calculate actual synthesis progress
		if book.TotalSegments > 0 {
			status.Progress = float64(book.SynthesizedSegments) / float64(book.TotalSegments) * 100
		} else {
			status.Progress = 0
		}
	case "synthesized":
		status.Progress = 100
	case "synthesis_error":
		// Show how far we got
		if book.TotalSegments > 0 {
			status.Progress = float64(book.SynthesizedSegments) / float64(book.TotalSegments) * 100
		}
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

	log.Printf("[SetVoiceMap] Received request for book %s", bookID)

	// Parse request body
	var voiceMap types.VoiceMap
	if err := json.NewDecoder(r.Body).Decode(&voiceMap); err != nil {
		log.Printf("[SetVoiceMap] Failed to decode request body: %v", err)
		respondError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	voiceMap.BookID = bookID
	log.Printf("[SetVoiceMap] Voice map contains %d personas", len(voiceMap.Persons))
	for _, pv := range voiceMap.Persons {
		log.Printf("[SetVoiceMap]   - %s -> %s", pv.ID, pv.ProviderVoice)
	}

	// Save voice map
	if err := h.repo.SaveVoiceMap(r.Context(), &voiceMap); err != nil {
		log.Printf("[SetVoiceMap] Failed to save voice map: %v", err)
		respondError(w, "Failed to save voice map", http.StatusInternalServerError)
		return
	}
	log.Printf("[SetVoiceMap] Voice map saved successfully")

	// Check if this is initial mapping or update for newly discovered persona
	isInitial := r.URL.Query().Get("initial") == "true"
	isUpdate := r.URL.Query().Get("update") == "true"

	// Determine which type of mapping this is
	if !isInitial && !isUpdate {
		// Default behavior: if no query param, assume initial for backward compatibility
		isInitial = true
	}
	log.Printf("[SetVoiceMap] Mapping type: isInitial=%v, isUpdate=%v", isInitial, isUpdate)

	// Apply voice mapping to hybrid orchestrator
	// The orchestrator will update book.UnmappedPersonas and book.WaitingForMapping
	log.Printf("[SetVoiceMap] Applying voice mapping to orchestrator")
	if err := h.hybridOrchestrator.ApplyVoiceMapping(r.Context(), bookID, &voiceMap, isInitial); err != nil {
		// Log error but don't fail the request - orchestrator might not be running
		log.Printf("[SetVoiceMap] Failed to apply voice mapping to orchestrator: %v", err)

		// If orchestrator is not running, manually update book status
		book, err := h.repo.GetBook(r.Context(), bookID)
		if err == nil && book != nil {
			log.Printf("[SetVoiceMap] Manually updating book status (orchestrator not running)")
			if book.Status == "voice_mapping" {
				book.Status = "ready"
			}
			book.WaitingForMapping = false
			h.repo.UpdateBook(r.Context(), book)
		}
	} else {
		log.Printf("[SetVoiceMap] Voice mapping applied to orchestrator successfully")
	}
	// Note: If orchestrator is running, it will handle updating book status in applyVoiceMapping()

	log.Printf("[SetVoiceMap] Returning success response")
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

	for _, audioFormat := range util.AudioFormats() {
		audioPath := util.GetAudioPath(bookID, segmentID, audioFormat)
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

// GetPipelineStatus handles GET /api/v1/books/:id/pipeline/status
func (h *BookHandler) GetPipelineStatus(w http.ResponseWriter, r *http.Request) {
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

	// Get book to ensure it exists
	book, err := h.repo.GetBook(r.Context(), bookID)
	if err != nil {
		respondError(w, "Book not found", http.StatusNotFound)
		return
	}

	// Try to get status from active hybrid orchestrator pipeline
	pipelineStatus, err := h.hybridOrchestrator.GetPipelineStatus(bookID)
	if err == nil && pipelineStatus != nil {
		// Convert pipeline status to processing status
		status := convertPipelineStatusToProcessingStatus(pipelineStatus, book)
		respondJSON(w, status, http.StatusOK)
		return
	}

	// If no active pipeline, build status from book metadata
	status := buildPipelineStatusFromBook(book)
	respondJSON(w, status, http.StatusOK)
}

// GetPersonas handles GET /api/v1/books/:id/personas
func (h *BookHandler) GetPersonas(w http.ResponseWriter, r *http.Request) {
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

	log.Printf("[GetPersonas] Received request for book %s", bookID)

	// Get book to check discovered personas
	book, err := h.repo.GetBook(r.Context(), bookID)
	if err != nil {
		log.Printf("[GetPersonas] Book not found: %v", err)
		respondError(w, "Book not found", http.StatusNotFound)
		return
	}

	log.Printf("[GetPersonas] Book status: %s, DiscoveredPersonas: %v, UnmappedPersonas: %v",
		book.Status, book.DiscoveredPersonas, book.UnmappedPersonas)

	// Get voice map
	voiceMap, err := h.repo.GetVoiceMap(r.Context(), bookID)
	mapped := make(map[string]string)
	if err == nil && voiceMap != nil {
		log.Printf("[GetPersonas] Found voice map with %d personas", len(voiceMap.Persons))
		for _, pv := range voiceMap.Persons {
			mapped[pv.ID] = pv.ProviderVoice
			log.Printf("[GetPersonas]   - %s -> %s", pv.ID, pv.ProviderVoice)
		}
	} else {
		log.Printf("[GetPersonas] No voice map found or error: %v", err)
	}

	// Build persona discovery response
	personaDiscovery := &types.PersonaDiscovery{
		Discovered:      book.DiscoveredPersonas,
		Mapped:          mapped,
		Unmapped:        book.UnmappedPersonas,
		PendingSegments: book.PendingSegmentCount,
	}

	log.Printf("[GetPersonas] Returning: Discovered=%v, Mapped=%v, Unmapped=%v, Pending=%d",
		personaDiscovery.Discovered, len(personaDiscovery.Mapped),
		personaDiscovery.Unmapped, personaDiscovery.PendingSegments)

	respondJSON(w, personaDiscovery, http.StatusOK)
}

// Helper functions

// convertPipelineStatusToProcessingStatus converts pipeline.PipelineStatus to types.ProcessingStatus
func convertPipelineStatusToProcessingStatus(pipelineStatus *pipeline.PipelineStatus, book *types.Book) *types.ProcessingStatus {
	status := &types.ProcessingStatus{
		BookID:              book.ID,
		Status:              book.Status,
		Stage:               book.Status,
		TotalChapters:       book.TotalChapters,
		ParsedChapters:      book.TotalChapters,
		TotalSegments:       book.TotalSegments,
		TotalParagraphs:     book.TotalParagraphs,
		SegmentedParagraphs: book.SegmentedParagraphs,
		SynthesizedSegments: book.SynthesizedSegments,
		Error:               book.Error,
		UpdatedAt:           pipelineStatus.UpdatedAt,
	}

	// Extract progress from stages
	for _, stage := range pipelineStatus.Stages {
		switch stage.Stage {
		case "segmenting":
			status.SegmentedParagraphs = stage.Current
			if stage.Total > 0 {
				status.TotalParagraphs = stage.Total
			}
			if stage.Status == "in_progress" {
				status.Stage = "segmenting"
				if stage.Total > 0 {
					status.Progress = float64(stage.Current) / float64(stage.Total) * 100
				}
			} else if stage.Status == "waiting_for_mapping" {
				status.Stage = "voice_mapping"
				status.Progress = 100
			}
		case "synthesizing":
			status.SynthesizedSegments = stage.Current
			if stage.Total > 0 {
				status.TotalSegments = stage.Total
			}
			if stage.Status == "in_progress" {
				status.Stage = "synthesizing"
				if stage.Total > 0 {
					status.Progress = float64(stage.Current) / float64(stage.Total) * 100
				}
			}
		case "ready":
			if stage.Status == "completed" {
				status.Stage = "synthesized"
				status.Progress = 100
			}
		}
	}

	return status
}

// buildPipelineStatusFromBook creates a pipeline status from book metadata
func buildPipelineStatusFromBook(book *types.Book) *types.ProcessingStatus {
	status := &types.ProcessingStatus{
		BookID:              book.ID,
		Status:              book.Status,
		Stage:               book.Status,
		TotalChapters:       book.TotalChapters,
		ParsedChapters:      book.TotalChapters,
		TotalSegments:       book.TotalSegments,
		TotalParagraphs:     book.TotalParagraphs,
		SegmentedParagraphs: book.SegmentedParagraphs,
		SynthesizedSegments: book.SynthesizedSegments,
		Error:               book.Error,
		UpdatedAt:           time.Now(),
	}

	// Calculate progress based on current stage
	switch book.Status {
	case "uploaded":
		status.Progress = 0
	case "parsing":
		status.Progress = 0
	case "segmenting":
		if book.TotalParagraphs > 0 {
			status.Progress = float64(book.SegmentedParagraphs) / float64(book.TotalParagraphs) * 100
		}
	case "voice_mapping":
		status.Progress = 100 // Segmentation complete, waiting for user
	case "synthesizing":
		if book.TotalSegments > 0 {
			status.Progress = float64(book.SynthesizedSegments) / float64(book.TotalSegments) * 100
		}
	case "synthesized":
		status.Progress = 100
	}

	return status
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
