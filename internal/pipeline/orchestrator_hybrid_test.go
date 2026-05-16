package pipeline

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/book"
	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func TestSegmentQueueTracksRetryFailureLifecycle(t *testing.T) {
	queue := NewSegmentQueue()
	segment := &types.Segment{ID: "seg_001", Person: "narrator"}

	queue.Enqueue(segment, true)
	if got := queue.DequeueNext(true); got == nil || got.ID != segment.ID {
		t.Fatalf("expected first dequeue to return %s, got %#v", segment.ID, got)
	}

	if retryCount := queue.RecordFailure(segment.ID); retryCount != 1 {
		t.Fatalf("expected first retry count 1, got %d", retryCount)
	}
	queue.Enqueue(segment, true)

	if got := queue.DequeueNext(true); got == nil || got.ID != segment.ID {
		t.Fatalf("expected requeued segment %s, got %#v", segment.ID, got)
	}
	if retryCount := queue.RecordFailure(segment.ID); retryCount != 2 {
		t.Fatalf("expected second retry count 2, got %d", retryCount)
	}
	queue.MarkPermanentlyFailed(segment.ID)

	if queue.PermanentlyFailedCount() != 1 {
		t.Fatalf("expected one permanently failed segment, got %d", queue.PermanentlyFailedCount())
	}
	queue.ClearRetryTracker(segment.ID)
	if queue.RetryCount(segment.ID) != 0 {
		t.Fatalf("expected retry tracker to clear, got %d", queue.RetryCount(segment.ID))
	}
	if queue.PermanentlyFailedCount() != 0 {
		t.Fatalf("expected permanent failure tracker to clear, got %d", queue.PermanentlyFailedCount())
	}
}

func TestHybridTTSWorkerRequeuesFailedSegmentAndThenSucceeds(t *testing.T) {
	repo := newPipelineTestRepository()
	store := newPipelineTestStorage()
	ttsProvider := &pipelineTestTTSProvider{failuresBeforeSuccess: 1}
	registry := provider.NewRegistry()
	if err := registry.RegisterTTS(ttsProvider); err != nil {
		t.Fatalf("register tts provider: %v", err)
	}

	book := &types.Book{ID: "book_retry", Title: "Retry", Status: "synthesizing"}
	if err := repo.SaveBook(context.Background(), book); err != nil {
		t.Fatalf("save book: %v", err)
	}

	segment := &types.Segment{
		ID:               "seg_retry",
		BookID:           book.ID,
		Text:             "retry me",
		Language:         "en",
		Person:           "narrator",
		VoiceDescription: "neutral",
		Processing:       &types.ProcessingInfo{GeneratedAt: time.Now()},
	}

	orchestrator := NewHybridOrchestrator(
		PipelineConfig{TTSConcurrency: 1, MinSegmentsBeforeTTS: 1, SegmentationBatchSize: 1},
		repo,
		store,
		&pipelineTestLLMProvider{},
		registry,
	)

	state := newWorkerTestState(book.ID, segment)
	state.mappedPersonas["narrator"] = "voice-a"
	state.segmentQueue.Enqueue(segment, true)
	state.maxRetries = 1

	state.ttsWorkers.Add(1)
	orchestrator.ttsWorker(context.Background(), state, 0)
	orchestrator.completePipeline(state)

	if got := ttsProvider.callsFor("retry me"); got != 2 {
		t.Fatalf("expected one failed call and one retried success, got %d calls", got)
	}
	if state.synthesizedCount != 1 {
		t.Fatalf("expected synthesized count 1, got %d", state.synthesizedCount)
	}
	if state.segmentQueue.RetryCount(segment.ID) != 0 {
		t.Fatalf("expected retry tracker cleared after success")
	}

	updatedBook, err := repo.GetBook(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("get book: %v", err)
	}
	if updatedBook.Status != "synthesized" {
		t.Fatalf("expected book synthesized after retry success, got %q", updatedBook.Status)
	}
}

