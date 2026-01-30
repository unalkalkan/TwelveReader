package pipeline

import (
	"sync"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// SegmentQueue manages segments with priority based on voice mapping status
type SegmentQueue struct {
	mappedQueue   []*types.Segment // Segments with mapped voices, ready for TTS
	unmappedQueue []*types.Segment // Segments waiting for voice mapping
	mu            sync.RWMutex
}

// NewSegmentQueue creates a new segment queue
func NewSegmentQueue() *SegmentQueue {
	return &SegmentQueue{
		mappedQueue:   make([]*types.Segment, 0),
		unmappedQueue: make([]*types.Segment, 0),
	}
}

// Enqueue adds a segment to the appropriate queue
func (sq *SegmentQueue) Enqueue(segment *types.Segment, isMapped bool) {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	if isMapped {
		sq.mappedQueue = append(sq.mappedQueue, segment)
	} else {
		sq.unmappedQueue = append(sq.unmappedQueue, segment)
	}
}

// DequeueNext returns the next segment ready for TTS, or nil if none available
func (sq *SegmentQueue) DequeueNext() *types.Segment {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	if len(sq.mappedQueue) == 0 {
		return nil
	}

	// Dequeue from front
	segment := sq.mappedQueue[0]
	sq.mappedQueue = sq.mappedQueue[1:]
	return segment
}

// PromotePendingSegments moves segments with the given persona from unmapped to mapped queue
// Returns the number of segments promoted
func (sq *SegmentQueue) PromotePendingSegments(persona string) int {
	sq.mu.Lock()
	defer sq.mu.Unlock()

	promoted := 0
	remaining := make([]*types.Segment, 0)

	// Find segments with this persona
	toPromote := make([]*types.Segment, 0)
	for _, segment := range sq.unmappedQueue {
		if segment.Person == persona {
			toPromote = append(toPromote, segment)
			promoted++
		} else {
			remaining = append(remaining, segment)
		}
	}

	// Update unmapped queue
	sq.unmappedQueue = remaining

	// Add promoted segments to the FRONT of mapped queue (priority)
	sq.mappedQueue = append(toPromote, sq.mappedQueue...)

	return promoted
}

// UnmappedCount returns the number of segments waiting for voice mapping
func (sq *SegmentQueue) UnmappedCount() int {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	return len(sq.unmappedQueue)
}

// MappedCount returns the number of segments ready for TTS
func (sq *SegmentQueue) MappedCount() int {
	sq.mu.RLock()
	defer sq.mu.RUnlock()
	return len(sq.mappedQueue)
}

// GetUnmappedPersonas returns the unique list of personas in the unmapped queue
func (sq *SegmentQueue) GetUnmappedPersonas() []string {
	sq.mu.RLock()
	defer sq.mu.RUnlock()

	personaMap := make(map[string]bool)
	for _, segment := range sq.unmappedQueue {
		personaMap[segment.Person] = true
	}

	personas := make([]string, 0, len(personaMap))
	for persona := range personaMap {
		personas = append(personas, persona)
	}
	return personas
}

// Close signals that no more segments will be added
func (sq *SegmentQueue) Close() {
	// For future use if we add channels
}
