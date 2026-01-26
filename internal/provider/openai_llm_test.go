package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func TestNewOpenAILLMProvider(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := types.LLMProviderConfig{
			Name:     "test-openai",
			Enabled:  true,
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "test-key",
			Model:    "gpt-4",
		}

		provider, err := NewOpenAILLMProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		if provider.Name() != "test-openai" {
			t.Errorf("Expected name 'test-openai', got '%s'", provider.Name())
		}
	})

	t.Run("MissingEndpoint", func(t *testing.T) {
		cfg := types.LLMProviderConfig{
			Name:    "test-openai",
			Enabled: true,
			Model:   "gpt-4",
		}

		_, err := NewOpenAILLMProvider(cfg)
		if err == nil {
			t.Error("Expected error for missing endpoint")
		}
	})

	t.Run("MissingModel", func(t *testing.T) {
		cfg := types.LLMProviderConfig{
			Name:     "test-openai",
			Enabled:  true,
			Endpoint: "https://api.openai.com/v1",
		}

		_, err := NewOpenAILLMProvider(cfg)
		if err == nil {
			t.Error("Expected error for missing model")
		}
	})
}

func TestOpenAILLMProvider_Segment(t *testing.T) {
	t.Run("SuccessfulSegmentation", func(t *testing.T) {
		// Create a mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			if r.Method != "POST" {
				t.Errorf("Expected POST request, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/chat/completions") {
				t.Errorf("Expected /chat/completions endpoint, got %s", r.URL.Path)
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-key" {
				t.Errorf("Expected 'Bearer test-key', got '%s'", authHeader)
			}

			// Send mock response
			resp := chatCompletionResponse{
				ID:      "test-id",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4",
				Choices: []choice{
					{
						Index: 0,
						Message: message{
							Role:    "assistant",
							Content: `[{"text": "Hello world", "person": "narrator", "language": "en", "voice_description": "neutral"}]`,
						},
						FinishReason: "stop",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		// Create provider with mock server endpoint
		cfg := types.LLMProviderConfig{
			Name:     "test-openai",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Model:    "gpt-4",
		}

		provider, err := NewOpenAILLMProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		// Test segmentation
		ctx := context.Background()
		req := SegmentRequest{
			Text: "Hello world",
		}

		resp, err := provider.Segment(ctx, req)
		if err != nil {
			t.Fatalf("Segment failed: %v", err)
		}

		if len(resp.Segments) != 1 {
			t.Fatalf("Expected 1 segment, got %d", len(resp.Segments))
		}

		segment := resp.Segments[0]
		if segment.Text != "Hello world" {
			t.Errorf("Expected text 'Hello world', got '%s'", segment.Text)
		}
		if segment.Person != "narrator" {
			t.Errorf("Expected person 'narrator', got '%s'", segment.Person)
		}
		if segment.Language != "en" {
			t.Errorf("Expected language 'en', got '%s'", segment.Language)
		}
	})

	t.Run("APIError", func(t *testing.T) {
		// Create a mock HTTP server that returns an error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			resp := apiErrorResponse{}
			resp.Error.Message = "Invalid API key"
			resp.Error.Type = "invalid_request_error"
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.LLMProviderConfig{
			Name:     "test-openai",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "invalid-key",
			Model:    "gpt-4",
		}

		provider, err := NewOpenAILLMProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := SegmentRequest{
			Text: "Hello world",
		}

		_, err = provider.Segment(ctx, req)
		if err == nil {
			t.Error("Expected error for API failure")
		}
		if !strings.Contains(err.Error(), "Invalid API key") {
			t.Errorf("Expected error to contain 'Invalid API key', got: %v", err)
		}
	})

	t.Run("MultipleSegments", func(t *testing.T) {
		// Create a mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := chatCompletionResponse{
				ID:      "test-id",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4",
				Choices: []choice{
					{
						Index: 0,
						Message: message{
							Role: "assistant",
							Content: `[
								{"text": "Hello", "person": "speaker1", "language": "en", "voice_description": "excited"},
								{"text": "World", "person": "speaker2", "language": "en", "voice_description": "calm"}
							]`,
						},
						FinishReason: "stop",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.LLMProviderConfig{
			Name:     "test-openai",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Model:    "gpt-4",
		}

		provider, err := NewOpenAILLMProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := SegmentRequest{
			Text: "Hello World",
		}

		resp, err := provider.Segment(ctx, req)
		if err != nil {
			t.Fatalf("Segment failed: %v", err)
		}

		if len(resp.Segments) != 2 {
			t.Fatalf("Expected 2 segments, got %d", len(resp.Segments))
		}

		if resp.Segments[0].Person != "speaker1" {
			t.Errorf("Expected person 'speaker1', got '%s'", resp.Segments[0].Person)
		}
		if resp.Segments[1].Person != "speaker2" {
			t.Errorf("Expected person 'speaker2', got '%s'", resp.Segments[1].Person)
		}
	})

	t.Run("NonJSONResponse", func(t *testing.T) {
		// Create a mock HTTP server that returns non-JSON
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			resp := chatCompletionResponse{
				ID:      "test-id",
				Object:  "chat.completion",
				Created: 1234567890,
				Model:   "gpt-4",
				Choices: []choice{
					{
						Index: 0,
						Message: message{
							Role:    "assistant",
							Content: "This is just plain text without JSON",
						},
						FinishReason: "stop",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.LLMProviderConfig{
			Name:     "test-openai",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Model:    "gpt-4",
		}

		provider, err := NewOpenAILLMProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := SegmentRequest{
			Text: "Hello world",
		}

		resp, err := provider.Segment(ctx, req)
		if err != nil {
			t.Fatalf("Segment failed: %v", err)
		}

		// Should fallback to single segment with defaults
		if len(resp.Segments) != 1 {
			t.Fatalf("Expected 1 segment, got %d", len(resp.Segments))
		}
		if resp.Segments[0].Person != "narrator" {
			t.Errorf("Expected fallback person 'narrator', got '%s'", resp.Segments[0].Person)
		}
	})
}

func TestOpenAILLMProvider_Close(t *testing.T) {
	cfg := types.LLMProviderConfig{
		Name:     "test-openai",
		Enabled:  true,
		Endpoint: "https://api.openai.com/v1",
		APIKey:   "test-key",
		Model:    "gpt-4",
	}

	provider, err := NewOpenAILLMProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	err = provider.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestRegistryWithOpenAIProvider(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := chatCompletionResponse{
			ID:      "test-id",
			Object:  "chat.completion",
			Created: 1234567890,
			Model:   "gpt-4",
			Choices: []choice{
				{
					Index: 0,
					Message: message{
						Role:    "assistant",
						Content: `[{"text": "Test", "person": "narrator", "language": "en", "voice_description": "neutral"}]`,
					},
					FinishReason: "stop",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	registry := NewRegistry()

	cfg := types.ProvidersConfig{
		LLM: []types.LLMProviderConfig{
			{
				Name:     "openai",
				Enabled:  true,
				Endpoint: server.URL,
				APIKey:   "test-key",
				Model:    "gpt-4",
			},
			{
				Name:    "stub",
				Enabled: true,
				// No endpoint/model - should use stub
			},
		},
	}

	err := registry.InitializeProviders(cfg)
	if err != nil {
		t.Fatalf("InitializeProviders failed: %v", err)
	}

	// Check that both providers were registered
	llmList := registry.ListLLM()
	if len(llmList) != 2 {
		t.Fatalf("Expected 2 LLM providers, got %d", len(llmList))
	}

	// Test the OpenAI provider
	openaiProvider, err := registry.GetLLM("openai")
	if err != nil {
		t.Fatalf("Failed to get OpenAI provider: %v", err)
	}

	ctx := context.Background()
	req := SegmentRequest{
		Text: "Test text",
	}

	resp, err := openaiProvider.Segment(ctx, req)
	if err != nil {
		t.Fatalf("Segment failed: %v", err)
	}

	if len(resp.Segments) == 0 {
		t.Error("Expected at least one segment")
	}

	// Test the stub provider
	stubProvider, err := registry.GetLLM("stub")
	if err != nil {
		t.Fatalf("Failed to get stub provider: %v", err)
	}

	resp, err = stubProvider.Segment(ctx, req)
	if err != nil {
		t.Fatalf("Stub segment failed: %v", err)
	}

	if len(resp.Segments) == 0 {
		t.Error("Expected at least one segment from stub")
	}
}
