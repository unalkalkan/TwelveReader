package pipeline

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/book"
	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/internal/segmentation"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

const defaultSegmentSynthesisMaxRetries = 3

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
	segmentsMu           sync.RWMutex
	allSegments          []*types.Segment
	segmentCounter       int
	totalParagraphs      int
	processedParagraphs  int
	segmentationComplete bool // Signals when all segments have been processed and queued

	// Persona tracking
	personaMu          sync.RWMutex
	discoveredPersonas map[string]bool   // All personas seen
	mappedPersonas     map[string]string // persona -> voiceID
	unmappedPersonas   []string          // Personas needing mapping
	initialMappingDone bool              // Whether initial 5-segment mapping is complete

	// Segment queue
	segmentQueue *SegmentQueue

	// Channels for coordination
	voiceMappingNeeded      chan PersonaDiscoveryEvent
	voiceMappingDone        chan VoiceMappingUpdate
	initialMappingReceived  chan struct{} // Closed when initial mapping is received and applied
	closeInitialMappingOnce sync.Once     // Ensures initialMappingReceived is closed exactly once

	// TTS state
	ttsMu                  sync.RWMutex
	synthesizedCount       int
	permanentlyFailedCount int
	ttsWorkers             sync.WaitGroup
	maxRetries             int
	activeSynthesis        int32
}

func (state *hybridPipelineState) staleProcessingAllowed() bool {
	state.segmentsMu.RLock()
	segmentationComplete := state.segmentationComplete
	state.segmentsMu.RUnlock()
	return segmentationComplete
}

func (state *hybridPipelineState) currentVoiceForPersona(persona string) string {
	state.personaMu.RLock()
	defer state.personaMu.RUnlock()
	return state.mappedPersonas[persona]
}

func (state *hybridPipelineState) voiceMapLocked() *types.VoiceMap {
	voiceMap := &types.VoiceMap{BookID: state.bookID, Persons: make([]types.PersonVoice, 0, len(state.mappedPersonas))}
	for persona, voiceID := range state.mappedPersonas {
		if persona == "" || voiceID == "" {
			continue
		}
		voiceMap.Persons = append(voiceMap.Persons, types.PersonVoice{ID: persona, ProviderVoice: voiceID})
	}
	return voiceMap
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
		bookID:                 bookID,
		chapters:               chapters,
		allSegments:            make([]*types.Segment, 0),
		discoveredPersonas:     make(map[string]bool),
		mappedPersonas:         make(map[string]string),
		unmappedPersonas:       make([]string, 0),
		segmentQueue:           NewSegmentQueue(),
		voiceMappingNeeded:     make(chan PersonaDiscoveryEvent, 10),
		voiceMappingDone:       make(chan VoiceMappingUpdate, 10),
		initialMappingReceived: make(chan struct{}),
		progressCallback:       progressCallback,
		cancelFunc:             cancel,
		maxRetries:             defaultSegmentSynthesisMaxRetries,
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
	defer func() {
		// Mark segmentation as complete so TTS workers know when to exit
		state.segmentsMu.Lock()
		state.segmentationComplete = true
		state.segmentsMu.Unlock()
		log.Printf("[runSegmentationStage] Segmentation marked complete")
	}()

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

	o.ensureInitialMappingRequested(ctx, state)

	// Update book metadata
	book, err := o.repo.GetBook(ctx, state.bookID)
	if err == nil && book != nil {
		state.segmentsMu.RLock()
		book.TotalSegments = len(state.allSegments)
		state.segmentsMu.RUnlock()
		// Only update status if we're still in a state where this makes sense
		// Don't overwrite if already synthesized or in error state
		if book.Status == "segmenting" {
			book.Status = "synthesizing"
		}
		o.repo.UpdateBook(ctx, book)
	}
}

