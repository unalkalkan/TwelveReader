package provider

import (
	"context"
	"fmt"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

// StubLLMProvider is a stub implementation of LLMProvider for testing
type StubLLMProvider struct {
	name   string
	config types.LLMProviderConfig
}

// NewStubLLMProvider creates a new stub LLM provider
func NewStubLLMProvider(config types.LLMProviderConfig) *StubLLMProvider {
	return &StubLLMProvider{
		name:   config.Name,
		config: config,
	}
}

func (s *StubLLMProvider) Name() string {
	return s.name
}

func (s *StubLLMProvider) Segment(ctx context.Context, req SegmentRequest) (*SegmentResponse, error) {
	// Stub implementation - returns the input text as a single segment
	return &SegmentResponse{
		Segments: []Segment{
			{
				Text:             req.Text,
				Person:           "narrator",
				Language:         "en",
				VoiceDescription: "neutral",
			},
		},
	}, nil
}

func (s *StubLLMProvider) Close() error {
	return nil
}

// StubTTSProvider is a stub implementation of TTSProvider for testing
type StubTTSProvider struct {
	name   string
	config types.TTSProviderConfig
}

// NewStubTTSProvider creates a new stub TTS provider
func NewStubTTSProvider(config types.TTSProviderConfig) *StubTTSProvider {
	return &StubTTSProvider{
		name:   config.Name,
		config: config,
	}
}

func (s *StubTTSProvider) Name() string {
	return s.name
}

func (s *StubTTSProvider) Synthesize(ctx context.Context, req TTSRequest) (*TTSResponse, error) {
	// Stub implementation - returns empty audio data
	// In a real implementation, this would call the TTS API
	textPreview := req.Text
	if len(textPreview) > 10 {
		textPreview = textPreview[:10]
	}
	return &TTSResponse{
		AudioData: []byte(fmt.Sprintf("STUB_AUDIO_%s", textPreview)),
		Format:    "wav",
		Timestamps: []WordTimestamp{
			{Word: "stub", Start: 0.0, End: 0.5},
		},
	}, nil
}

func (s *StubTTSProvider) ListVoices(ctx context.Context) ([]Voice, error) {
	// Stub implementation - returns a few test voices
	voices := []Voice{
		{
			ID:          "stub-voice-1",
			Name:        "Stub Voice 1",
			Languages:   []string{"en"},
			Gender:      "neutral",
			Description: "A stub voice for testing",
		},
		{
			ID:          "stub-voice-2",
			Name:        "Stub Voice 2",
			Languages:   []string{"en", "es"},
			Gender:      "male",
			Accent:      "american",
			Description: "Another stub voice",
		},
	}
	return voices, nil
}
func (s *StubTTSProvider) Close() error {
	return nil
}

// StubOCRProvider is a stub implementation of OCRProvider for testing
type StubOCRProvider struct {
	name   string
	config types.OCRProviderConfig
}

// NewStubOCRProvider creates a new stub OCR provider
func NewStubOCRProvider(config types.OCRProviderConfig) *StubOCRProvider {
	return &StubOCRProvider{
		name:   config.Name,
		config: config,
	}
}

func (s *StubOCRProvider) Name() string {
	return s.name
}

func (s *StubOCRProvider) ExtractText(ctx context.Context, req OCRRequest) (*OCRResponse, error) {
	// Stub implementation - returns placeholder text
	return &OCRResponse{
		Text:       "Stub OCR extracted text",
		Confidence: 0.95,
	}, nil
}

func (s *StubOCRProvider) Close() error {
	return nil
}