func TestHybridTTSWorkerMarksBookErrorAfterRetryBudgetExhausted(t *testing.T) {
	repo := newPipelineTestRepository()
	store := newPipelineTestStorage()
	ttsProvider := &pipelineTestTTSProvider{alwaysFail: true}
	registry := provider.NewRegistry()
	if err := registry.RegisterTTS(ttsProvider); err != nil {
		t.Fatalf("register tts provider: %v", err)
	}

	book := &types.Book{ID: "book_fail", Title: "Fail", Status: "synthesizing"}
	if err := repo.SaveBook(context.Background(), book); err != nil {
		t.Fatalf("save book: %v", err)
	}

	segment := &types.Segment{
		ID:               "seg_fail",
		BookID:           book.ID,
		Text:             "fail me",
		Language:         "en",
		Person:           "narrator",
		VoiceDescription: "neutral",
		Processing:       &types.ProcessingInfo{GeneratedAt: time.Now()},
	}

	orchestrator := NewHybridOrchestrator(
		PipelineConfig{TTSConcurrency: 1, MinSegmentsBeforeTTS: 1, SegmentationBatchSize: 1},
		repo,
		store,
		&pipelineTestLLMProvider{},
		registry,
	)

	state := newWorkerTestState(book.ID, segment)
	state.mappedPersonas["narrator"] = "voice-a"
	state.segmentQueue.Enqueue(segment, true)
	state.maxRetries = 1

	state.ttsWorkers.Add(1)
	orchestrator.ttsWorker(context.Background(), state, 0)
	orchestrator.completePipeline(state)

	if got := ttsProvider.callsFor("fail me"); got != 2 {
		t.Fatalf("expected initial attempt plus one retry, got %d calls", got)
	}
	if state.synthesizedCount != 0 {
		t.Fatalf("expected no synthesized segments, got %d", state.synthesizedCount)
	}
	if state.segmentQueue.PermanentlyFailedCount() != 1 {
		t.Fatalf("expected one permanently failed segment, got %d", state.segmentQueue.PermanentlyFailedCount())
	}

	updatedBook, err := repo.GetBook(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("get book: %v", err)
	}
	if updatedBook.Status != "error" {
		t.Fatalf("expected book error after exhausted retries, got %q", updatedBook.Status)
	}
	if !strings.Contains(updatedBook.Error, "TTS synthesis failed") {
		t.Fatalf("expected TTS failure message, got %q", updatedBook.Error)
	}
}

func TestShortBookRequestsInitialMappingAfterSegmentationCompletesWithoutDefaultVoice(t *testing.T) {
	repo := newPipelineTestRepository()
	store := newPipelineTestStorage()
	registry := provider.NewRegistry()
	if err := registry.RegisterTTS(&pipelineTestTTSProvider{}); err != nil {
		t.Fatalf("register tts provider: %v", err)
	}

	book := &types.Book{ID: "book_short", Title: "Short", Status: "segmenting"}
	if err := repo.SaveBook(context.Background(), book); err != nil {
		t.Fatalf("save book: %v", err)
	}

	orchestrator := NewHybridOrchestrator(
		PipelineConfig{TTSConcurrency: 1, MinSegmentsBeforeTTS: 5, SegmentationBatchSize: 1},
		repo,
		store,
		&pipelineTestLLMProvider{},
		registry,
	)
	state := newWorkerTestState(book.ID, &types.Segment{
		ID:               "seg_short",
		BookID:           book.ID,
		Text:             "short text",
		Language:         "en",
		Person:           "narrator",
		VoiceDescription: "neutral",
	})
	state.initialMappingDone = false
	state.unmappedPersonas = nil

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		orchestrator.ensureInitialMappingRequested(ctx, state)
		close(done)
	}()

	select {
	case event := <-state.voiceMappingNeeded:
		if !event.IsInitial {
			t.Fatalf("expected initial mapping event")
		}
		if len(event.Personas) != 1 || event.Personas[0] != "narrator" {
			t.Fatalf("expected narrator persona event, got %#v", event.Personas)
		}
	case <-time.After(time.Second):
		t.Fatalf("expected voice mapping event for short book")
	}

	updatedBook, err := repo.GetBook(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("get book: %v", err)
	}
	if updatedBook.Status != "voice_mapping" || !updatedBook.WaitingForMapping {
		t.Fatalf("expected book waiting for voice mapping, got status=%q waiting=%v", updatedBook.Status, updatedBook.WaitingForMapping)
	}

	close(state.initialMappingReceived)
	select {
	case <-done:
	case <-time.After(time.Second):
		cancel()
		t.Fatalf("expected short-book mapping wait to unblock")
	}
}

