package segmentation

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

const (
	// DefaultBatchSize is the default number of paragraphs per batch
	DefaultBatchSize = 5
	// MinBatchSize is the minimum batch size when retrying after token errors
	MinBatchSize = 1
)

// ProgressCallback is called to report segmentation progress
type ProgressCallback func(segmentedParagraphs, totalParagraphs int)

// Service handles text segmentation using LLM providers
type Service struct {
	llmProvider      provider.LLMProvider
	contextWindow    int // Number of surrounding paragraphs to include
	segmenterVersion string
	batchSize        int
	knownPersons     []string
	knownPersonMap   map[string]string
}

// NewService creates a new segmentation service
func NewService(llmProvider provider.LLMProvider, contextWindow int) *Service {
	service := &Service{
		llmProvider:      llmProvider,
		contextWindow:    contextWindow,
		segmenterVersion: "v1",
		batchSize:        DefaultBatchSize,
	}
	service.initKnownPersons([]string{"narrator"})
	return service
}

// SetBatchSize sets the batch size for processing
func (s *Service) SetBatchSize(size int) {
	if size < MinBatchSize {
		size = MinBatchSize
	}
	s.batchSize = size
}

// SegmentChapters processes chapters and generates segments
func (s *Service) SegmentChapters(ctx context.Context, bookID string, chapters []*types.Chapter) ([]*types.Segment, error) {
	return s.SegmentChaptersWithProgress(ctx, bookID, chapters, nil)
}

// SegmentChaptersWithProgress processes chapters with progress reporting
func (s *Service) SegmentChaptersWithProgress(ctx context.Context, bookID string, chapters []*types.Chapter, progressCb ProgressCallback) ([]*types.Segment, error) {
	// Calculate total paragraphs
	totalParagraphs := 0
	for _, chapter := range chapters {
		totalParagraphs += len(chapter.Paragraphs)
	}

	segments := make([]*types.Segment, 0)
	segmentCounter := 0
	processedParagraphs := 0

	for _, chapter := range chapters {
		chapterSegments, processed, err := s.segmentChapterBatch(ctx, bookID, chapter, &segmentCounter, processedParagraphs, totalParagraphs, progressCb)
		if err != nil {
			return nil, fmt.Errorf("failed to segment chapter %s: %w", chapter.ID, err)
		}
		segments = append(segments, chapterSegments...)
		processedParagraphs = processed
	}

	// Final progress update
	if progressCb != nil {
		progressCb(processedParagraphs, totalParagraphs)
	}

	return segments, nil
}

// segmentChapterBatch processes a chapter using batch segmentation
func (s *Service) segmentChapterBatch(ctx context.Context, bookID string, chapter *types.Chapter, counter *int, processedSoFar, totalParagraphs int, progressCb ProgressCallback) ([]*types.Segment, int, error) {
	segments := make([]*types.Segment, 0)
	paragraphs := chapter.Paragraphs

	// Process paragraphs in batches
	for i := 0; i < len(paragraphs); {
		batchEnd := i + s.batchSize
		if batchEnd > len(paragraphs) {
			batchEnd = len(paragraphs)
		}

		// Build batch request
		batchReq := s.buildBatchRequest(paragraphs, i, batchEnd)

		// Try batch segmentation with retry on token limit errors
		batchSegments, err := s.processBatchWithRetry(ctx, bookID, chapter, paragraphs, i, batchEnd, counter, batchReq)
		if err != nil {
			return nil, processedSoFar, fmt.Errorf("batch segmentation failed: %w", err)
		}

		segments = append(segments, batchSegments...)

		// Update progress
		processedSoFar += (batchEnd - i)
		if progressCb != nil {
			progressCb(processedSoFar, totalParagraphs)
		}

		i = batchEnd
	}

	return segments, processedSoFar, nil
}

