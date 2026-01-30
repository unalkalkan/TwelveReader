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
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// HybridOrchestrator manages the hybrid LLM->TTS->Playback pipeline
// with incremental voice mapping as personas are discovered
type HybridOrchestrator struct {
	config      PipelineConfig
	repo        book.Repository
	storage     storage.Adapter
	llmProvider provider.LLMProvider
	providerReg *provider.Registry

	// Pipeline state
	mu        sync.RWMutex
	pipelines map[string]*hybridPipelineState
}

// hybridPipelineState tracks state for a single book's hybrid pipeline
type hybridPipelineState struct {
	bookID           string
	status           *PipelineStatus
	chapters         []*types.Chapter
	progressCallback ProgressCallback
	cancelFunc       context.CancelFunc
	wg               sync.WaitGroup

	// Segmentation state
	segmentsMu          sync.RWMutex
	allSegments         []*types.Segment
	segmentCounter      int
	totalParagraphs     int
	processedParagraphs int

	// Persona tracking
	personaMu          sync.RWMutex
	discoveredPersonas map[string]bool   // All personas seen
	mappedPersonas     map[string]string // persona -> voiceID
	unmappedPersonas   []string          // Personas needing mapping
	initialMappingDone bool              // Whether initial 5-segment mapping is complete
	initialMappingOnce sync.Once         // Ensure initial mapping is processed only once

	// Segment queue
	segmentQueue *SegmentQueue

	// Channels for coordination
	voiceMappingNeeded chan PersonaDiscoveryEvent
	voiceMappingDone   chan VoiceMappingUpdate

	// TTS state
	ttsMu            sync.RWMutex
	synthesizedCount int
	ttsWorkers       sync.WaitGroup
}

// PersonaDiscoveryEvent signals that new personas need voice mapping
type PersonaDiscoveryEvent struct {
	Personas        []string       // Newly discovered personas
	IsInitial       bool           // True if this is the initial 5-segment pause
	BlockingSegment *types.Segment // First segment blocked by unmapped persona
}

// VoiceMappingUpdate signals that voice mapping has been updated
type VoiceMappingUpdate struct {
	VoiceMap  *types.VoiceMap
	IsInitial bool
}

// NewHybridOrchestrator creates a new hybrid pipeline orchestrator
func NewHybridOrchestrator(
	config PipelineConfig,
	repo book.Repository,
	storageAdapter storage.Adapter,
	llmProvider provider.LLMProvider,
	providerReg *provider.Registry,
) *HybridOrchestrator {
	return &HybridOrchestrator{
		config:      config,
		repo:        repo,
		storage:     storageAdapter,
		llmProvider: llmProvider,
		providerReg: providerReg,
		pipelines:   make(map[string]*hybridPipelineState),
	}
}

