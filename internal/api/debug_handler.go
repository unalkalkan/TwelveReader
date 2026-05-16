package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/book"
	"github.com/unalkalkan/TwelveReader/internal/debugstate"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/internal/util"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// DebugHandler exposes read-only/debug telemetry endpoints for the dashboard.
type DebugHandler struct {
	repo    book.Repository
	storage storage.Adapter
	store   *debugstate.Store
}

func NewDebugHandler(repo book.Repository, storageAdapter storage.Adapter) *DebugHandler {
	return &DebugHandler{repo: repo, storage: storageAdapter, store: debugstate.NewStore(storageAdapter)}
}

func (h *DebugHandler) ListSynthJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	bookID := extractDebugBookID(r.URL.Path)
	if bookID == "" {
		respondError(w, "Book ID required", http.StatusBadRequest)
		return
	}
	jobs, err := h.buildSynthJobs(r.Context(), bookID)
	if err != nil {
		respondError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]interface{}{"book_id": bookID, "jobs": jobs}, http.StatusOK)
}

func (h *DebugHandler) AudioValidation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	bookID := extractDebugBookID(r.URL.Path)
	if bookID == "" {
		respondError(w, "Book ID required", http.StatusBadRequest)
		return
	}
	validations, err := h.validateAudioArtifacts(r.Context(), bookID)
	if err != nil {
		respondError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]interface{}{"book_id": bookID, "artifacts": validations}, http.StatusOK)
}