// buildBatchRequest creates a batch request for a range of paragraphs
func (s *Service) buildBatchRequest(paragraphs []string, start, end int) provider.BatchSegmentRequest {
	batchParagraphs := make([]provider.BatchParagraph, 0, end-start)

	for i := start; i < end; i++ {
		contextBefore := s.getContext(paragraphs, i, -1)
		contextAfter := s.getContext(paragraphs, i, 1)

		batchParagraphs = append(batchParagraphs, provider.BatchParagraph{
			Index:         i,
			Text:          paragraphs[i],
			ContextBefore: contextBefore,
			ContextAfter:  contextAfter,
		})
	}

	return provider.BatchSegmentRequest{
		Paragraphs:   batchParagraphs,
		KnownPersons: s.knownPersonsSnapshot(),
	}
}

// processBatchWithRetry attempts batch segmentation with automatic retry on token limit errors
func (s *Service) processBatchWithRetry(ctx context.Context, bookID string, chapter *types.Chapter, paragraphs []string, start, end int, counter *int, req provider.BatchSegmentRequest) ([]*types.Segment, error) {
	currentBatchSize := end - start

	for currentBatchSize >= MinBatchSize {
		// Try batch segmentation
		resp, err := s.llmProvider.BatchSegment(ctx, req)
		if err != nil {
			// Check if it's a token limit error
			if provider.IsTokenLimitError(err) {
				// Reduce batch size and retry
				currentBatchSize = currentBatchSize / 2
				if currentBatchSize < MinBatchSize {
					currentBatchSize = MinBatchSize
				}

				log.Printf("[Segmentation] Token limit error, reducing batch size to %d and retrying", currentBatchSize)

				// If we're already at minimum and still failing, fall back to single paragraph processing
				if currentBatchSize == MinBatchSize && len(req.Paragraphs) == 1 {
					log.Printf("[Segmentation] Token limit at minimum batch, using fallback for paragraph %d", start)
					return s.processSingleParagraphFallback(bookID, chapter, paragraphs[start], counter, start), nil
				}

				// Rebuild request with smaller batch
				newEnd := start + currentBatchSize
				if newEnd > end {
					newEnd = end
				}
				req = s.buildBatchRequest(paragraphs, start, newEnd)
				continue
			}

			// For other errors, fall back to single paragraph processing
			log.Printf("[Segmentation] Batch error: %v, falling back to single paragraph processing", err)
			return s.processParagraphsIndividually(ctx, bookID, chapter, paragraphs, start, end, counter), nil
		}

		// Process successful batch response
		return s.convertBatchResults(bookID, chapter, resp.Results, counter, paragraphs), nil
	}

	// Should not reach here, but fallback just in case
	return s.processParagraphsIndividually(ctx, bookID, chapter, paragraphs, start, end, counter), nil
}