// StartPipeline initiates the hybrid pipeline for a book
func (o *HybridOrchestrator) StartPipeline(
	ctx context.Context,
	bookID string,
	chapters []*types.Chapter,
	progressCallback ProgressCallback,
) error {
	o.mu.Lock()
	if _, exists := o.pipelines[bookID]; exists {
		o.mu.Unlock()
		return fmt.Errorf("pipeline already running for book %s", bookID)
	}

	pipelineCtx, cancel := context.WithCancel(ctx)
	state := &hybridPipelineState{
		bookID:             bookID,
		chapters:           chapters,
		allSegments:        make([]*types.Segment, 0),
		discoveredPersonas: make(map[string]bool),
		mappedPersonas:     make(map[string]string),
		unmappedPersonas:   make([]string, 0),
		segmentQueue:       NewSegmentQueue(),
		voiceMappingNeeded: make(chan PersonaDiscoveryEvent, 10),
		voiceMappingDone:   make(chan VoiceMappingUpdate, 10),
		progressCallback:   progressCallback,
		cancelFunc:         cancel,
	}

	// Calculate total paragraphs
	for _, chapter := range chapters {
		state.totalParagraphs += len(chapter.Paragraphs)
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
				Total:   state.totalParagraphs,
			},
			{
				Stage:   "synthesizing",
				Status:  "pending",
				Message: "Waiting for voice mapping",
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
	state.wg.Add(2)
	go o.runSegmentationStage(pipelineCtx, state)
	go o.runTTSStage(pipelineCtx, state)

	// Monitor pipeline completion
	go func() {
		state.wg.Wait()
		o.completePipeline(state)
	}()

	return nil
}

// runSegmentationStage processes chapters through LLM segmentation
func (o *HybridOrchestrator) runSegmentationStage(ctx context.Context, state *hybridPipelineState) {
	defer state.wg.Done()

	now := time.Now()
	o.updateStageProgress(state, "segmenting", func(stage *StageProgress) {
		stage.StartedAt = &now
	})

	segService := segmentation.NewService(o.llmProvider, o.config.SegmentationBatchSize)

	// Process chapters with persona discovery
	for _, chapter := range state.chapters {
		if ctx.Err() != nil {
			return
		}

		err := o.segmentChapterWithPersonaTracking(ctx, state, segService, chapter)
		if err != nil {
			log.Printf("Failed to segment chapter %s: %v", chapter.ID, err)
			now := time.Now()
			o.updateStageProgress(state, "segmenting", func(stage *StageProgress) {
				stage.Status = "error"
				stage.Message = fmt.Sprintf("Segmentation failed: %v", err)
				stage.CompletedAt = &now
			})
			o.notifyProgress(state)
			return
		}
	}

	// Mark segmentation as complete
	now = time.Now()
	o.updateStageProgress(state, "segmenting", func(stage *StageProgress) {
		stage.Status = "completed"
		stage.Current = state.totalParagraphs
		stage.Total = state.totalParagraphs
		stage.Percentage = 100
		stage.Message = "Book analysis complete"
		stage.CompletedAt = &now
	})
	o.notifyProgress(state)

	// Update book metadata
	book, err := o.repo.GetBook(ctx, state.bookID)
	if err == nil && book != nil {
		state.segmentsMu.RLock()
		book.TotalSegments = len(state.allSegments)
		state.segmentsMu.RUnlock()
		o.repo.UpdateBook(ctx, book)
	}
}

// segmentChapterWithPersonaTracking segments a chapter and tracks persona discovery
func (o *HybridOrchestrator) segmentChapterWithPersonaTracking(
	ctx context.Context,
	state *hybridPipelineState,
	segService *segmentation.Service,
	chapter *types.Chapter,
) error {
	paragraphs := chapter.Paragraphs

	// Process paragraphs in batches
	for i := 0; i < len(paragraphs); {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		segService.SetBatchSize(5)
		batchEnd := i + 5
		if batchEnd > len(paragraphs) {
			batchEnd = len(paragraphs)
		}

		// Build batch request manually since we need more control
		batchReq := o.buildBatchRequest(state, segService, paragraphs, i, batchEnd)

		// Segment the batch
		resp, err := o.llmProvider.BatchSegment(ctx, batchReq)
		if err != nil {
			// Fallback to individual processing on error
			log.Printf("Batch segmentation failed, falling back: %v", err)
			err = o.processParagraphsIndividually(ctx, state, segService, chapter, paragraphs, i, batchEnd)
			if err != nil {
				return err
			}
			i = batchEnd
			continue
		}

		// Process batch results
		for _, result := range resp.Results {
			for _, llmSeg := range result.Segments {
				segment := o.createSegment(state, chapter, &llmSeg, result.ParagraphIndex)

				// Save segment
				if err := o.repo.SaveSegment(ctx, segment); err != nil {
					log.Printf("Failed to save segment %s: %v", segment.ID, err)
					continue
				}

				// Add to state
				state.segmentsMu.Lock()
				state.allSegments = append(state.allSegments, segment)
				segmentCount := len(state.allSegments)
				state.segmentsMu.Unlock()

				// Check for persona discovery
				o.handlePersonaDiscovery(ctx, state, segment, segmentCount)
			}
		}

		// Update progress
		state.processedParagraphs += (batchEnd - i)
		o.updateStageProgress(state, "segmenting", func(stage *StageProgress) {
			stage.Current = state.processedParagraphs
			if state.totalParagraphs > 0 {
				stage.Percentage = float64(state.processedParagraphs) / float64(state.totalParagraphs) * 100
			}
		})
		o.notifyProgress(state)

		i = batchEnd
	}

	return nil
}

// buildBatchRequest creates a batch segmentation request
func (o *HybridOrchestrator) buildBatchRequest(
	state *hybridPipelineState,
	segService *segmentation.Service,
	paragraphs []string,
	start, end int,
) provider.BatchSegmentRequest {
	batchParagraphs := make([]provider.BatchParagraph, 0, end-start)

	for i := start; i < end; i++ {
		contextBefore := o.getContext(paragraphs, i, -2, 2)
		contextAfter := o.getContext(paragraphs, i, 1, 2)

		batchParagraphs = append(batchParagraphs, provider.BatchParagraph{
			Index:         i,
			Text:          paragraphs[i],
			ContextBefore: contextBefore,
			ContextAfter:  contextAfter,
		})
	}

	// Get known personas
	state.personaMu.RLock()
	knownPersonas := make([]string, 0, len(state.discoveredPersonas))
	for persona := range state.discoveredPersonas {
		knownPersonas = append(knownPersonas, persona)
	}
	state.personaMu.RUnlock()

	return provider.BatchSegmentRequest{
		Paragraphs:   batchParagraphs,
		KnownPersons: knownPersonas,
	}
}

// getContext retrieves context paragraphs
func (o *HybridOrchestrator) getContext(paragraphs []string, currentIndex, direction, windowSize int) []string {
	context := make([]string, 0, windowSize)

	if direction < 0 {
		start := currentIndex - windowSize
		if start < 0 {
			start = 0
		}
		for i := start; i < currentIndex; i++ {
			context = append(context, paragraphs[i])
		}
	} else {
		end := currentIndex + windowSize + 1
		if end > len(paragraphs) {
			end = len(paragraphs)
		}
		for i := currentIndex + 1; i < end; i++ {
			context = append(context, paragraphs[i])
		}
	}

	return context
}

// processParagraphsIndividually handles fallback for batch failures
func (o *HybridOrchestrator) processParagraphsIndividually(
	ctx context.Context,
	state *hybridPipelineState,
	segService *segmentation.Service,
	chapter *types.Chapter,
	paragraphs []string,
	start, end int,
) error {
	for i := start; i < end; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		contextBefore := o.getContext(paragraphs, i, -1, 2)
		contextAfter := o.getContext(paragraphs, i, 1, 2)

		req := provider.SegmentRequest{
			Text:          paragraphs[i],
			ContextBefore: contextBefore,
			ContextAfter:  contextAfter,
			KnownPersons:  o.getKnownPersonas(state),
		}

		resp, err := o.llmProvider.Segment(ctx, req)
		if err != nil {
			log.Printf("Segmentation failed for paragraph %d: %v", i, err)
			// Create fallback segment
			segment := o.createFallbackSegment(state, chapter, paragraphs[i], i)
			if err := o.repo.SaveSegment(ctx, segment); err != nil {
				log.Printf("Failed to save fallback segment: %v", err)
			}
			state.segmentsMu.Lock()
			state.allSegments = append(state.allSegments, segment)
			state.segmentsMu.Unlock()
			continue
		}

		// Process segments
		for _, llmSeg := range resp.Segments {
			segment := o.createSegment(state, chapter, &llmSeg, i)
			if err := o.repo.SaveSegment(ctx, segment); err != nil {
				log.Printf("Failed to save segment %s: %v", segment.ID, err)
				continue
			}

			state.segmentsMu.Lock()
			state.allSegments = append(state.allSegments, segment)
			segmentCount := len(state.allSegments)
			state.segmentsMu.Unlock()

			o.handlePersonaDiscovery(ctx, state, segment, segmentCount)
		}
	}

	return nil
}

