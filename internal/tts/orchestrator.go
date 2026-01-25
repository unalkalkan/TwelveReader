package tts

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/book"
	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/internal/storage"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// Orchestrator manages TTS synthesis for book segments
type Orchestrator struct {
	providerReg *provider.Registry
	bookRepo    book.Repository
	storage     storage.Adapter
	concurrency int
}

// NewOrchestrator creates a new TTS orchestrator
func NewOrchestrator(providerReg *provider.Registry, bookRepo book.Repository, storage storage.Adapter, concurrency int) *Orchestrator {
	if concurrency <= 0 {
		concurrency = 3 // Default concurrency
	}
	return &Orchestrator{
		providerReg: providerReg,
		bookRepo:    bookRepo,
		storage:     storage,
		concurrency: concurrency,
	}
}

// SynthesizeBook synthesizes all segments for a book
func (o *Orchestrator) SynthesizeBook(ctx context.Context, bookID string, ttsProviderName string) error {
	// Get book metadata
	book, err := o.bookRepo.GetBook(ctx, bookID)
	if err != nil {
		return fmt.Errorf("failed to get book: %w", err)
	}

	// Check if book is ready for synthesis
	if book.Status != "ready" {
		return fmt.Errorf("book is not ready for synthesis (status: %s)", book.Status)
	}

	// Get voice map
	voiceMap, err := o.bookRepo.GetVoiceMap(ctx, bookID)
	if err != nil {
		return fmt.Errorf("failed to get voice map: %w", err)
	}

	// Get TTS provider
	ttsProvider, err := o.providerReg.GetTTS(ttsProviderName)
	if err != nil {
		return fmt.Errorf("failed to get TTS provider: %w", err)
	}

	// Get all segments
	segments, err := o.bookRepo.ListSegments(ctx, bookID)
	if err != nil {
		return fmt.Errorf("failed to list segments: %w", err)
	}

	if len(segments) == 0 {
		return fmt.Errorf("no segments to synthesize")
	}

	// Update book status to synthesizing
	book.Status = "synthesizing"
	if err := o.bookRepo.UpdateBook(ctx, book); err != nil {
		log.Printf("Failed to update book status: %v", err)
	}

	// Create voice map lookup
	voiceLookup := make(map[string]string)
	for _, pv := range voiceMap.Persons {
		voiceLookup[pv.ID] = pv.ProviderVoice
	}

	// Synthesize segments with concurrency control
	semaphore := make(chan struct{}, o.concurrency)
	var wg sync.WaitGroup
	errCh := make(chan error, len(segments))
	successCount := 0
	var mu sync.Mutex

	for _, seg := range segments {
		wg.Add(1)
		go func(segment *types.Segment) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Synthesize segment
			if err := o.synthesizeSegment(ctx, segment, voiceLookup, ttsProvider); err != nil {
				log.Printf("Failed to synthesize segment %s: %v", segment.ID, err)
				errCh <- err
				return
			}

			mu.Lock()
			successCount++
			mu.Unlock()
		}(seg)
	}

	// Wait for all segments to complete
	wg.Wait()
	close(errCh)

	// Check for errors
	var errors []error
	for err := range errCh {
		errors = append(errors, err)
	}

	// Update book status
	if len(errors) > 0 {
		book.Status = "synthesis_error"
		book.Error = fmt.Sprintf("%d segments failed synthesis", len(errors))
	} else {
		book.Status = "synthesized"
		book.Error = ""
	}

	if err := o.bookRepo.UpdateBook(ctx, book); err != nil {
		log.Printf("Failed to update book status: %v", err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("synthesis completed with %d errors out of %d segments", len(errors), len(segments))
	}

	log.Printf("Successfully synthesized %d segments for book %s", successCount, bookID)
	return nil
}

// synthesizeSegment synthesizes a single segment
func (o *Orchestrator) synthesizeSegment(ctx context.Context, segment *types.Segment, voiceLookup map[string]string, ttsProvider provider.TTSProvider) error {
	// Get voice ID from voice map
	voiceID, ok := voiceLookup[segment.Person]
	if !ok {
		// Use default voice or skip
		log.Printf("No voice mapping found for person %s in segment %s, using default", segment.Person, segment.ID)
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
	audioPath := filepath.Join("books", segment.BookID, "audio", fmt.Sprintf("%s.%s", segment.ID, resp.Format))
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
	if err := o.bookRepo.SaveSegment(ctx, segment); err != nil {
		return fmt.Errorf("failed to update segment: %w", err)
	}

	return nil
}