// processParagraphsIndividually processes paragraphs one at a time (fallback)
func (s *Service) processParagraphsIndividually(ctx context.Context, bookID string, chapter *types.Chapter, paragraphs []string, start, end int, counter *int) []*types.Segment {
	segments := make([]*types.Segment, 0)

	for i := start; i < end; i++ {
		contextBefore := s.getContext(paragraphs, i, -1)
		contextAfter := s.getContext(paragraphs, i, 1)

		req := provider.SegmentRequest{
			Text:          paragraphs[i],
			ContextBefore: contextBefore,
			ContextAfter:  contextAfter,
			KnownPersons:  s.knownPersonsSnapshot(),
		}

		resp, err := s.llmProvider.Segment(ctx, req)
		if err != nil {
			// Create fallback segment
			segments = append(segments, s.processSingleParagraphFallback(bookID, chapter, paragraphs[i], counter, i)...)
			continue
		}

		// Convert response to segments
		for _, llmSeg := range resp.Segments {
			*counter++
			person := s.registerPerson(llmSeg.Person)
			segment := &types.Segment{
				ID:               fmt.Sprintf("seg_%05d", *counter),
				BookID:           bookID,
				Chapter:          chapter.ID,
				TOCPath:          chapter.TOCPath,
				Text:             llmSeg.Text,
				Language:         llmSeg.Language,
				Person:           person,
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

	return segments
}

// processSingleParagraphFallback creates a fallback segment for a single paragraph
func (s *Service) processSingleParagraphFallback(bookID string, chapter *types.Chapter, text string, counter *int, paragraphIndex int) []*types.Segment {
	*counter++
	s.registerPerson("narrator")
	return []*types.Segment{
		{
			ID:               fmt.Sprintf("seg_%05d", *counter),
			BookID:           bookID,
			Chapter:          chapter.ID,
			TOCPath:          chapter.TOCPath,
			Text:             text,
			Language:         "en",
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
		},
	}
}

// convertBatchResults converts batch results to segments
func (s *Service) convertBatchResults(bookID string, chapter *types.Chapter, results []provider.BatchParagraphResult, counter *int, paragraphs []string) []*types.Segment {
	segments := make([]*types.Segment, 0)

	for _, result := range results {
		paragraphIndex := result.ParagraphIndex

		for _, llmSeg := range result.Segments {
			*counter++
			person := s.registerPerson(llmSeg.Person)
			segment := &types.Segment{
				ID:               fmt.Sprintf("seg_%05d", *counter),
				BookID:           bookID,
				Chapter:          chapter.ID,
				TOCPath:          chapter.TOCPath,
				Text:             llmSeg.Text,
				Language:         llmSeg.Language,
				Person:           person,
				VoiceDescription: llmSeg.VoiceDescription,
				SourceContext: &types.SourceContext{
					PrevParagraphID: s.getParagraphID(chapter.ID, paragraphIndex-1),
					NextParagraphID: s.getParagraphID(chapter.ID, paragraphIndex+1),
				},
				Processing: &types.ProcessingInfo{
					SegmenterVersion: s.segmenterVersion,
					GeneratedAt:      time.Now(),
				},
			}
			segments = append(segments, segment)
		}
	}

	return segments
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

func (s *Service) initKnownPersons(persons []string) {
	s.knownPersonMap = make(map[string]string)
	s.knownPersons = make([]string, 0, len(persons))
	for _, person := range persons {
		s.registerPerson(person)
	}
}

func (s *Service) knownPersonsSnapshot() []string {
	if len(s.knownPersons) == 0 {
		return nil
	}
	known := make([]string, len(s.knownPersons))
	copy(known, s.knownPersons)
	return known
}

func (s *Service) registerPerson(person string) string {
	person = strings.TrimSpace(person)
	if person == "" {
		return person
	}
	if s.knownPersonMap == nil {
		s.knownPersonMap = make(map[string]string)
	}
	normalized := normalizePersonKey(person)
	if normalized == "" {
		return person
	}
	if existing, ok := s.knownPersonMap[normalized]; ok {
		return existing
	}
	s.knownPersonMap[normalized] = person
	s.knownPersons = append(s.knownPersons, person)
	return person
}

var personQualifierTokens = map[string]bool{
	"thought":   true,
	"spoken":    true,
	"inner":     true,
	"fantasy":   true,
	"quoted":    true,
	"exclaimed": true,
}

func normalizePersonKey(person string) string {
	person = strings.TrimSpace(person)
	if person == "" {
		return ""
	}

	var b strings.Builder
	lastSpace := false
	for _, r := range person {
		switch {
		case r == '(':
			lastSpace = true
		case r == ')':
			lastSpace = true
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(unicode.ToLower(r))
			lastSpace = false
		default:
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
		}
	}

	normalized := strings.TrimSpace(b.String())
	if normalized == "" {
		return ""
	}

	tokens := strings.Fields(normalized)
	if len(tokens) == 0 {
		return ""
	}
	if len(tokens) > 1 && tokens[0] == "character" {
		tokens = tokens[1:]
	}
	for len(tokens) > 0 {
		if personQualifierTokens[tokens[len(tokens)-1]] {
			tokens = tokens[:len(tokens)-1]
			continue
		}
		break
	}
	if len(tokens) == 0 {
		return ""
	}
	return strings.Join(tokens, " ")
}