// createSegment creates a segment from LLM response
func (o *HybridOrchestrator) createSegment(
	state *hybridPipelineState,
	chapter *types.Chapter,
	llmSeg *provider.Segment,
	paragraphIndex int,
) *types.Segment {
	state.segmentCounter++

	// Normalize persona name
	persona := o.normalizePersona(llmSeg.Person)

	return &types.Segment{
		ID:               fmt.Sprintf("seg_%05d", state.segmentCounter),
		BookID:           state.bookID,
		Chapter:          chapter.ID,
		TOCPath:          chapter.TOCPath,
		Text:             llmSeg.Text,
		Language:         llmSeg.Language,
		Person:           persona,
		VoiceDescription: llmSeg.VoiceDescription,
		SourceContext: &types.SourceContext{
			PrevParagraphID: fmt.Sprintf("%s_para_%03d", chapter.ID, paragraphIndex-1),
			NextParagraphID: fmt.Sprintf("%s_para_%03d", chapter.ID, paragraphIndex+1),
		},
		Processing: &types.ProcessingInfo{
			SegmenterVersion: "v1",
			GeneratedAt:      time.Now(),
		},
	}
}

// createFallbackSegment creates a fallback segment when LLM fails
func (o *HybridOrchestrator) createFallbackSegment(
	state *hybridPipelineState,
	chapter *types.Chapter,
	text string,
	paragraphIndex int,
) *types.Segment {
	state.segmentCounter++

	return &types.Segment{
		ID:               fmt.Sprintf("seg_%05d", state.segmentCounter),
		BookID:           state.bookID,
		Chapter:          chapter.ID,
		TOCPath:          chapter.TOCPath,
		Text:             text,
		Language:         "en",
		Person:           "narrator",
		VoiceDescription: "neutral",
		SourceContext: &types.SourceContext{
			PrevParagraphID: fmt.Sprintf("%s_para_%03d", chapter.ID, paragraphIndex-1),
			NextParagraphID: fmt.Sprintf("%s_para_%03d", chapter.ID, paragraphIndex+1),
		},
		Processing: &types.ProcessingInfo{
			SegmenterVersion: "v1",
			GeneratedAt:      time.Now(),
		},
	}
}