// ensureInitialMappingRequested handles short books that complete segmentation before
// reaching MinSegmentsBeforeTTS. Without this fallback, TTS waits forever for the
// initialMappingReceived signal while the UI can remain stuck at synthesizing 0%.
func (o *HybridOrchestrator) ensureInitialMappingRequested(ctx context.Context, state *hybridPipelineState) {
	state.personaMu.Lock()
	if state.initialMappingDone {
		state.personaMu.Unlock()
		return
	}
	personas := make([]string, 0, len(state.discoveredPersonas))
	for p := range state.discoveredPersonas {
		personas = append(personas, p)
	}
	state.personaMu.Unlock()

	if len(personas) == 0 {
		log.Printf("[ensureInitialMappingRequested] No personas discovered for book %s", state.bookID)
		return
	}

	defaultVoice, err := o.repo.GetDefaultVoice(ctx)
	if err == nil && defaultVoice != nil && defaultVoice.VoiceID != "" {
		state.personaMu.Lock()
		for _, persona := range personas {
			if state.mappedPersonas[persona] == "" {
				state.mappedPersonas[persona] = defaultVoice.VoiceID
			}
			state.unmappedPersonas = removeString(state.unmappedPersonas, persona)
		}
		state.initialMappingDone = true
		state.closeInitialMappingOnce.Do(func() {
			close(state.initialMappingReceived)
			log.Printf("[ensureInitialMappingRequested] Short-book initial mapping auto-applied from default voice")
		})
		voiceMap := state.voiceMapLocked()
		state.personaMu.Unlock()

		if err := o.repo.SaveVoiceMap(ctx, voiceMap); err != nil {
			log.Printf("[ensureInitialMappingRequested] Failed to persist default voice map: %v", err)
		}

		state.segmentsMu.RLock()
		segments := make([]*types.Segment, len(state.allSegments))
		copy(segments, state.allSegments)
		state.segmentsMu.RUnlock()
		for _, segment := range segments {
			state.segmentQueue.Enqueue(segment, true)
		}

		o.updateBookAfterDefaultVoiceMapping(ctx, state)
		return
	}
	if err != nil {
		log.Printf("[ensureInitialMappingRequested] Failed to load default voice for short book %s: %v", state.bookID, err)
	}

	state.personaMu.Lock()
	if state.initialMappingDone {
		state.personaMu.Unlock()
		return
	}
	state.initialMappingDone = true
	state.personaMu.Unlock()

	select {
	case state.voiceMappingNeeded <- PersonaDiscoveryEvent{Personas: personas, IsInitial: true}:
	default:
		log.Printf("[ensureInitialMappingRequested] Warning: voiceMappingNeeded channel full")
	}

	book, err := o.repo.GetBook(ctx, state.bookID)
	if err == nil && book != nil {
		book.Status = "voice_mapping"
		book.WaitingForMapping = true
		book.DiscoveredPersonas = personas
		book.UnmappedPersonas = personas
		book.PendingSegmentCount = state.segmentQueue.UnmappedCount()
		o.repo.UpdateBook(ctx, book)
	}

	log.Printf("[ensureInitialMappingRequested] Waiting for initial voice mapping for short book %s", state.bookID)
	select {
	case <-state.initialMappingReceived:
		log.Printf("[ensureInitialMappingRequested] Initial voice mapping received for short book %s", state.bookID)
	case <-ctx.Done():
		log.Printf("[ensureInitialMappingRequested] Context cancelled while waiting for voice mapping")
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
// This function must NOT hold any locks when waiting for external events (like voice mapping)
func (o *HybridOrchestrator) handlePersonaDiscovery(
	ctx context.Context,
	state *hybridPipelineState,
	segment *types.Segment,
	segmentCount int,
) {
	persona := segment.Person

	if o.tryAutoMapPersonaToDefaultVoice(ctx, state, persona) {
		state.segmentQueue.Enqueue(segment, true)
		o.updateBookAfterDefaultVoiceMapping(ctx, state)
		return
	}

	// First, check and update persona discovery under lock
	state.personaMu.Lock()
	isNewPersona := !state.discoveredPersonas[persona]
	if isNewPersona {
		state.discoveredPersonas[persona] = true
	}

	// Check if we need to trigger initial mapping
	needsInitialMapping := !state.initialMappingDone && segmentCount >= o.config.MinSegmentsBeforeTTS
	if needsInitialMapping {
		state.initialMappingDone = true
	}

	// Collect discovered personas if needed (while under lock)
	var personas []string
	if needsInitialMapping {
		personas = make([]string, 0, len(state.discoveredPersonas))
		for p := range state.discoveredPersonas {
			personas = append(personas, p)
		}
	}
	state.personaMu.Unlock()

	// Track if this is the segment that triggers initial mapping (the 5th segment)
	// This segment and all prior ones will be queued by applyVoiceMapping,
	// so this function should NOT queue them to avoid duplicates
	isInitialMappingTrigger := needsInitialMapping

	// Handle initial voice mapping (outside of lock)
	if needsInitialMapping {
		// Send event for initial voice mapping (non-blocking, buffered channel)
		select {
		case state.voiceMappingNeeded <- PersonaDiscoveryEvent{
			Personas:  personas,
			IsInitial: true,
		}:
		default:
			log.Printf("[handlePersonaDiscovery] Warning: voiceMappingNeeded channel full")
		}

		// Update book status asynchronously
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

		// Wait for initial voice mapping before continuing
		// This blocks segmentation until the user provides voice mappings
		log.Printf("[handlePersonaDiscovery] Waiting for initial voice mapping...")
		select {
		case <-state.initialMappingReceived:
			log.Printf("[handlePersonaDiscovery] Initial voice mapping received, continuing segmentation")
		case <-ctx.Done():
			log.Printf("[handlePersonaDiscovery] Context cancelled while waiting for voice mapping")
			return
		}
	}

	// Handle new persona discovered after initial mapping
	state.personaMu.Lock()
	if state.initialMappingDone && isNewPersona && !isInitialMappingTrigger {
		isMapped := state.mappedPersonas[persona] != ""
		if !isMapped {
			state.unmappedPersonas = append(state.unmappedPersonas, persona)
			unmappedCopy := make([]string, len(state.unmappedPersonas))
			copy(unmappedCopy, state.unmappedPersonas)
			state.personaMu.Unlock()

			// Send event for new persona mapping (non-blocking)
			select {
			case state.voiceMappingNeeded <- PersonaDiscoveryEvent{
				Personas:        []string{persona},
				IsInitial:       false,
				BlockingSegment: segment,
			}:
			default:
				log.Printf("[handlePersonaDiscovery] Warning: voiceMappingNeeded channel full")
			}

			// Update book status asynchronously
			go func() {
				book, err := o.repo.GetBook(ctx, state.bookID)
				if err == nil && book != nil {
					book.UnmappedPersonas = unmappedCopy
					book.WaitingForMapping = true
					book.PendingSegmentCount = state.segmentQueue.UnmappedCount()
					o.repo.UpdateBook(ctx, book)
				}
			}()

			state.personaMu.Lock()
		}
	}

	// Queue segment for TTS (under lock to check mapping status)
	// Only queue if initial mapping is done AND this is NOT the trigger segment
	// The trigger segment and all prior ones are queued by applyVoiceMapping
	if state.initialMappingDone && !isInitialMappingTrigger {
		isMapped := state.mappedPersonas[persona] != ""
		state.personaMu.Unlock()

		state.segmentQueue.Enqueue(segment, isMapped)

		if !isMapped {
			// Update pending count asynchronously
			go func() {
				book, err := o.repo.GetBook(ctx, state.bookID)
				if err == nil && book != nil {
					book.PendingSegmentCount = state.segmentQueue.UnmappedCount()
					o.repo.UpdateBook(ctx, book)
				}
			}()
		}
	} else {
		state.personaMu.Unlock()
	}
}

func (o *HybridOrchestrator) tryAutoMapPersonaToDefaultVoice(ctx context.Context, state *hybridPipelineState, persona string) bool {
	defaultVoice, err := o.repo.GetDefaultVoice(ctx)
	if err != nil || defaultVoice == nil || defaultVoice.VoiceID == "" {
		if err != nil {
			log.Printf("[tryAutoMapPersonaToDefaultVoice] Failed to load default voice: %v", err)
		}
		return false
	}

	state.personaMu.Lock()
	if state.mappedPersonas[persona] != "" {
		state.personaMu.Unlock()
		return false
	}
	state.discoveredPersonas[persona] = true
	state.mappedPersonas[persona] = defaultVoice.VoiceID
	if !state.initialMappingDone {
		state.initialMappingDone = true
		state.closeInitialMappingOnce.Do(func() {
			close(state.initialMappingReceived)
			log.Printf("[tryAutoMapPersonaToDefaultVoice] Initial mapping auto-applied from default voice")
		})
	}
	state.unmappedPersonas = removeString(state.unmappedPersonas, persona)
	voiceMap := state.voiceMapLocked()
	state.personaMu.Unlock()

	if err := o.repo.SaveVoiceMap(ctx, voiceMap); err != nil {
		log.Printf("[tryAutoMapPersonaToDefaultVoice] Failed to persist voice map: %v", err)
	}
	return true
}

func (o *HybridOrchestrator) updateBookAfterDefaultVoiceMapping(ctx context.Context, state *hybridPipelineState) {
	book, err := o.repo.GetBook(ctx, state.bookID)
	if err != nil || book == nil {
		return
	}
	state.personaMu.RLock()
	discovered := make([]string, 0, len(state.discoveredPersonas))
	for persona := range state.discoveredPersonas {
		discovered = append(discovered, persona)
	}
	unmapped := make([]string, len(state.unmappedPersonas))
	copy(unmapped, state.unmappedPersonas)
	state.personaMu.RUnlock()

	book.Status = "synthesizing"
	book.WaitingForMapping = len(unmapped) > 0
	book.DiscoveredPersonas = discovered
	book.UnmappedPersonas = unmapped
	pendingCount := state.segmentQueue.UnmappedCount()
	if len(unmapped) == 0 {
		pendingCount = 0
	}
	book.PendingSegmentCount = pendingCount
	if err := o.repo.UpdateBook(ctx, book); err != nil {
		log.Printf("[updateBookAfterDefaultVoiceMapping] Failed to update book: %v", err)
	}
}

func removeString(values []string, value string) []string {
	if len(values) == 0 {
		return values
	}
	filtered := values[:0]
	for _, candidate := range values {
		if candidate != value {
			filtered = append(filtered, candidate)
		}
	}
	return filtered
}

// runTTSStage processes segments through TTS synthesis
func (o *HybridOrchestrator) runTTSStage(ctx context.Context, state *hybridPipelineState) {
	defer state.wg.Done()

	// Wait for initial voice mapping signal
	log.Printf("[runTTSStage] Waiting for initial voice mapping...")
	select {
	case <-state.initialMappingReceived:
		log.Printf("[runTTSStage] Initial voice mapping received, starting TTS")
	case <-ctx.Done():
		log.Printf("[runTTSStage] Context cancelled while waiting for voice mapping")
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
	log.Printf("[ttsWorker-%d] Starting", workerID)

	for {
		if ctx.Err() != nil {
			log.Printf("[ttsWorker-%d] Context cancelled, exiting", workerID)
			return
		}

		// Reserve an active slot before dequeue so a dequeued segment is never
		// invisible to other idle workers' completion checks.
		atomic.AddInt32(&state.activeSynthesis, 1)

		segment := state.segmentQueue.DequeueNext(state.staleProcessingAllowed())
		if segment == nil {
			atomic.AddInt32(&state.activeSynthesis, -1)

			state.segmentsMu.RLock()
			segmentationDone := state.segmentationComplete
			totalSegments := len(state.allSegments)
			state.segmentsMu.RUnlock()

			mappedCount := state.segmentQueue.ReadyCount()
			unmappedCount := state.segmentQueue.UnmappedCount()
			activeCount := atomic.LoadInt32(&state.activeSynthesis)

			if segmentationDone && mappedCount == 0 && unmappedCount == 0 && activeCount == 0 {
				state.ttsMu.RLock()
				synthesizedCount := state.synthesizedCount
				permanentlyFailed := state.permanentlyFailedCount
				state.ttsMu.RUnlock()

				log.Printf("[ttsWorker-%d] All segments processed (synthesized: %d, failed: %d/%d), exiting",
					workerID, synthesizedCount, permanentlyFailed, totalSegments)
				return
			}

			time.Sleep(100 * time.Millisecond)
			continue
		}

		state.personaMu.RLock()
		voiceID := state.mappedPersonas[segment.Person]
		state.personaMu.RUnlock()

		if voiceID == "" {
			atomic.AddInt32(&state.activeSynthesis, -1)
			log.Printf("[ttsWorker-%d] Warning: Segment %s has no voice mapping for persona %s, re-queueing as unmapped",
				workerID, segment.ID, segment.Person)
			state.segmentQueue.Enqueue(segment, false)
			time.Sleep(50 * time.Millisecond)
			continue
		}

		wasStale := segment.AudioStale
		log.Printf("[ttsWorker-%d] Synthesizing segment %s (persona: %s, voice: %s)",
			workerID, segment.ID, segment.Person, voiceID)
		err := o.synthesizeSegment(ctx, state, segment, voiceID)
		if err != nil {
			retryCount := state.segmentQueue.RecordFailure(segment.ID)
			if retryCount <= state.maxRetries {
				if wasStale {
					state.segmentQueue.EnqueueStale(segment)
				} else {
					state.segmentQueue.Enqueue(segment, true)
				}
				log.Printf("[ttsWorker-%d] Requeued segment %s for retry (attempt %d/%d): %v",
					workerID, segment.ID, retryCount, state.maxRetries, err)
			} else {
				state.segmentQueue.MarkPermanentlyFailed(segment.ID)
				state.ttsMu.Lock()
				state.permanentlyFailedCount++
				state.ttsMu.Unlock()
				log.Printf("[ttsWorker-%d] Segment %s permanently failed after %d retries: %v",
					workerID, segment.ID, state.maxRetries, err)
			}
			atomic.AddInt32(&state.activeSynthesis, -1)
			continue
		}

		state.segmentQueue.ClearRetryTracker(segment.ID)

		state.segmentsMu.RLock()
		totalSegments := len(state.allSegments)
		state.segmentsMu.RUnlock()

		state.ttsMu.Lock()
		if !wasStale || state.synthesizedCount < totalSegments {
			state.synthesizedCount++
		}
		currentCount := state.synthesizedCount
		state.ttsMu.Unlock()

		atomic.AddInt32(&state.activeSynthesis, -1)

		log.Printf("[ttsWorker-%d] Completed segment %s (%d/%d)", workerID, segment.ID, currentCount, totalSegments)

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

		go func(count int) {
			book, err := o.repo.GetBook(ctx, state.bookID)
			if err == nil && book != nil {
				book.SynthesizedSegments = count
				o.repo.UpdateBook(ctx, book)
			}
		}(currentCount)
	}
}

// monitorVoiceMappings listens for voice mapping updates from the voiceMappingDone channel
// Note: Most voice mappings are now applied directly via ApplyVoiceMapping(),
// but this goroutine handles any updates that come through the channel
func (o *HybridOrchestrator) monitorVoiceMappings(ctx context.Context, state *hybridPipelineState) {
	log.Printf("[monitorVoiceMappings] Starting for book %s", state.bookID)
	for {
		select {
		case mappingUpdate := <-state.voiceMappingDone:
			log.Printf("[monitorVoiceMappings] Received mapping update via channel, isInitial=%v", mappingUpdate.IsInitial)
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
	state *hybridPipelineState,
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
	audioPath := fmt.Sprintf("books/%s/audio/%s.%s", state.bookID, segment.ID, resp.Format)
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
	if currentVoice := state.currentVoiceForPersona(segment.Person); currentVoice != "" && currentVoice != voiceID {
		segment.AudioStale = true
		segment.StaleVoiceID = voiceID
		if err := o.repo.SaveSegment(ctx, segment); err != nil {
			log.Printf("[synthesizeSegment] Failed to mark remapped in-flight segment %s stale: %v", segment.ID, err)
		}
		state.segmentQueue.EnqueueStale(segment)
		return nil
	}

	segment.AudioStale = false
	segment.StaleVoiceID = ""

	// Save updated segment
	if err := o.repo.SaveSegment(ctx, segment); err != nil {
		return fmt.Errorf("failed to update segment: %w", err)
	}

	return nil
}

// ApplyVoiceMapping updates the pipeline with new voice mappings
// This is called from the API handler when the user submits voice mappings
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

	log.Printf("[ApplyVoiceMapping] Applying voice mapping for book %s, isInitial=%v", bookID, isInitial)

	// Apply the mapping directly (synchronously)
	o.applyVoiceMapping(ctx, state, VoiceMappingUpdate{
		VoiceMap:  voiceMap,
		IsInitial: isInitial,
	})

	// If this is the initial mapping, signal both the segmentation and TTS stages to continue
	if isInitial {
		// Use sync.Once to ensure the channel is closed exactly once
		state.closeInitialMappingOnce.Do(func() {
			close(state.initialMappingReceived)
			log.Printf("[ApplyVoiceMapping] Initial mapping signal sent")
		})
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

	previousMappings := make(map[string]string, len(state.mappedPersonas))
	for persona, voiceID := range state.mappedPersonas {
		previousMappings[persona] = voiceID
	}

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

	if !mappingUpdate.IsInitial {
		o.markRemappedAudioStale(ctx, state, mappingUpdate.VoiceMap, previousMappings)
	}

	state.personaMu.RLock()
	persistedVoiceMap := state.voiceMapLocked()
	state.personaMu.RUnlock()
	if err := o.repo.SaveVoiceMap(ctx, persistedVoiceMap); err != nil {
		log.Printf("[applyVoiceMapping] Failed to persist voice map: %v", err)
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

func (o *HybridOrchestrator) markRemappedAudioStale(ctx context.Context, state *hybridPipelineState, voiceMap *types.VoiceMap, previousMappings map[string]string) {
	if voiceMap == nil {
		return
	}
	changedVoices := make(map[string]string)
	for _, pv := range voiceMap.Persons {
		if pv.ID == "" || pv.ProviderVoice == "" {
			continue
		}
		oldVoice := previousMappings[pv.ID]
		if oldVoice == "" || oldVoice == pv.ProviderVoice {
			continue
		}
		changedVoices[pv.ID] = oldVoice
	}
	if len(changedVoices) == 0 {
		return
	}

	state.segmentsMu.RLock()
	segments := make([]*types.Segment, len(state.allSegments))
	copy(segments, state.allSegments)
	state.segmentsMu.RUnlock()

	marked := 0
	for _, segment := range segments {
		oldVoice, changed := changedVoices[segment.Person]
		if !changed || segment.VoiceID == "" || segment.VoiceID == state.mappedPersonas[segment.Person] {
			continue
		}
		segment.AudioStale = true
		segment.StaleVoiceID = oldVoice
		if err := o.repo.SaveSegment(ctx, segment); err != nil {
			log.Printf("[markRemappedAudioStale] Failed to mark segment %s stale: %v", segment.ID, err)
			continue
		}
		state.segmentQueue.EnqueueStale(segment)
		marked++
	}
	if marked > 0 {
		log.Printf("[markRemappedAudioStale] Marked %d segment(s) stale for deferred regeneration", marked)
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

	state.segmentsMu.RLock()
	totalSegments := len(state.allSegments)
	state.segmentsMu.RUnlock()

	state.ttsMu.RLock()
	synthesizedCount := state.synthesizedCount
	state.ttsMu.RUnlock()

	permanentlyFailed := state.segmentQueue.PermanentlyFailedCount()
	incompleteSegments := totalSegments - synthesizedCount - permanentlyFailed
	if incompleteSegments < 0 {
		incompleteSegments = 0
	}

	if permanentlyFailed > 0 || incompleteSegments > 0 {
		message := fmt.Sprintf("TTS synthesis failed for %d segment(s) after max retries", permanentlyFailed)
		if incompleteSegments > 0 {
			message = fmt.Sprintf("TTS synthesis incomplete: %d segment(s) did not finish", incompleteSegments)
		}

		now := time.Now()
		o.updateStageProgress(state, "synthesizing", func(stage *StageProgress) {
			stage.Status = "error"
			stage.Message = message
			stage.CompletedAt = &now
		})
		o.updateStageProgress(state, "ready", func(stage *StageProgress) {
			stage.Status = "error"
			stage.Message = message
			stage.CompletedAt = &now
		})
		o.notifyProgress(state)

		book, err := o.repo.GetBook(ctx, state.bookID)
		if err == nil && book != nil {
			book.Status = "error"
			book.Error = message
			book.WaitingForMapping = false
			o.repo.UpdateBook(ctx, book)
		}
	} else {
		now := time.Now()
		o.updateStageProgress(state, "ready", func(stage *StageProgress) {
			stage.Status = "completed"
			stage.Percentage = 100
			stage.Message = "Book ready for playback"
			stage.CompletedAt = &now
		})
		o.notifyProgress(state)

		book, err := o.repo.GetBook(ctx, state.bookID)
		if err == nil && book != nil {
			book.Status = "synthesized"
			book.WaitingForMapping = false
			o.repo.UpdateBook(ctx, book)
		}
	}

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