func TestShortBookUsesDefaultVoiceInsteadOfRequestingInitialMapping(t *testing.T) {
	repo := newPipelineTestRepository()
	store := newPipelineTestStorage()
	registry := provider.NewRegistry()
	if err := registry.RegisterTTS(&pipelineTestTTSProvider{}); err != nil {
		t.Fatalf("register tts provider: %v", err)
	}
	repo.defaultVoice = &types.DefaultVoice{Provider: "pipeline-test-tts", VoiceID: "voice-default", Language: "en"}

	book := &types.Book{ID: "book_short_default", Title: "Short Default", Status: "segmenting"}
	if err := repo.SaveBook(context.Background(), book); err != nil {
		t.Fatalf("save book: %v", err)
	}
	segment := &types.Segment{
		ID:               "seg_short_default",
		BookID:           book.ID,
		Text:             "short text",
		Language:         "en",
		Person:           "narrator",
		VoiceDescription: "neutral",
	}

	orchestrator := NewHybridOrchestrator(
		PipelineConfig{TTSConcurrency: 1, MinSegmentsBeforeTTS: 5, SegmentationBatchSize: 1},
		repo,
		store,
		&pipelineTestLLMProvider{},
		registry,
	)
	state := newWorkerTestState(book.ID, segment)
	state.initialMappingDone = false
	state.unmappedPersonas = nil

	orchestrator.ensureInitialMappingRequested(context.Background(), state)

	select {
	case event := <-state.voiceMappingNeeded:
		t.Fatalf("default voice should avoid short-book mapping event, got %#v", event)
	default:
	}
	select {
	case <-state.initialMappingReceived:
	default:
		t.Fatalf("expected short-book default voice to unblock synthesis")
	}
	if got := state.mappedPersonas["narrator"]; got != "voice-default" {
		t.Fatalf("expected narrator auto-mapped to default voice, got %q", got)
	}
	if got := state.segmentQueue.MappedCount(); got != 1 {
		t.Fatalf("expected short-book segment queued for synthesis, got %d", got)
	}

	voiceMap, err := repo.GetVoiceMap(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("expected persisted short-book voice map: %v", err)
	}
	if len(voiceMap.Persons) != 1 || voiceMap.Persons[0].ID != "narrator" || voiceMap.Persons[0].ProviderVoice != "voice-default" {
		t.Fatalf("unexpected short-book voice map: %#v", voiceMap.Persons)
	}

	updatedBook, err := repo.GetBook(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("get book: %v", err)
	}
	if updatedBook.WaitingForMapping || len(updatedBook.UnmappedPersonas) != 0 || updatedBook.PendingSegmentCount != 0 {
		t.Fatalf("short book should not wait for mapping with default voice, got waiting=%v unmapped=%v pending=%d", updatedBook.WaitingForMapping, updatedBook.UnmappedPersonas, updatedBook.PendingSegmentCount)
	}
	if updatedBook.Status != "synthesizing" {
		t.Fatalf("expected book synthesizing after default fallback, got %q", updatedBook.Status)
	}
}

func TestDefaultVoiceAutoMapsFirstPersonaAndStartsSynthesisWithoutManualMapping(t *testing.T) {
	repo := newPipelineTestRepository()
	store := newPipelineTestStorage()
	registry := provider.NewRegistry()
	if err := registry.RegisterTTS(&pipelineTestTTSProvider{}); err != nil {
		t.Fatalf("register tts provider: %v", err)
	}
	repo.defaultVoice = &types.DefaultVoice{Provider: "pipeline-test-tts", VoiceID: "voice-default", Language: "en"}

	book := &types.Book{ID: "book_default", Title: "Default", Status: "segmenting"}
	if err := repo.SaveBook(context.Background(), book); err != nil {
		t.Fatalf("save book: %v", err)
	}
	segment := &types.Segment{ID: "seg_default", BookID: book.ID, Text: "hello", Language: "en", Person: "narrator", VoiceDescription: "neutral"}
	orchestrator := NewHybridOrchestrator(
		PipelineConfig{TTSConcurrency: 1, MinSegmentsBeforeTTS: 5, SegmentationBatchSize: 1},
		repo,
		store,
		&pipelineTestLLMProvider{},
		registry,
	)
	state := newWorkerTestState(book.ID, segment)
	state.initialMappingDone = false
	state.discoveredPersonas = make(map[string]bool)

	orchestrator.handlePersonaDiscovery(context.Background(), state, segment, 1)

	select {
	case event := <-state.voiceMappingNeeded:
		t.Fatalf("default voice should avoid manual mapping event, got %#v", event)
	default:
	}
	select {
	case <-state.initialMappingReceived:
	case <-time.After(time.Second):
		t.Fatalf("expected default voice auto-mapping to unblock initial synthesis")
	}
	if got := state.mappedPersonas["narrator"]; got != "voice-default" {
		t.Fatalf("expected narrator to be auto-mapped to default voice, got %q", got)
	}
	if !state.initialMappingDone {
		t.Fatalf("expected initial mapping to be considered done after default auto-map")
	}
	if got := state.segmentQueue.MappedCount(); got != 1 {
		t.Fatalf("expected current segment queued for immediate synthesis, got %d mapped segments", got)
	}
	voiceMap, err := repo.GetVoiceMap(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("expected default auto-mapping to persist voice map: %v", err)
	}
	if len(voiceMap.Persons) != 1 || voiceMap.Persons[0].ID != "narrator" || voiceMap.Persons[0].ProviderVoice != "voice-default" {
		t.Fatalf("unexpected persisted voice map after default auto-map: %#v", voiceMap.Persons)
	}

	updatedBook, err := repo.GetBook(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("get book: %v", err)
	}
	if updatedBook.WaitingForMapping {
		t.Fatalf("book should not wait for voice mapping when default voice is available")
	}
	if updatedBook.Status != "synthesizing" {
		t.Fatalf("expected book to move to synthesizing, got %q", updatedBook.Status)
	}
}

