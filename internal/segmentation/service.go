package segmentation

import (
	"context"
	"fmt"
	"time"

	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// Service handles text segmentation using LLM providers
type Service struct {
	llmProvider     provider.LLMProvider
	contextWindow   int // Number of surrounding paragraphs to include
	segmenterVersion string
}

// NewService creates a new segmentation service
func NewService(llmProvider provider.LLMProvider, contextWindow int) *Service {
	return &Service{
		llmProvider:     llmProvider,
		contextWindow:   contextWindow,
		segmenterVersion: "v1",
	}
}

// SegmentChapters processes chapters and generates segments
func (s *Service) SegmentChapters(ctx context.Context, bookID string, chapters []*types.Chapter) ([]*types.Segment, error) {
	segments := make([]*types.Segment, 0)
	segmentCounter := 0
	
	for _, chapter := range chapters {
		chapterSegments, err := s.segmentChapter(ctx, bookID, chapter, &segmentCounter)
		if err != nil {
			return nil, fmt.Errorf("failed to segment chapter %s: %w", chapter.ID, err)
		}
		segments = append(segments, chapterSegments...)
	}
	
	return segments, nil
}

// segmentChapter processes a single chapter
func (s *Service) segmentChapter(ctx context.Context, bookID string, chapter *types.Chapter, counter *int) ([]*types.Segment, error) {
	segments := make([]*types.Segment, 0)
	
	for i, paragraph := range chapter.Paragraphs {
		// Build context
		contextBefore := s.getContext(chapter.Paragraphs, i, -1)
		contextAfter := s.getContext(chapter.Paragraphs, i, 1)
		
		// Call LLM for segmentation
		req := provider.SegmentRequest{
			Text:          paragraph,
			ContextBefore: contextBefore,
			ContextAfter:  contextAfter,
			Language:      "", // Let LLM detect
		}
		
		resp, err := s.llmProvider.Segment(ctx, req)
		if err != nil {
			// If LLM fails, create a fallback segment with the full paragraph
			*counter++
			segment := s.createFallbackSegment(bookID, chapter, paragraph, *counter, i)
			segments = append(segments, segment)
			continue
		}
		
		// Process LLM response segments
		for _, llmSeg := range resp.Segments {
			*counter++
			segment := &types.Segment{
				ID:               fmt.Sprintf("seg_%05d", *counter),
				BookID:           bookID,
				Chapter:          chapter.ID,
				TOCPath:          chapter.TOCPath,
				Text:             llmSeg.Text,
				Language:         llmSeg.Language,
				Person:           llmSeg.Person,
				VoiceDescription: llmSeg.VoiceDescription,
				SourceContext: &types.SourceContext{
					PrevParagraphID: s.getParagraphID(chapter.ID, i-1),
					NextParagraphID: s.getParagraphID(chapter.ID, i+1),
				},
				Processing: &types.ProcessingInfo{
					SegmenterVersion: s.segmenterVersion,
					GeneratedAt:      time.Now(),
				},
			}
			segments = append(segments, segment)
		}
	}
	
	return segments, nil
}

// createFallbackSegment creates a segment when LLM fails
func (s *Service) createFallbackSegment(bookID string, chapter *types.Chapter, text string, counter, paragraphIndex int) *types.Segment {
	return &types.Segment{
		ID:               fmt.Sprintf("seg_%05d", counter),
		BookID:           bookID,
		Chapter:          chapter.ID,
		TOCPath:          chapter.TOCPath,
		Text:             text,
		Language:         "en", // Default to English
		Person:           "narrator",
		VoiceDescription: "neutral",
		SourceContext: &types.SourceContext{
			PrevParagraphID: s.getParagraphID(chapter.ID, paragraphIndex-1),
			NextParagraphID: s.getParagraphID(chapter.ID, paragraphIndex+1),
		},
		Processing: &types.ProcessingInfo{
			SegmenterVersion: s.segmenterVersion,
			GeneratedAt:      time.Now(),
		},
	}
}

// getContext retrieves context paragraphs around the current index
func (s *Service) getContext(paragraphs []string, currentIndex, direction int) []string {
	context := make([]string, 0, s.contextWindow)
	
	if direction < 0 {
		// Get previous paragraphs
		start := currentIndex - s.contextWindow
		if start < 0 {
			start = 0
		}
		for i := start; i < currentIndex; i++ {
			context = append(context, paragraphs[i])
		}
	} else {
		// Get following paragraphs
		end := currentIndex + s.contextWindow + 1
		if end > len(paragraphs) {
			end = len(paragraphs)
		}
		for i := currentIndex + 1; i < end; i++ {
			context = append(context, paragraphs[i])
		}
	}
	
	return context
}

// getParagraphID generates a paragraph ID
func (s *Service) getParagraphID(chapterID string, paragraphIndex int) string {
	if paragraphIndex < 0 {
		return ""
	}
	return fmt.Sprintf("%s_para_%03d", chapterID, paragraphIndex)
}

// DiscoverPersonas extracts unique personas from segments
func DiscoverPersonas(segments []*types.Segment) []string {
	personaMap := make(map[string]bool)
	personas := make([]string, 0)
	
	for _, segment := range segments {
		if segment.Person != "" && !personaMap[segment.Person] {
			personaMap[segment.Person] = true
			personas = append(personas, segment.Person)
		}
	}
	
	return personas
}