// normalizePersona normalizes persona names for consistency
func (o *HybridOrchestrator) normalizePersona(persona string) string {
	// TODO: Add normalization logic if needed
	return persona
}

// getKnownPersonas returns the list of known personas
func (o *HybridOrchestrator) getKnownPersonas(state *hybridPipelineState) []string {
	state.personaMu.RLock()
	defer state.personaMu.RUnlock()

	personas := make([]string, 0, len(state.discoveredPersonas))
	for persona := range state.discoveredPersonas {
		personas = append(personas, persona)
	}
	return personas
}

// handlePersonaDiscovery checks for new personas and triggers mapping if needed
func (o *HybridOrchestrator) handlePersonaDiscovery(
	ctx context.Context,
	state *hybridPipelineState,
	segment *types.Segment,
	segmentCount int,
) {
	state.personaMu.Lock()
	defer state.personaMu.Unlock()

	persona := segment.Person
	isNewPersona := !state.discoveredPersonas[persona]

	if isNewPersona {
		state.discoveredPersonas[persona] = true
	}

	// Check if initial mapping is needed (after 5 segments)
	if !state.initialMappingDone && segmentCount >= o.config.MinSegmentsBeforeTTS {
		state.initialMappingDone = true

		// Collect all discovered personas
		personas := make([]string, 0, len(state.discoveredPersonas))
		for p := range state.discoveredPersonas {
			personas = append(personas, p)
		}

		// Send event for initial voice mapping
		state.voiceMappingNeeded <- PersonaDiscoveryEvent{
			Personas:  personas,
			IsInitial: true,
		}

		// Update book status
		go func() {
			book, err := o.repo.GetBook(ctx, state.bookID)
			if err == nil && book != nil {
				book.Status = "voice_mapping"
				book.WaitingForMapping = true
				book.DiscoveredPersonas = personas
				book.UnmappedPersonas = personas
				o.repo.UpdateBook(ctx, book)
			}
		}()

		// Unlock mutex before waiting for voice mapping
		// This prevents deadlock when applyVoiceMapping tries to acquire the lock
		state.personaMu.Unlock()

		// Wait for initial voice mapping before continuing
		select {
		case mappingUpdate := <-state.voiceMappingDone:
			log.Printf("[runSegmentationStage] Received initial voice mapping")
			// Use Once to ensure this is only processed once across all goroutines
			state.initialMappingOnce.Do(func() {
				log.Printf("[runSegmentationStage] Processing initial mapping (first receiver)")
				o.applyVoiceMapping(ctx, state, mappingUpdate)
				state.initialMappingDone = true

				// Broadcast to other waiting goroutines by resending
				// This won't block because channel has buffer size 10
				state.voiceMappingDone <- mappingUpdate
			})
		case <-ctx.Done():
			return
		}

		// Re-acquire lock before continuing
		state.personaMu.Lock()
	} else if state.initialMappingDone && isNewPersona {
		// New persona discovered after initial mapping
		isMapped := state.mappedPersonas[persona] != ""

		if !isMapped {
			// Add to unmapped list
			state.unmappedPersonas = append(state.unmappedPersonas, persona)

			// Send event for new persona mapping
			state.voiceMappingNeeded <- PersonaDiscoveryEvent{
				Personas:        []string{persona},
				IsInitial:       false,
				BlockingSegment: segment,
			}

			// Update book status
			go func() {
				book, err := o.repo.GetBook(ctx, state.bookID)
				if err == nil && book != nil {
					book.UnmappedPersonas = state.unmappedPersonas
					book.WaitingForMapping = true
					book.PendingSegmentCount = state.segmentQueue.UnmappedCount()
					o.repo.UpdateBook(ctx, book)
				}
			}()
		}
	}

	// Queue segment for TTS
	if state.initialMappingDone {
		isMapped := state.mappedPersonas[persona] != ""
		state.segmentQueue.Enqueue(segment, isMapped)

		if !isMapped {
			// Update pending count
			go func() {
				book, err := o.repo.GetBook(ctx, state.bookID)
				if err == nil && book != nil {
					book.PendingSegmentCount = state.segmentQueue.UnmappedCount()
					o.repo.UpdateBook(ctx, book)
				}
			}()
		}
	}
}