func TestDefaultVoiceAutoMapsNewPersonaAfterInitialMapping(t *testing.T) {
	repo := newPipelineTestRepository()
	store := newPipelineTestStorage()
	registry := provider.NewRegistry()
	if err := registry.RegisterTTS(&pipelineTestTTSProvider{}); err != nil {
		t.Fatalf("register tts provider: %v", err)
	}
	repo.defaultVoice = &types.DefaultVoice{Provider: "pipeline-test-tts", VoiceID: "voice-default", Language: "en"}

	book := &types.Book{ID: "book_new_persona", Title: "Default New Persona", Status: "synthesizing"}
	if err := repo.SaveBook(context.Background(), book); err != nil {
		t.Fatalf("save book: %v", err)
	}
	segment := &types.Segment{ID: "seg_alice", BookID: book.ID, Text: "hi", Language: "en", Person: "alice", VoiceDescription: "bright"}
	orchestrator := NewHybridOrchestrator(
		PipelineConfig{TTSConcurrency: 1, MinSegmentsBeforeTTS: 5, SegmentationBatchSize: 1},
		repo,
		store,
		&pipelineTestLLMProvider{},
		registry,
	)
	state := newWorkerTestState(book.ID, segment)
	state.initialMappingDone = true
	state.discoveredPersonas = map[string]bool{"narrator": true}
	state.mappedPersonas = map[string]string{"narrator": "manual-voice"}

	orchestrator.handlePersonaDiscovery(context.Background(), state, segment, 6)

	select {
	case event := <-state.voiceMappingNeeded:
		t.Fatalf("new persona should use default voice without manual mapping event, got %#v", event)
	default:
	}
	if got := state.mappedPersonas["alice"]; got != "voice-default" {
		t.Fatalf("expected new persona alice to be auto-mapped to default voice, got %q", got)
	}
	if state.mappedPersonas["narrator"] != "manual-voice" {
		t.Fatalf("default auto-mapping should not overwrite existing narrator mapping")
	}
	if got := state.segmentQueue.MappedCount(); got != 1 {
		t.Fatalf("expected new persona segment queued for immediate synthesis, got %d", got)
	}
	voiceMap, err := repo.GetVoiceMap(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("expected new persona default mapping to persist voice map: %v", err)
	}
	persisted := make(map[string]string)
	for _, pv := range voiceMap.Persons {
		persisted[pv.ID] = pv.ProviderVoice
	}
	if persisted["narrator"] != "manual-voice" || persisted["alice"] != "voice-default" {
		t.Fatalf("unexpected persisted voice map after new persona default mapping: %#v", persisted)
	}
	updatedBook, err := repo.GetBook(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("get book: %v", err)
	}
	if updatedBook.WaitingForMapping || len(updatedBook.UnmappedPersonas) != 0 {
		t.Fatalf("book should not wait for mapping after default new-persona mapping, waiting=%v unmapped=%v", updatedBook.WaitingForMapping, updatedBook.UnmappedPersonas)
	}
}

