package provider

import (
	"context"
)

// LLMProvider defines the interface for LLM providers
type LLMProvider interface {
	// Name returns the provider name
	Name() string

	// Segment calls the LLM to segment text and extract speaker information
	Segment(ctx context.Context, req SegmentRequest) (*SegmentResponse, error)

	// Close cleans up resources
	Close() error
}

// SegmentRequest contains the text and context for segmentation
type SegmentRequest struct {
	Text          string   // Text to segment
	ContextBefore []string // Previous paragraphs for context
	ContextAfter  []string // Following paragraphs for context
	Language      string   // Optional language hint
}

// SegmentResponse contains the segmentation results
type SegmentResponse struct {
	Segments []Segment // Identified segments
}

// Segment represents a single text segment with metadata
type Segment struct {
	Text             string // Segment text
	Person           string // Speaker identifier
	Language         string // ISO-639-1 language code
	VoiceDescription string // Voice/tone description
}

// TTSProvider defines the interface for TTS providers
type TTSProvider interface {
	// Name returns the provider name
	Name() string

	// Synthesize converts text to speech
	Synthesize(ctx context.Context, req TTSRequest) (*TTSResponse, error)

	// Close cleans up resources
	Close() error
}

// TTSRequest contains the text and voice settings for synthesis
type TTSRequest struct {
	Text             string // Text to synthesize
	VoiceID          string // Provider-specific voice ID
	Language         string // ISO-639-1 language code
	VoiceDescription string // Optional voice/tone description
}

// TTSResponse contains the synthesized audio and metadata
type TTSResponse struct {
	AudioData  []byte          // Audio file data
	Format     string          // Audio format (e.g., "wav", "mp3")
	Timestamps []WordTimestamp // Word-level timestamps if available
}

// WordTimestamp represents timing information for a word
type WordTimestamp struct {
	Word  string  // The word
	Start float64 // Start time in seconds
	End   float64 // End time in seconds
}

// OCRProvider defines the interface for OCR providers
type OCRProvider interface {
	// Name returns the provider name
	Name() string

	// ExtractText extracts text from an image
	ExtractText(ctx context.Context, req OCRRequest) (*OCRResponse, error)

	// Close cleans up resources
	Close() error
}

// OCRRequest contains the image data for OCR
type OCRRequest struct {
	ImageData []byte // Image file data
	Language  string // Optional language hint
}

// OCRResponse contains the extracted text
type OCRResponse struct {
	Text       string  // Extracted text
	Confidence float64 // OCR confidence score (0-1)
}