// runTTSStage processes segments through TTS synthesis
func (o *HybridOrchestrator) runTTSStage(ctx context.Context, state *hybridPipelineState) {
	defer state.wg.Done()

	// Wait for initial voice mapping
	select {
	case mappingUpdate := <-state.voiceMappingDone:
		log.Printf("[runTTSStage] Received initial voice mapping signal")
		// Use Once to ensure initial mapping is only processed once
		// Either runTTSStage or runSegmentationStage can win the race
		processed := false
		state.initialMappingOnce.Do(func() {
			log.Printf("[runTTSStage] Processing initial mapping (first receiver)")
			o.applyVoiceMapping(ctx, state, mappingUpdate)
			state.initialMappingDone = true
			processed = true
			// Broadcast to other waiting goroutine
			state.voiceMappingDone <- mappingUpdate
		})
		if !processed {
			log.Printf("[runTTSStage] Initial mapping already processed by runSegmentationStage")
		}
	case <-ctx.Done():
		return
	}

	now := time.Now()
	o.updateStageProgress(state, "synthesizing", func(stage *StageProgress) {
		stage.Status = "in_progress"
		stage.Message = "Generating audio"
		stage.StartedAt = &now
	})
	o.notifyProgress(state)

	// Start TTS workers
	for i := 0; i < o.config.TTSConcurrency; i++ {
		state.ttsWorkers.Add(1)
		go o.ttsWorker(ctx, state, i)
	}

	// Monitor for new voice mappings and handle them
	go o.monitorVoiceMappings(ctx, state)

	// Wait for all TTS workers to complete
	state.ttsWorkers.Wait()

	// Mark TTS as complete
	now = time.Now()
	o.updateStageProgress(state, "synthesizing", func(stage *StageProgress) {
		stage.Status = "completed"
		stage.Percentage = 100
		stage.Message = "All audio synthesized"
		stage.CompletedAt = &now
	})
	o.notifyProgress(state)
}