func TestStaleQueueWaitsForSegmentationCompletionBeforeRegeneration(t *testing.T) {
	queue := NewSegmentQueue()
	stale := &types.Segment{ID: "seg_stale", Person: "narrator", AudioStale: true}
	fresh := &types.Segment{ID: "seg_fresh", Person: "narrator"}
	queue.EnqueueStale(stale)

	if got := queue.DequeueNext(false); got != nil {
		t.Fatalf("expected stale work to wait while fresh segmentation can still arrive, got %#v", got)
	}
	queue.Enqueue(fresh, true)
	if got := queue.DequeueNext(false); got == nil || got.ID != fresh.ID {
		t.Fatalf("expected fresh work before stale regeneration, got %#v", got)
	}
	if got := queue.DequeueNext(true); got == nil || got.ID != stale.ID {
		t.Fatalf("expected stale regeneration once allowed, got %#v", got)
	}
}

func TestRemapMarksOldAudioStaleAndRegeneratesItAfterFreshSegments(t *testing.T) {
	repo := newPipelineTestRepository()
	store := newPipelineTestStorage()
	ttsProvider := &pipelineTestTTSProvider{}
	registry := provider.NewRegistry()
	if err := registry.RegisterTTS(ttsProvider); err != nil {
		t.Fatalf("register tts provider: %v", err)
	}

	book := &types.Book{ID: "book_remap", Title: "Remap", Status: "synthesizing", TotalSegments: 2, SynthesizedSegments: 1}
	if err := repo.SaveBook(context.Background(), book); err != nil {
		t.Fatalf("save book: %v", err)
	}
	oldAudio := &types.Segment{ID: "seg_old", BookID: book.ID, Text: "old audio", Language: "en", Person: "narrator", VoiceID: "old-voice", Processing: &types.ProcessingInfo{TTSProvider: "pipeline-test-tts", GeneratedAt: time.Now()}}
	fresh := &types.Segment{ID: "seg_fresh", BookID: book.ID, Text: "fresh first", Language: "en", Person: "narrator", Processing: &types.ProcessingInfo{GeneratedAt: time.Now()}}
	if err := repo.SaveSegment(context.Background(), oldAudio); err != nil {
		t.Fatalf("save old segment: %v", err)
	}
	if err := repo.SaveSegment(context.Background(), fresh); err != nil {
		t.Fatalf("save fresh segment: %v", err)
	}

	orchestrator := NewHybridOrchestrator(
		PipelineConfig{TTSConcurrency: 1, MinSegmentsBeforeTTS: 1, SegmentationBatchSize: 1},
		repo,
		store,
		&pipelineTestLLMProvider{},
		registry,
	)
	state := newWorkerTestState(book.ID, oldAudio)
	state.allSegments = []*types.Segment{oldAudio, fresh}
	state.discoveredPersonas = map[string]bool{"narrator": true}
	state.mappedPersonas["narrator"] = "old-voice"
	state.initialMappingDone = true
	state.synthesizedCount = 1
	state.segmentQueue.Enqueue(fresh, true)

	orchestrator.applyVoiceMapping(context.Background(), state, VoiceMappingUpdate{
		VoiceMap:  &types.VoiceMap{BookID: book.ID, Persons: []types.PersonVoice{{ID: "narrator", ProviderVoice: "new-voice"}}},
		IsInitial: false,
	})

	staleSegment, err := repo.GetSegment(context.Background(), book.ID, oldAudio.ID)
	if err != nil {
		t.Fatalf("get stale segment: %v", err)
	}
	if !staleSegment.AudioStale {
		t.Fatalf("expected existing old-voice audio to be marked stale")
	}
	if state.segmentQueue.StaleCount() != 1 {
		t.Fatalf("expected one stale segment queued for deferred regeneration, got %d", state.segmentQueue.StaleCount())
	}
	voiceMap, err := repo.GetVoiceMap(context.Background(), book.ID)
	if err != nil {
		t.Fatalf("expected remap to persist updated voice map: %v", err)
	}
	if len(voiceMap.Persons) != 1 || voiceMap.Persons[0].ID != "narrator" || voiceMap.Persons[0].ProviderVoice != "new-voice" {
		t.Fatalf("unexpected persisted remap: %#v", voiceMap.Persons)
	}

	state.ttsWorkers.Add(1)
	orchestrator.ttsWorker(context.Background(), state, 0)
	orchestrator.completePipeline(state)

	wantOrder := []string{"fresh first:new-voice", "old audio:new-voice"}
	if got := ttsProvider.callOrder(); strings.Join(got, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("expected fresh synthesis before stale regeneration with new voice, got %#v", got)
	}
	updatedOld, err := repo.GetSegment(context.Background(), book.ID, oldAudio.ID)
	if err != nil {
		t.Fatalf("get regenerated old segment: %v", err)
	}
	if updatedOld.AudioStale {
		t.Fatalf("expected stale marker cleared after regeneration")
	}
	if updatedOld.VoiceID != "new-voice" {
		t.Fatalf("expected old segment regenerated with new voice, got %q", updatedOld.VoiceID)
	}
	if state.synthesizedCount != 2 {
		t.Fatalf("expected fresh synthesis count to remain bounded by total segments, got %d", state.synthesizedCount)
	}
}