func (h *DebugHandler) PlaybackEvents(w http.ResponseWriter, r *http.Request) {
	bookID := extractDebugBookID(r.URL.Path)
	if bookID == "" {
		respondError(w, "Book ID required", http.StatusBadRequest)
		return
	}
	switch r.Method {
	case http.MethodGet:
		limit := parseLimit(r, 100)
		events, err := h.store.ListPlaybackEvents(r.Context(), bookID, limit)
		if err != nil {
			respondError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		respondJSON(w, map[string]interface{}{"book_id": bookID, "events": events}, http.StatusOK)
	case http.MethodPost:
		var event types.PlaybackEvent
		if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
			respondError(w, "Invalid playback event payload", http.StatusBadRequest)
			return
		}
		event.BookID = bookID
		if event.UserID == "" {
			event.UserID = "single-user"
		}
		if event.CreatedAt.IsZero() {
			event.CreatedAt = time.Now().UTC()
		}
		if err := h.store.SavePlaybackEvent(r.Context(), &event); err != nil {
			respondError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = h.store.SaveEvent(r.Context(), &types.DebugEvent{BookID: bookID, SegmentID: event.SegmentID, Scope: "user", Severity: playbackSeverity(event), Title: fmt.Sprintf("User %s", event.EventType), Detail: event.Error, Source: "playback", CreatedAt: event.CreatedAt})
		respondJSON(w, event, http.StatusCreated)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *DebugHandler) UserProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	bookID := extractDebugBookID(r.URL.Path)
	if bookID == "" {
		respondError(w, "Book ID required", http.StatusBadRequest)
		return
	}
	progress, err := h.buildUserProgress(r.Context(), bookID)
	if err != nil {
		respondError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, progress, http.StatusOK)
}

func (h *DebugHandler) Events(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	bookID := extractDebugBookID(r.URL.Path)
	limit := parseLimit(r, 100)
	events, err := h.buildEvents(r.Context(), bookID, limit)
	if err != nil {
		respondError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	respondJSON(w, map[string]interface{}{"book_id": bookID, "events": events}, http.StatusOK)
}

func (h *DebugHandler) EventStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		respondError(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}
	bookID := extractDebugBookID(r.URL.Path)
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			events, _ := h.buildEvents(r.Context(), bookID, 25)
			data, _ := json.Marshal(events)
			fmt.Fprintf(w, "event: debug-state\ndata: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (h *DebugHandler) buildSynthJobs(ctx context.Context, bookID string) ([]*types.SynthJob, error) {
	segments, err := h.repo.ListSegments(ctx, bookID)
	if err != nil {
		return nil, err
	}
	persisted, _ := h.store.ListSynthJobs(ctx, bookID)
	bySegment := make(map[string]*types.SynthJob)
	for _, job := range persisted {
		bySegment[job.SegmentID] = job
	}
	validations, _ := h.validateAudioArtifacts(ctx, bookID)
	validationBySegment := make(map[string]*types.AudioArtifactValidation)
	for _, validation := range validations {
		validationBySegment[validation.SegmentID] = validation
	}
	jobs := make([]*types.SynthJob, 0, len(segments))
	now := time.Now().UTC()
	for _, segment := range segments {
		if segment == nil {
			continue
		}
		if job := bySegment[segment.ID]; job != nil {
			jobs = append(jobs, job)
			continue
		}
		validation := validationBySegment[segment.ID]
		status := "queued"
		if validation != nil && validation.Status == "attached" {
			status = "completed"
		} else if validation != nil && validation.Status == "stale" {
			status = "retrying"
		} else if segment.VoiceID == "" {
			status = "not_created"
		}
		job := &types.SynthJob{ID: fmt.Sprintf("synth_%s_%s", debugstate.SafeID(bookID), debugstate.SafeID(segment.ID)), BookID: bookID, SegmentID: segment.ID, Status: status, VoiceID: segment.VoiceID, VoiceDescription: segment.VoiceDescription, UpdatedAt: now}
		if segment.Processing != nil {
			job.Provider = segment.Processing.TTSProvider
			if !segment.Processing.GeneratedAt.IsZero() {
				completed := segment.Processing.GeneratedAt.UTC()
				job.CompletedAt = &completed
			}
		}
		if validation != nil {
			job.OutputPath = validation.Path
			job.OutputFormat = validation.Format
			job.OutputBytes = validation.Bytes
			if validation.Status == "missing" && segment.VoiceID != "" {
				job.Status = "failed"
				job.Error = validation.Error
			}
		}
		jobs = append(jobs, job)
	}
	sort.Slice(jobs, func(i, j int) bool { return jobs[i].SegmentID < jobs[j].SegmentID })
	return jobs, nil
}

func (h *DebugHandler) validateAudioArtifacts(ctx context.Context, bookID string) ([]*types.AudioArtifactValidation, error) {
	segments, err := h.repo.ListSegments(ctx, bookID)
	if err != nil {
		return nil, err
	}
	checkedAt := time.Now().UTC()
	validations := make([]*types.AudioArtifactValidation, 0, len(segments))
	for _, segment := range segments {
		if segment == nil {
			continue
		}
		validation := &types.AudioArtifactValidation{BookID: bookID, SegmentID: segment.ID, Status: "missing", CheckedAt: checkedAt}
		for _, format := range util.AudioFormats() {
			path := util.GetAudioPath(bookID, segment.ID, format)
			reader, err := h.storage.Get(ctx, path)
			if err != nil {
				continue
			}
			data, readErr := io.ReadAll(reader)
			reader.Close()
			validation.Format = format
			validation.Path = path
			validation.Bytes = int64(len(data))
			validation.ContentType = audioContentType(format)
			if readErr != nil {
				validation.Status = "invalid"
				validation.Error = readErr.Error()
			} else if len(data) == 0 {
				validation.Status = "invalid"
				validation.Error = "audio artifact is empty"
			} else if segment.AudioStale {
				validation.Status = "stale"
			} else {
				validation.Status = "attached"
			}
			break
		}
		if validation.Path == "" && segment.VoiceID != "" {
			validation.Error = "segment has voice_id but audio artifact is missing"
		}
		validations = append(validations, validation)
	}
	return validations, nil
}

func (h *DebugHandler) buildUserProgress(ctx context.Context, bookID string) (*types.UserProgress, error) {
	segments, err := h.repo.ListSegments(ctx, bookID)
	if err != nil {
		return nil, err
	}
	validations, _ := h.validateAudioArtifacts(ctx, bookID)
	events, _ := h.store.ListPlaybackEvents(ctx, bookID, 1000)
	progress := &types.UserProgress{BookID: bookID, UserID: "single-user", CanRead: len(segments) > 0, TotalSegments: len(segments), JourneyState: "not_started", UpdatedAt: time.Now().UTC()}
	audioReady := 0
	for _, validation := range validations {
		if validation.Status == "attached" {
			audioReady++
		}
	}
	progress.CanListenAll = len(segments) > 0 && audioReady == len(segments)
	for i := len(events) - 1; i >= 0; i-- {
		event := events[i]
		if event == nil {
			continue
		}
		progress.UpdatedAt = maxTime(progress.UpdatedAt, event.CreatedAt)
		switch event.EventType {
		case "book_opened":
			progress.JourneyState = "opened"
		case "segment_opened":
			progress.LastOpenedSegmentID = event.SegmentID
			progress.JourneyState = "reading"
		case "read":
			progress.LastReadSegmentID = event.SegmentID
			progress.JourneyState = "reading"
		case "play", "pause":
			progress.LastListenedSegmentID = event.SegmentID
			progress.JourneyState = "listening"
		case "complete":
			progress.LastListenedSegmentID = event.SegmentID
			progress.CompletedListenSegments++
		case "failed":
			progress.PlaybackFailures++
			progress.StuckSegmentID = event.SegmentID
			progress.JourneyState = "stuck"
		}
	}
	if progress.CompletedListenSegments >= len(segments) && len(segments) > 0 {
		progress.JourneyState = "completed"
	}
	return progress, nil
}

func (h *DebugHandler) buildEvents(ctx context.Context, bookID string, limit int) ([]*types.DebugEvent, error) {
	persisted, _ := h.store.ListEvents(ctx, bookID, limit)
	generated := make([]*types.DebugEvent, 0)
	books := []*types.Book{}
	if bookID != "" {
		book, err := h.repo.GetBook(ctx, bookID)
		if err == nil && book != nil {
			books = append(books, book)
		}
	} else if list, err := h.repo.ListBooks(ctx); err == nil {
		books = list
	}
	now := time.Now().UTC()
	for _, book := range books {
		if book == nil {
			continue
		}
		severity := "info"
		if book.Status == "synthesized" {
			severity = "success"
		} else if strings.Contains(book.Status, "error") || book.Error != "" {
			severity = "danger"
		} else if book.Status == "voice_mapping" || book.Status == "synthesizing" {
			severity = "warning"
		}
		generated = append(generated, &types.DebugEvent{ID: fmt.Sprintf("book_%s_state", book.ID), BookID: book.ID, Scope: "book", Severity: severity, Title: fmt.Sprintf("Book %s", book.Status), Detail: book.Error, Source: "derived", CreatedAt: now})
		validations, _ := h.validateAudioArtifacts(ctx, book.ID)
		missing := 0
		for _, validation := range validations {
			if validation.Status == "missing" || validation.Status == "invalid" {
				missing++
			}
		}
		if missing > 0 {
			generated = append(generated, &types.DebugEvent{ID: fmt.Sprintf("book_%s_missing_audio", book.ID), BookID: book.ID, Scope: "segment", Severity: "warning", Title: "Missing audio artifacts", Detail: fmt.Sprintf("%d segment(s) missing or invalid", missing), Source: "derived", CreatedAt: now})
		}
	}
	all := append(generated, persisted...)
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })
	if limit > 0 && len(all) > limit {
		all = all[:limit]
	}
	return all, nil
}

func extractDebugBookID(path string) string {
	prefix := "/api/v1/debug/books/"
	if !strings.HasPrefix(path, prefix) {
		return ""
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

func parseLimit(r *http.Request, fallback int) int {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit <= 0 {
		return fallback
	}
	if limit > 1000 {
		return 1000
	}
	return limit
}

func audioContentType(format string) string {
	switch format {
	case "mp3":
		return "audio/mpeg"
	case "ogg":
		return "audio/ogg"
	case "flac":
		return "audio/flac"
	default:
		return "audio/wav"
	}
}

func playbackSeverity(event types.PlaybackEvent) string {
	if event.EventType == "failed" || event.Error != "" {
		return "danger"
	}
	if event.EventType == "complete" {
		return "success"
	}
	return "info"
}

func maxTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
}
