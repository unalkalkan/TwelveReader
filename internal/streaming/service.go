package streaming

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/unalkalkan/TwelveReader/internal/book"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// Service handles streaming of book segments
type Service struct {
	bookRepo book.Repository
}

// NewService creates a new streaming service
func NewService(bookRepo book.Repository) *Service {
	return &Service{
		bookRepo: bookRepo,
	}
}

// StreamItem represents a single item in the NDJSON stream
type StreamItem struct {
	*types.Segment
	AudioURL string `json:"audio_url"`
}

// StreamSegments returns segments as NDJSON for streaming playback
func (s *Service) StreamSegments(ctx context.Context, bookID string, afterSegmentID string) ([]StreamItem, error) {
	// Get segments
	segments, err := s.bookRepo.ListSegments(ctx, bookID)
	if err != nil {
		return nil, fmt.Errorf("failed to list segments: %w", err)
	}

	// Filter segments if afterSegmentID is provided
	var filteredSegments []*types.Segment
	if afterSegmentID != "" {
		found := false
		for _, seg := range segments {
			if found {
				filteredSegments = append(filteredSegments, seg)
			} else if seg.ID == afterSegmentID {
				found = true
			}
		}
	} else {
		filteredSegments = segments
	}

	// Build stream items
	items := make([]StreamItem, 0, len(filteredSegments))
	for _, seg := range filteredSegments {
		// Generate audio URL path
		audioURL := s.getAudioURL(bookID, seg.ID)

		item := StreamItem{
			Segment:  seg,
			AudioURL: audioURL,
		}
		items = append(items, item)
	}

	return items, nil
}

// getAudioURL generates the audio URL for a segment
func (s *Service) getAudioURL(bookID, segmentID string) string {
	// In production, this would be a signed URL or CDN URL
	// For now, we return a relative path using forward slashes for URLs
	return fmt.Sprintf("/api/v1/books/%s/audio/%s", bookID, segmentID)
}

// EncodeNDJSON encodes stream items as NDJSON
func EncodeNDJSON(items []StreamItem) (string, error) {
	var result string
	for _, item := range items {
		jsonData, err := json.Marshal(item)
		if err != nil {
			return "", fmt.Errorf("failed to marshal item: %w", err)
		}
		result += string(jsonData) + "\n"
	}
	return result, nil
}