func TestInFlightOldVoiceSynthesisIsMarkedStaleAfterRemap(t *testing.T) {
	repo := newPipelineTestRepository()
	store := newPipelineTestStorage()
	registry := provider.NewRegistry()
	if err := registry.RegisterTTS(&pipelineTestTTSProvider{}); err != nil {
		t.Fatalf("register tts provider: %v", err)
	}
	book := &types.Book{ID: "book_inflight", Title: "Inflight", Status: "synthesizing", TotalSegments: 1}
	if err := repo.SaveBook(context.Background(), book); err != nil {
		t.Fatalf("save book: %v", err)
	}
	segment := &types.Segment{ID: "seg_inflight", BookID: book.ID, Text: "old inflight", Language: "en", Person: "narrator", Processing: &types.ProcessingInfo{GeneratedAt: time.Now()}}
	orchestrator := NewHybridOrchestrator(
		PipelineConfig{TTSConcurrency: 1, MinSegmentsBeforeTTS: 1, SegmentationBatchSize: 1},
		repo,
		store,
		&pipelineTestLLMProvider{},
		registry,
	)
	state := newWorkerTestState(book.ID, segment)
	state.mappedPersonas["narrator"] = "new-voice"
	state.segmentationComplete = false

	if err := orchestrator.synthesizeSegment(context.Background(), state, segment, "old-voice"); err != nil {
		t.Fatalf("synthesize in-flight old voice: %v", err)
	}
	updated, err := repo.GetSegment(context.Background(), book.ID, segment.ID)
	if err != nil {
		t.Fatalf("get segment: %v", err)
	}
	if !updated.AudioStale || updated.StaleVoiceID != "old-voice" {
		t.Fatalf("expected in-flight old voice result marked stale, got stale=%v staleVoice=%q", updated.AudioStale, updated.StaleVoiceID)
	}
	if state.segmentQueue.StaleCount() != 1 {
		t.Fatalf("expected stale in-flight result queued for regeneration, got %d", state.segmentQueue.StaleCount())
	}
}

func newWorkerTestState(bookID string, segment *types.Segment) *hybridPipelineState {
	return &hybridPipelineState{
		bookID:                 bookID,
		status:                 newPipelineTestStatus(bookID),
		allSegments:            []*types.Segment{segment},
		segmentationComplete:   true,
		discoveredPersonas:     map[string]bool{segment.Person: true},
		mappedPersonas:         make(map[string]string),
		unmappedPersonas:       make([]string, 0),
		segmentQueue:           NewSegmentQueue(),
		voiceMappingNeeded:     make(chan PersonaDiscoveryEvent, 1),
		voiceMappingDone:       make(chan VoiceMappingUpdate, 1),
		initialMappingReceived: make(chan struct{}),
		maxRetries:             defaultSegmentSynthesisMaxRetries,
	}
}

func newPipelineTestStatus(bookID string) *PipelineStatus {
	return &PipelineStatus{
		BookID: bookID,
		Stages: []StageProgress{
			{Stage: "segmenting", Status: "completed", Percentage: 100},
			{Stage: "synthesizing", Status: "in_progress"},
			{Stage: "ready", Status: "pending"},
		},
		UpdatedAt: time.Now(),
	}
}

type pipelineTestTTSProvider struct {
	mu                    sync.Mutex
	calls                 map[string]int
	callRecords           []string
	failuresBeforeSuccess int
	alwaysFail            bool
}

func (p *pipelineTestTTSProvider) Name() string { return "pipeline-test-tts" }

