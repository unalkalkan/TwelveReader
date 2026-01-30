package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/book"
	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/internal/segmentation"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/internal/tts"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// PipelineConfig defines configuration for the pipeline orchestrator
type PipelineConfig struct {
	// MinSegmentsBeforeTTS is the minimum number of segments that must be
	// completed before starting TTS synthesis
	MinSegmentsBeforeTTS int

	// TTSConcurrency is the number of concurrent TTS workers
	TTSConcurrency int

	// SegmentationBatchSize is the batch size for LLM segmentation
	SegmentationBatchSize int
}

// DefaultPipelineConfig returns the default pipeline configuration
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		MinSegmentsBeforeTTS:  5,
		TTSConcurrency:        3,
		SegmentationBatchSize: 2,
	}
}

// StageProgress represents progress in a specific pipeline stage
type StageProgress struct {
	Stage       string     `json:"stage"`      // "parsing", "segmenting", "synthesizing", "ready"
	Current     int        `json:"current"`    // Current item count
	Total       int        `json:"total"`      // Total items to process
	Percentage  float64    `json:"percentage"` // Progress percentage
	Status      string     `json:"status"`     // "pending", "in_progress", "completed", "error"
	Message     string     `json:"message"`    // Status message
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// PipelineStatus represents the overall pipeline status
type PipelineStatus struct {
	BookID    string          `json:"book_id"`
	Stages    []StageProgress `json:"stages"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// ProgressCallback is called when pipeline progress is updated
type ProgressCallback func(status *PipelineStatus)

// Orchestrator manages the parallel LLM->TTS->Playback pipeline
type Orchestrator struct {
	config      PipelineConfig
	repo        book.Repository
	storage     storage.Adapter
	llmProvider provider.LLMProvider
	providerReg *provider.Registry
	ttsOrch     *tts.Orchestrator

	// Pipeline state
	mu        sync.RWMutex
	pipelines map[string]*pipelineState
}

// pipelineState tracks state for a single book's pipeline
type pipelineState struct {
	bookID           string
	status           *PipelineStatus
	segments         []*types.Segment
	segmentsMu       sync.RWMutex
	ttsQueue         chan *types.Segment
	voiceMap         *types.VoiceMap
	progressCallback ProgressCallback
	cancelFunc       context.CancelFunc
	wg               sync.WaitGroup
}

// NewOrchestrator creates a new pipeline orchestrator
func NewOrchestrator(
	config PipelineConfig,
	repo book.Repository,
	storageAdapter storage.Adapter,
	llmProvider provider.LLMProvider,
	providerReg *provider.Registry,
) *Orchestrator {
	ttsOrch := tts.NewOrchestrator(providerReg, repo, storageAdapter, config.TTSConcurrency)

	return &Orchestrator{
		config:      config,
		repo:        repo,
		storage:     storageAdapter,
		llmProvider: llmProvider,
		providerReg: providerReg,
		ttsOrch:     ttsOrch,
		pipelines:   make(map[string]*pipelineState),
	}
}

// StartPipeline initiates the parallel pipeline for a book
func (o *Orchestrator) StartPipeline(
	ctx context.Context,
	bookID string,
	chapters []*types.Chapter,
	voiceMap *types.VoiceMap,
	progressCallback ProgressCallback,
) error {
	o.mu.Lock()
	if _, exists := o.pipelines[bookID]; exists {
		o.mu.Unlock()
		return fmt.Errorf("pipeline already running for book %s", bookID)
	}

	pipelineCtx, cancel := context.WithCancel(ctx)
	state := &pipelineState{
		bookID:           bookID,
		segments:         make([]*types.Segment, 0),
		ttsQueue:         make(chan *types.Segment, 100),
		voiceMap:         voiceMap,
		progressCallback: progressCallback,
		cancelFunc:       cancel,
	}

	// Initialize pipeline status
	state.status = &PipelineStatus{
		BookID: bookID,
		Stages: []StageProgress{
			{
				Stage:   "segmenting",
				Status:  "in_progress",
				Message: "Analyzing book content with LLM",
				Current: 0,
			},
			{
				Stage:   "synthesizing",
				Status:  "pending",
				Message: "Waiting for segments to be ready",
				Current: 0,
			},
			{
				Stage:   "ready",
				Status:  "pending",
				Message: "Waiting for audio synthesis",
				Current: 0,
			},
		},
		UpdatedAt: time.Now(),
	}

	o.pipelines[bookID] = state
	o.mu.Unlock()

	// Start the pipeline stages
	state.wg.Add(2) // Segmentation and TTS stages
	go o.runSegmentationStage(pipelineCtx, state, chapters)
	go o.runTTSStage(pipelineCtx, state)

	// Monitor pipeline completion
	go func() {
		state.wg.Wait()
		o.completePipeline(state)
	}()

	return nil
}

// runSegmentationStage processes chapters through LLM segmentation
func (o *Orchestrator) runSegmentationStage(ctx context.Context, state *pipelineState, chapters []*types.Chapter) {
	defer state.wg.Done()
	defer close(state.ttsQueue) // Signal TTS stage when done

	now := time.Now()
	o.updateStageProgress(state, "segmenting", func(stage *StageProgress) {
		stage.StartedAt = &now
	})

	segService := segmentation.NewService(o.llmProvider, o.config.SegmentationBatchSize)

	// Count total paragraphs
	totalParagraphs := 0
	for _, chapter := range chapters {
		totalParagraphs += len(chapter.Paragraphs)
	}

	o.updateStageProgress(state, "segmenting", func(stage *StageProgress) {
		stage.Total = totalParagraphs
	})

	// Progress callback for segmentation
	progressCallback := func(segmentedParagraphs, total int) {
		o.updateStageProgress(state, "segmenting", func(stage *StageProgress) {
			stage.Current = segmentedParagraphs
			stage.Total = total
			if total > 0 {
				stage.Percentage = float64(segmentedParagraphs) / float64(total) * 100
			}
		})
		o.notifyProgress(state)
	}

	// Custom segmentation with streaming
	segments, err := o.segmentWithStreaming(ctx, state, segService, chapters, progressCallback)

	if err != nil {
		log.Printf("Segmentation failed for book %s: %v", state.bookID, err)
		now := time.Now()
		o.updateStageProgress(state, "segmenting", func(stage *StageProgress) {
			stage.Status = "error"
			stage.Message = fmt.Sprintf("Segmentation failed: %v", err)
			stage.CompletedAt = &now
		})
		o.notifyProgress(state)
		return
	}

	// Mark segmentation as complete
	now = time.Now()
	o.updateStageProgress(state, "segmenting", func(stage *StageProgress) {
		stage.Status = "completed"
		stage.Current = len(segments)
		stage.Total = len(segments)
		stage.Percentage = 100
		stage.Message = "Book analysis complete"
		stage.CompletedAt = &now
	})
	o.notifyProgress(state)

	// Update book metadata
	book, err := o.repo.GetBook(ctx, state.bookID)
	if err == nil && book != nil {
		book.TotalSegments = len(segments)
		book.Status = "synthesizing"
		o.repo.UpdateBook(ctx, book)
	}
}

// segmentWithStreaming performs segmentation and streams segments to TTS queue
func (o *Orchestrator) segmentWithStreaming(
	ctx context.Context,
	state *pipelineState,
	segService *segmentation.Service,
	chapters []*types.Chapter,
	progressCallback func(int, int),
) ([]*types.Segment, error) {
	allSegments := make([]*types.Segment, 0)
	ttsStarted := false

	// Custom progress callback that also checks threshold and queues segments
	totalParagraphs := 0
	for _, ch := range chapters {
		totalParagraphs += len(ch.Paragraphs)
	}

	segmentHandler := func(segment *types.Segment) error {
		// Save segment metadata
		if err := o.repo.SaveSegment(ctx, segment); err != nil {
			log.Printf("Failed to save segment %s: %v", segment.ID, err)
			return err
		}

		// Add to state
		state.segmentsMu.Lock()
		state.segments = append(state.segments, segment)
		currentCount := len(state.segments)
		state.segmentsMu.Unlock()

		allSegments = append(allSegments, segment)

		// Check if we should start TTS
		if !ttsStarted && currentCount >= o.config.MinSegmentsBeforeTTS {
			ttsStarted = true
			now := time.Now()
			o.updateStageProgress(state, "synthesizing", func(stage *StageProgress) {
				stage.Status = "in_progress"
				stage.Message = "Generating audio"
				stage.StartedAt = &now
			})
			o.notifyProgress(state)

			// Queue all segments collected so far
			state.segmentsMu.RLock()
			for _, seg := range state.segments {
				select {
				case state.ttsQueue <- seg:
				case <-ctx.Done():
					state.segmentsMu.RUnlock()
					return ctx.Err()
				}
			}
			state.segmentsMu.RUnlock()
		} else if ttsStarted {
			// Queue new segment immediately
			select {
			case state.ttsQueue <- segment:
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		return nil
	}

	// Segment chapters with streaming
	_, err := o.segmentChaptersStreaming(ctx, state.bookID, chapters, segService, segmentHandler, progressCallback)
	if err != nil {
		return nil, err
	}

	// If we never hit threshold, start TTS now
	if !ttsStarted {
		now := time.Now()
		o.updateStageProgress(state, "synthesizing", func(stage *StageProgress) {
			stage.Status = "in_progress"
			stage.Message = "Generating audio"
			stage.StartedAt = &now
		})
		o.notifyProgress(state)

		for _, segment := range allSegments {
			select {
			case state.ttsQueue <- segment:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return allSegments, nil
}

// segmentChaptersStreaming is a helper that processes chapters and calls handler for each segment
func (o *Orchestrator) segmentChaptersStreaming(
	ctx context.Context,
	bookID string,
	chapters []*types.Chapter,
	segService *segmentation.Service,
	segmentHandler func(*types.Segment) error,
	progressCallback func(int, int),
) ([]*types.Segment, error) {
	allSegments := make([]*types.Segment, 0)

	wrappedProgressCallback := func(processed, total int) {
		if progressCallback != nil {
			progressCallback(processed, total)
		}
	}

	// Use the service's existing method but process segments one by one
	segments, err := segService.SegmentChaptersWithProgress(ctx, bookID, chapters, wrappedProgressCallback)
	if err != nil {
		return nil, err
	}

	// Call handler for each segment
	for _, segment := range segments {
		if err := segmentHandler(segment); err != nil {
			return nil, err
		}
		allSegments = append(allSegments, segment)
	}

	return allSegments, nil
}

// synthesizeSegmentWithProgress synthesizes a single segment with progress callback
func (o *Orchestrator) synthesizeSegmentWithProgress(
	ctx context.Context,
	bookID string,
	segment *types.Segment,
	voiceID string,
	progressCallback func(string, int, int),
) error {
	// Get first available TTS provider
	ttsProviders := o.providerReg.ListTTS()
	if len(ttsProviders) == 0 {
		return fmt.Errorf("no TTS provider available")
	}

	ttsProvider, err := o.providerReg.GetTTS(ttsProviders[0])
	if err != nil {
		return fmt.Errorf("failed to get TTS provider: %w", err)
	}

	// Use default voice if not specified
	if voiceID == "" {
		voiceID = "default"
	}

	// Prepare TTS request
	req := provider.TTSRequest{
		Text:             segment.Text,
		VoiceID:          voiceID,
		Language:         segment.Language,
		VoiceDescription: segment.VoiceDescription,
	}

	// Call TTS provider
	resp, err := ttsProvider.Synthesize(ctx, req)
	if err != nil {
		return fmt.Errorf("TTS provider failed: %w", err)
	}

	// Store audio file
	audioPath := fmt.Sprintf("books/%s/audio/%s.%s", bookID, segment.ID, resp.Format)
	if err := o.storage.Put(ctx, audioPath, bytes.NewReader(resp.AudioData)); err != nil {
		return fmt.Errorf("failed to store audio: %w", err)
	}

	// Update segment with audio path and timestamps
	segment.VoiceID = voiceID
	if len(resp.Timestamps) > 0 {
		segment.Timestamps = &types.TimestampData{
			Precision: "word",
			Items:     make([]types.TimestampItem, len(resp.Timestamps)),
		}
		for i, ts := range resp.Timestamps {
			segment.Timestamps.Items[i] = types.TimestampItem{
				Word:  ts.Word,
				Start: ts.Start,
				End:   ts.End,
			}
		}
	}

	// Update processing info
	if segment.Processing == nil {
		segment.Processing = &types.ProcessingInfo{}
	}
	segment.Processing.TTSProvider = ttsProvider.Name()
	segment.Processing.GeneratedAt = time.Now()

	// Save updated segment
	if err := o.repo.SaveSegment(ctx, segment); err != nil {
		return fmt.Errorf("failed to update segment: %w", err)
	}

	return nil
}

// runTTSStage processes segments through TTS synthesis
func (o *Orchestrator) runTTSStage(ctx context.Context, state *pipelineState) {
	defer state.wg.Done()

	processedCount := 0
	totalSegments := 0

	// TTS progress callback
	ttsProgressCallback := func(segmentID string, current, total int) {
		o.updateStageProgress(state, "synthesizing", func(stage *StageProgress) {
			stage.Current = current
			stage.Total = total
			if total > 0 {
				stage.Percentage = float64(current) / float64(total) * 100
			}
			stage.Message = fmt.Sprintf("Synthesizing segment %d of %d", current, total)
		})
		o.notifyProgress(state)
	}

	// Process segments from queue
	for segment := range state.ttsQueue {
		if ctx.Err() != nil {
			return
		}

		totalSegments++

		// Update total count
		o.updateStageProgress(state, "synthesizing", func(stage *StageProgress) {
			stage.Total = totalSegments
		})

		// Get voice for segment from voice map
		voice := ""
		if state.voiceMap != nil {
			for _, pv := range state.voiceMap.Persons {
				if pv.ID == segment.Person {
					voice = pv.ProviderVoice
					break
				}
			}
		}

		// Synthesize audio using internal method
		err := o.synthesizeSegmentWithProgress(ctx, state.bookID, segment, voice, ttsProgressCallback)
		if err != nil {
			log.Printf("Failed to synthesize segment %s: %v", segment.ID, err)
			continue
		}

		processedCount++

		// Update ready stage as segments become available
		o.updateStageProgress(state, "ready", func(stage *StageProgress) {
			if stage.Status == "pending" {
				now := time.Now()
				stage.Status = "in_progress"
				stage.Message = "Audio available for playback"
				stage.StartedAt = &now
			}
			stage.Current = processedCount
			stage.Total = totalSegments
			if totalSegments > 0 {
				stage.Percentage = float64(processedCount) / float64(totalSegments) * 100
			}
		})
		o.notifyProgress(state)
	}

	// Mark TTS stage as complete
	now := time.Now()
	o.updateStageProgress(state, "synthesizing", func(stage *StageProgress) {
		stage.Status = "completed"
		stage.Percentage = 100
		stage.Message = "All audio synthesized"
		stage.CompletedAt = &now
	})
	o.notifyProgress(state)
}

// completePipeline finalizes the pipeline
func (o *Orchestrator) completePipeline(state *pipelineState) {
	ctx := context.Background()

	// Mark ready stage as complete
	now := time.Now()
	o.updateStageProgress(state, "ready", func(stage *StageProgress) {
		stage.Status = "completed"
		stage.Percentage = 100
		stage.Message = "Book ready for playback"
		stage.CompletedAt = &now
	})
	o.notifyProgress(state)

	// Update book status
	book, err := o.repo.GetBook(ctx, state.bookID)
	if err == nil && book != nil {
		book.Status = "synthesized"
		o.repo.UpdateBook(ctx, book)
	}

	// Clean up pipeline state
	o.mu.Lock()
	delete(o.pipelines, state.bookID)
	o.mu.Unlock()
}

// updateStageProgress updates a specific stage's progress
func (o *Orchestrator) updateStageProgress(state *pipelineState, stageName string, updateFn func(*StageProgress)) {
	state.segmentsMu.Lock()
	defer state.segmentsMu.Unlock()

	for i := range state.status.Stages {
		if state.status.Stages[i].Stage == stageName {
			updateFn(&state.status.Stages[i])
			break
		}
	}
	state.status.UpdatedAt = time.Now()
}

// notifyProgress sends progress update to callback
func (o *Orchestrator) notifyProgress(state *pipelineState) {
	if state.progressCallback != nil {
		// Create a copy to avoid race conditions
		state.segmentsMu.RLock()
		statusCopy := *state.status
		statusCopy.Stages = make([]StageProgress, len(state.status.Stages))
		copy(statusCopy.Stages, state.status.Stages)
		state.segmentsMu.RUnlock()

		state.progressCallback(&statusCopy)
	}
}

// GetPipelineStatus returns the current status of a pipeline
func (o *Orchestrator) GetPipelineStatus(bookID string) (*PipelineStatus, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	state, exists := o.pipelines[bookID]
	if !exists {
		return nil, fmt.Errorf("no active pipeline for book %s", bookID)
	}

	state.segmentsMu.RLock()
	defer state.segmentsMu.RUnlock()

	// Return a copy
	statusCopy := *state.status
	statusCopy.Stages = make([]StageProgress, len(state.status.Stages))
	copy(statusCopy.Stages, state.status.Stages)

	return &statusCopy, nil
}

// CancelPipeline stops a running pipeline
func (o *Orchestrator) CancelPipeline(bookID string) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	state, exists := o.pipelines[bookID]
	if !exists {
		return fmt.Errorf("no active pipeline for book %s", bookID)
	}

	state.cancelFunc()
	delete(o.pipelines, bookID)

	return nil
}