// ttsWorker processes segments from the queue
func (o *HybridOrchestrator) ttsWorker(ctx context.Context, state *hybridPipelineState, workerID int) {
	defer state.ttsWorkers.Done()

	for {
		if ctx.Err() != nil {
			return
		}

		// Dequeue next segment
		segment := state.segmentQueue.DequeueNext()
		if segment == nil {
			// No segments available, check if we're done
			state.segmentsMu.RLock()
			allSegmentsProcessed := state.segmentCounter > 0 &&
				state.segmentQueue.MappedCount() == 0 &&
				state.segmentQueue.UnmappedCount() == 0
			state.segmentsMu.RUnlock()

			if allSegmentsProcessed {
				return
			}

			// Wait a bit and try again
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// Get voice ID for segment
		state.personaMu.RLock()
		voiceID := state.mappedPersonas[segment.Person]
		state.personaMu.RUnlock()

		if voiceID == "" {
			log.Printf("Warning: Segment %s has no voice mapping for persona %s", segment.ID, segment.Person)
			continue
		}

		// Synthesize audio
		err := o.synthesizeSegment(ctx, state.bookID, segment, voiceID)
		if err != nil {
			log.Printf("Failed to synthesize segment %s: %v", segment.ID, err)
			// TODO: Add to retry queue
			continue
		}

		// Update progress
		state.ttsMu.Lock()
		state.synthesizedCount++
		currentCount := state.synthesizedCount
		state.ttsMu.Unlock()

		state.segmentsMu.RLock()
		totalSegments := len(state.allSegments)
		state.segmentsMu.RUnlock()

		o.updateStageProgress(state, "synthesizing", func(stage *StageProgress) {
			stage.Current = currentCount
			stage.Total = totalSegments
			if totalSegments > 0 {
				stage.Percentage = float64(currentCount) / float64(totalSegments) * 100
			}
			stage.Message = fmt.Sprintf("Synthesizing segment %d of %d", currentCount, totalSegments)
		})

		o.updateStageProgress(state, "ready", func(stage *StageProgress) {
			if stage.Status == "pending" {
				now := time.Now()
				stage.Status = "in_progress"
				stage.Message = "Audio available for playback"
				stage.StartedAt = &now
			}
			stage.Current = currentCount
			stage.Total = totalSegments
			if totalSegments > 0 {
				stage.Percentage = float64(currentCount) / float64(totalSegments) * 100
			}
		})
		o.notifyProgress(state)

		// Update book
		go func() {
			book, err := o.repo.GetBook(ctx, state.bookID)
			if err == nil && book != nil {
				book.SynthesizedSegments = currentCount
				o.repo.UpdateBook(ctx, book)
			}
		}()
	}
}

// monitorVoiceMappings listens for voice mapping updates
func (o *HybridOrchestrator) monitorVoiceMappings(ctx context.Context, state *hybridPipelineState) {
	log.Printf("[monitorVoiceMappings] Starting for book %s", state.bookID)
	for {
		select {
		case mappingUpdate := <-state.voiceMappingDone:
			log.Printf("[monitorVoiceMappings] Received mapping update, isInitial=%v", mappingUpdate.IsInitial)
			// Process all voice mapping updates (both initial and subsequent)
			o.applyVoiceMapping(ctx, state, mappingUpdate)
		case <-ctx.Done():
			log.Printf("[monitorVoiceMappings] Context cancelled, exiting")
			return
		}
	}
}

// synthesizeSegment synthesizes audio for a segment
func (o *HybridOrchestrator) synthesizeSegment(
	ctx context.Context,
	bookID string,
	segment *types.Segment,
	voiceID string,
) error {
	// Get TTS provider
	ttsProviders := o.providerReg.ListTTS()
	if len(ttsProviders) == 0 {
		return fmt.Errorf("no TTS provider available")
	}

	ttsProvider, err := o.providerReg.GetTTS(ttsProviders[0])
	if err != nil {
		return fmt.Errorf("failed to get TTS provider: %w", err)
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

	// Update segment with audio info
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

// ApplyVoiceMapping updates the pipeline with new voice mappings
func (o *HybridOrchestrator) ApplyVoiceMapping(
	ctx context.Context,
	bookID string,
	voiceMap *types.VoiceMap,
	isInitial bool,
) error {
	o.mu.RLock()
	state, exists := o.pipelines[bookID]
	o.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no active pipeline for book %s", bookID)
	}

	// Send mapping update
	state.voiceMappingDone <- VoiceMappingUpdate{
		VoiceMap:  voiceMap,
		IsInitial: isInitial,
	}

	return nil
}

// applyVoiceMapping applies a voice mapping update to the pipeline
func (o *HybridOrchestrator) applyVoiceMapping(
	ctx context.Context,
	state *hybridPipelineState,
	mappingUpdate VoiceMappingUpdate,
) {
	log.Printf("[applyVoiceMapping] Starting for book %s, isInitial=%v", state.bookID, mappingUpdate.IsInitial)

	state.personaMu.Lock()

	log.Printf("[applyVoiceMapping] Before update - Discovered: %v, Mapped: %v, Unmapped: %v",
		keysFromMap(state.discoveredPersonas), state.mappedPersonas, state.unmappedPersonas)

	// Update mapped personas
	for _, pv := range mappingUpdate.VoiceMap.Persons {
		state.mappedPersonas[pv.ID] = pv.ProviderVoice
		log.Printf("[applyVoiceMapping] Mapped persona: %s -> %s", pv.ID, pv.ProviderVoice)
	}

	// Update unmapped personas list
	newUnmapped := make([]string, 0)
	for persona := range state.discoveredPersonas {
		if state.mappedPersonas[persona] == "" {
			newUnmapped = append(newUnmapped, persona)
			log.Printf("[applyVoiceMapping] Persona %s still unmapped", persona)
		}
	}
	state.unmappedPersonas = newUnmapped

	log.Printf("[applyVoiceMapping] After update - Mapped: %v, Unmapped: %v",
		state.mappedPersonas, state.unmappedPersonas)

	// Get newly mapped personas
	newlyMapped := make([]string, 0)
	for _, pv := range mappingUpdate.VoiceMap.Persons {
		if state.mappedPersonas[pv.ID] != "" {
			newlyMapped = append(newlyMapped, pv.ID)
		}
	}

	state.personaMu.Unlock()

	// If this is initial mapping, queue all existing segments
	if mappingUpdate.IsInitial {
		state.segmentsMu.RLock()
		existingSegments := make([]*types.Segment, len(state.allSegments))
		copy(existingSegments, state.allSegments)
		state.segmentsMu.RUnlock()

		log.Printf("[applyVoiceMapping] Initial mapping - queueing %d existing segments", len(existingSegments))

		state.personaMu.RLock()
		for _, segment := range existingSegments {
			isMapped := state.mappedPersonas[segment.Person] != ""
			state.segmentQueue.Enqueue(segment, isMapped)
			log.Printf("[applyVoiceMapping] Queued segment %s (persona: %s, mapped: %v)", segment.ID, segment.Person, isMapped)
		}
		state.personaMu.RUnlock()
	}

	// Promote pending segments with newly mapped personas
	for _, persona := range newlyMapped {
		promoted := state.segmentQueue.PromotePendingSegments(persona)
		if promoted > 0 {
			log.Printf("[applyVoiceMapping] Promoted %d segments for persona %s", promoted, persona)
		}
	}

	// Update book status
	book, err := o.repo.GetBook(ctx, state.bookID)
	if err == nil && book != nil {
		log.Printf("[applyVoiceMapping] Updating book - WaitingForMapping=%v, UnmappedPersonas=%v",
			len(state.unmappedPersonas) > 0, state.unmappedPersonas)

		book.WaitingForMapping = len(state.unmappedPersonas) > 0
		book.UnmappedPersonas = state.unmappedPersonas
		book.PendingSegmentCount = state.segmentQueue.UnmappedCount()

		if mappingUpdate.IsInitial {
			book.Status = "synthesizing"
			log.Printf("[applyVoiceMapping] Setting book status to 'synthesizing' (initial mapping)")
		}

		o.repo.UpdateBook(ctx, book)
		log.Printf("[applyVoiceMapping] Book updated successfully")
	} else {
		log.Printf("[applyVoiceMapping] Failed to update book: %v", err)
	}
}

// Helper function to get keys from a map[string]bool
func keysFromMap(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// GetPipelineStatus returns the current status of a pipeline
func (o *HybridOrchestrator) GetPipelineStatus(bookID string) (*PipelineStatus, error) {
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

// GetPersonaDiscovery returns the persona discovery status for a book
func (o *HybridOrchestrator) GetPersonaDiscovery(bookID string) (*types.PersonaDiscovery, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()

	state, exists := o.pipelines[bookID]
	if !exists {
		return nil, fmt.Errorf("no active pipeline for book %s", bookID)
	}

	state.personaMu.RLock()
	defer state.personaMu.RUnlock()

	discovered := make([]string, 0, len(state.discoveredPersonas))
	for persona := range state.discoveredPersonas {
		discovered = append(discovered, persona)
	}

	mapped := make(map[string]string)
	for persona, voiceID := range state.mappedPersonas {
		mapped[persona] = voiceID
	}

	unmapped := make([]string, len(state.unmappedPersonas))
	copy(unmapped, state.unmappedPersonas)

	return &types.PersonaDiscovery{
		Discovered:      discovered,
		Mapped:          mapped,
		Unmapped:        unmapped,
		PendingSegments: state.segmentQueue.UnmappedCount(),
	}, nil
}

// CancelPipeline stops a running pipeline
func (o *HybridOrchestrator) CancelPipeline(bookID string) error {
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

// completePipeline finalizes the pipeline
func (o *HybridOrchestrator) completePipeline(state *hybridPipelineState) {
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
		book.WaitingForMapping = false
		o.repo.UpdateBook(ctx, book)
	}

	// Clean up pipeline state
	o.mu.Lock()
	delete(o.pipelines, state.bookID)
	o.mu.Unlock()
}

// updateStageProgress updates a specific stage's progress
func (o *HybridOrchestrator) updateStageProgress(state *hybridPipelineState, stageName string, updateFn func(*StageProgress)) {
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
func (o *HybridOrchestrator) notifyProgress(state *hybridPipelineState) {
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