func (p *pipelineTestTTSProvider) Synthesize(ctx context.Context, req provider.TTSRequest) (*provider.TTSResponse, error) {
	p.mu.Lock()
	if p.calls == nil {
		p.calls = make(map[string]int)
	}
	p.calls[req.Text]++
	p.callRecords = append(p.callRecords, fmt.Sprintf("%s:%s", req.Text, req.VoiceID))
	callCount := p.calls[req.Text]
	p.mu.Unlock()

	if p.alwaysFail || callCount <= p.failuresBeforeSuccess {
		return nil, fmt.Errorf("intentional tts failure for %s", req.Text)
	}

	return &provider.TTSResponse{
		AudioData: []byte("audio:" + req.Text),
		Format:    "wav",
	}, nil
}

func (p *pipelineTestTTSProvider) ListVoices(ctx context.Context) ([]provider.Voice, error) {
	return []provider.Voice{{ID: "voice-a", Name: "Voice A", Languages: []string{"en"}}}, nil
}

func (p *pipelineTestTTSProvider) Close() error { return nil }

func (p *pipelineTestTTSProvider) callsFor(text string) int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.calls[text]
}

func (p *pipelineTestTTSProvider) callOrder() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	records := make([]string, len(p.callRecords))
	copy(records, p.callRecords)
	return records
}

type pipelineTestLLMProvider struct{}

func (p *pipelineTestLLMProvider) Name() string { return "pipeline-test-llm" }
func (p *pipelineTestLLMProvider) Segment(ctx context.Context, req provider.SegmentRequest) (*provider.SegmentResponse, error) {
	return &provider.SegmentResponse{Segments: []provider.Segment{{Text: req.Text, Person: "narrator", Language: "en", VoiceDescription: "neutral"}}}, nil
}
func (p *pipelineTestLLMProvider) BatchSegment(ctx context.Context, req provider.BatchSegmentRequest) (*provider.BatchSegmentResponse, error) {
	return &provider.BatchSegmentResponse{}, nil
}
func (p *pipelineTestLLMProvider) Close() error { return nil }

type pipelineTestRepository struct {
	mu           sync.RWMutex
	books        map[string]*types.Book
	segments     map[string]*types.Segment
	voiceMaps    map[string]*types.VoiceMap
	defaultVoice *types.DefaultVoice
}

func newPipelineTestRepository() *pipelineTestRepository {
	return &pipelineTestRepository{books: make(map[string]*types.Book), segments: make(map[string]*types.Segment), voiceMaps: make(map[string]*types.VoiceMap)}
}

func (r *pipelineTestRepository) SaveBook(ctx context.Context, book *types.Book) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *book
	r.books[book.ID] = &copy
	return nil
}

func (r *pipelineTestRepository) GetBook(ctx context.Context, bookID string) (*types.Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	book, ok := r.books[bookID]
	if !ok {
		return nil, fmt.Errorf("book not found: %s", bookID)
	}
	copy := *book
	return &copy, nil
}

func (r *pipelineTestRepository) UpdateBook(ctx context.Context, book *types.Book) error {
	return r.SaveBook(ctx, book)
}

func (r *pipelineTestRepository) ListBooks(ctx context.Context) ([]*types.Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	books := make([]*types.Book, 0, len(r.books))
	for _, book := range r.books {
		copy := *book
		books = append(books, &copy)
	}
	return books, nil
}

func (r *pipelineTestRepository) SaveChapter(ctx context.Context, chapter *types.Chapter) error {
	return nil
}
func (r *pipelineTestRepository) GetChapter(ctx context.Context, bookID, chapterID string) (*types.Chapter, error) {
	return nil, fmt.Errorf("chapter not found")
}
func (r *pipelineTestRepository) ListChapters(ctx context.Context, bookID string) ([]*types.Chapter, error) {
	return nil, nil
}

func (r *pipelineTestRepository) SaveSegment(ctx context.Context, segment *types.Segment) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	copy := *segment
	r.segments[segment.ID] = &copy
	return nil
}
func (r *pipelineTestRepository) GetSegment(ctx context.Context, bookID, segmentID string) (*types.Segment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	segment, ok := r.segments[segmentID]
	if !ok {
		return nil, fmt.Errorf("segment not found: %s", segmentID)
	}
	copy := *segment
	return &copy, nil
}
func (r *pipelineTestRepository) ListSegments(ctx context.Context, bookID string) ([]*types.Segment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	segments := make([]*types.Segment, 0, len(r.segments))
	for _, segment := range r.segments {
		if segment.BookID == bookID {
			copy := *segment
			segments = append(segments, &copy)
		}
	}
	return segments, nil
}

func (r *pipelineTestRepository) SaveVoiceMap(ctx context.Context, voiceMap *types.VoiceMap) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if voiceMap == nil {
		return nil
	}
	copyMap := *voiceMap
	copyMap.Persons = append([]types.PersonVoice(nil), voiceMap.Persons...)
	r.voiceMaps[voiceMap.BookID] = &copyMap
	return nil
}
func (r *pipelineTestRepository) GetVoiceMap(ctx context.Context, bookID string) (*types.VoiceMap, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	voiceMap, ok := r.voiceMaps[bookID]
	if !ok {
		return nil, fmt.Errorf("voice map not found")
	}
	copyMap := *voiceMap
	copyMap.Persons = append([]types.PersonVoice(nil), voiceMap.Persons...)
	return &copyMap, nil
}
func (r *pipelineTestRepository) SaveDefaultVoice(ctx context.Context, setting *types.DefaultVoice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if setting == nil {
		r.defaultVoice = nil
		return nil
	}
	copy := *setting
	r.defaultVoice = &copy
	return nil
}
func (r *pipelineTestRepository) GetDefaultVoice(ctx context.Context) (*types.DefaultVoice, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.defaultVoice == nil {
		return nil, nil
	}
	copy := *r.defaultVoice
	return &copy, nil
}
func (r *pipelineTestRepository) SavePersonaProfiles(ctx context.Context, bookID string, profiles []*types.PersonaProfile) error {
	return nil
}
func (r *pipelineTestRepository) GetPersonaProfiles(ctx context.Context, bookID string) ([]*types.PersonaProfile, error) {
	return []*types.PersonaProfile{}, nil
}
func (r *pipelineTestRepository) UpdatePersonaProfilesFromSegments(ctx context.Context, bookID string, segments []*types.Segment) error {
	return nil
}
func (r *pipelineTestRepository) SaveRawFile(ctx context.Context, bookID string, data []byte, format string) error {
	return nil
}
func (r *pipelineTestRepository) GetRawFile(ctx context.Context, bookID string) ([]byte, string, error) {
	return nil, "", fmt.Errorf("raw file not found")
}
func (r *pipelineTestRepository) DeleteBook(ctx context.Context, bookID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.books, bookID)
	for segmentID, segment := range r.segments {
		if segment.BookID == bookID {
			delete(r.segments, segmentID)
		}
	}
	return nil
}

type pipelineTestStorage struct {
	mu   sync.RWMutex
	data map[string][]byte
}

func newPipelineTestStorage() *pipelineTestStorage {
	return &pipelineTestStorage{data: make(map[string][]byte)}
}

func (s *pipelineTestStorage) Put(ctx context.Context, path string, data io.Reader) error {
	bytes, err := io.ReadAll(data)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data[path] = bytes
	return nil
}
func (s *pipelineTestStorage) Get(ctx context.Context, path string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.data[path]
	if !ok {
		return nil, fmt.Errorf("not found: %s", path)
	}
	return io.NopCloser(strings.NewReader(string(data))), nil
}
func (s *pipelineTestStorage) Delete(ctx context.Context, path string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, path)
	return nil
}
func (s *pipelineTestStorage) Exists(ctx context.Context, path string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.data[path]
	return ok, nil
}
func (s *pipelineTestStorage) List(ctx context.Context, prefix string) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	paths := make([]string, 0, len(s.data))
	for path := range s.data {
		if strings.HasPrefix(path, prefix) {
			paths = append(paths, path)
		}
	}
	return paths, nil
}
func (s *pipelineTestStorage) DeleteAll(ctx context.Context, prefix string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for path := range s.data {
		if strings.HasPrefix(path, prefix) {
			delete(s.data, path)
		}
	}
	return nil
}
func (s *pipelineTestStorage) Close() error { return nil }

var _ book.Repository = (*pipelineTestRepository)(nil)
var _ storage.Adapter = (*pipelineTestStorage)(nil)
var _ provider.LLMProvider = (*pipelineTestLLMProvider)(nil)
var _ provider.TTSProvider = (*pipelineTestTTSProvider)(nil)
