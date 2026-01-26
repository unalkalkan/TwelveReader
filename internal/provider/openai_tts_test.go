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

func TestNewOpenAITTSProvider(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := types.TTSProviderConfig{
			Name:     "test-openai-tts",
			Enabled:  true,
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o-mini-tts",
			},
		}

		provider, err := NewOpenAITTSProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		if provider.Name() != "test-openai-tts" {
			t.Errorf("Expected name 'test-openai-tts', got '%s'", provider.Name())
		}

		if provider.model != "gpt-4o-mini-tts" {
			t.Errorf("Expected model 'gpt-4o-mini-tts', got '%s'", provider.model)
		}
	})

	t.Run("MissingEndpoint", func(t *testing.T) {
		cfg := types.TTSProviderConfig{
			Name:    "test-openai-tts",
			Enabled: true,
			Options: map[string]string{
				"model": "gpt-4o-mini-tts",
			},
		}

		_, err := NewOpenAITTSProvider(cfg)
		if err == nil {
			t.Error("Expected error for missing endpoint")
		}
		if !strings.Contains(err.Error(), "endpoint is required") {
			t.Errorf("Expected error about endpoint, got: %v", err)
		}
	})

	t.Run("MissingModel", func(t *testing.T) {
		cfg := types.TTSProviderConfig{
			Name:     "test-openai-tts",
			Enabled:  true,
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "test-key",
			Options:  map[string]string{},
		}

		_, err := NewOpenAITTSProvider(cfg)
		if err == nil {
			t.Error("Expected error for missing model")
		}
		if !strings.Contains(err.Error(), "model is required") {
			t.Errorf("Expected error about model, got: %v", err)
		}
	})

	t.Run("CustomTimeout", func(t *testing.T) {
		cfg := types.TTSProviderConfig{
			Name:     "test-openai-tts",
			Enabled:  true,
			Endpoint: "https://api.openai.com/v1",
			APIKey:   "test-key",
			Options: map[string]string{
				"model":   "gpt-4o-mini-tts",
				"timeout": "60",
			},
		}

		provider, err := NewOpenAITTSProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		if provider.httpClient.Timeout.Seconds() != 60 {
			t.Errorf("Expected timeout 60s, got %v", provider.httpClient.Timeout.Seconds())
		}
	})
}

func TestOpenAITTSProvider_Synthesize(t *testing.T) {
	t.Run("SuccessfulSynthesis", func(t *testing.T) {
		// Create a mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			if r.Method != "POST" {
				t.Errorf("Expected POST request, got %s", r.Method)
			}
			if !strings.HasSuffix(r.URL.Path, "/audio/speech") {
				t.Errorf("Expected /audio/speech endpoint, got %s", r.URL.Path)
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader != "Bearer test-key" {
				t.Errorf("Expected 'Bearer test-key', got '%s'", authHeader)
			}

			// Parse request body
			var reqBody ttsAPIRequest
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Errorf("Failed to decode request: %v", err)
			}

			// Verify request fields
			if reqBody.Model != "gpt-4o-mini-tts" {
				t.Errorf("Expected model 'gpt-4o-mini-tts', got '%s'", reqBody.Model)
			}
			if reqBody.Input != "Hello, world!" {
				t.Errorf("Expected input 'Hello, world!', got '%s'", reqBody.Input)
			}
			if reqBody.Voice != "coral" {
				t.Errorf("Expected voice 'coral', got '%s'", reqBody.Voice)
			}
			if reqBody.Instructions != "cheerful and positive" {
				t.Errorf("Expected instructions 'cheerful and positive', got '%s'", reqBody.Instructions)
			}

			// Send mock MP3 data
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write([]byte("MOCK_MP3_DATA"))
		}))
		defer server.Close()

		// Create provider with mock server endpoint
		cfg := types.TTSProviderConfig{
			Name:     "test-openai-tts",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o-mini-tts",
			},
		}

		provider, err := NewOpenAITTSProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		// Test synthesis
		ctx := context.Background()
		req := TTSRequest{
			Text:             "Hello, world!",
			VoiceID:          "coral",
			Language:         "en",
			VoiceDescription: "cheerful and positive",
		}

		resp, err := provider.Synthesize(ctx, req)
		if err != nil {
			t.Fatalf("Synthesize failed: %v", err)
		}

		if string(resp.AudioData) != "MOCK_MP3_DATA" {
			t.Errorf("Expected audio data 'MOCK_MP3_DATA', got '%s'", string(resp.AudioData))
		}
		if resp.Format != "mp3" {
			t.Errorf("Expected format 'mp3', got '%s'", resp.Format)
		}
	})

	t.Run("SynthesisWithoutInstructions", func(t *testing.T) {
		// Create a mock HTTP server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Parse request body
			var reqBody ttsAPIRequest
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Errorf("Failed to decode request: %v", err)
			}

			// Verify instructions are empty
			if reqBody.Instructions != "" {
				t.Errorf("Expected empty instructions, got '%s'", reqBody.Instructions)
			}

			// Send mock MP3 data
			w.Header().Set("Content-Type", "audio/mpeg")
			w.Write([]byte("MOCK_MP3_DATA"))
		}))
		defer server.Close()

		cfg := types.TTSProviderConfig{
			Name:     "test-openai-tts",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o-mini-tts",
			},
		}

		provider, err := NewOpenAITTSProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := TTSRequest{
			Text:             "Hello, world!",
			VoiceID:          "coral",
			Language:         "en",
			VoiceDescription: "", // Empty description
		}

		_, err = provider.Synthesize(ctx, req)
		if err != nil {
			t.Fatalf("Synthesize failed: %v", err)
		}
	})

	t.Run("APIError", func(t *testing.T) {
		// Create a mock HTTP server that returns an error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			resp := ttsAPIErrorResponse{}
			resp.Error.Message = "Invalid API key"
			resp.Error.Type = "invalid_request_error"
			json.NewEncoder(w).Encode(resp)
		}))
		defer server.Close()

		cfg := types.TTSProviderConfig{
			Name:     "test-openai-tts",
			Enabled:  true,
			Endpoint: server.URL,
			APIKey:   "invalid-key",
			Options: map[string]string{
				"model": "gpt-4o-mini-tts",
			},
		}

		provider, err := NewOpenAITTSProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := TTSRequest{
			Text:    "Hello, world!",
			VoiceID: "coral",
		}

		_, err = provider.Synthesize(ctx, req)
		if err == nil {
			t.Error("Expected error for API failure")
		}
		if !strings.Contains(err.Error(), "Invalid API key") {
			t.Errorf("Expected error to contain 'Invalid API key', got: %v", err)
		}
	})

	t.Run("NetworkError", func(t *testing.T) {
		cfg := types.TTSProviderConfig{
			Name:     "test-openai-tts",
			Enabled:  true,
			Endpoint: "http://localhost:99999", // Invalid endpoint
			APIKey:   "test-key",
			Options: map[string]string{
				"model": "gpt-4o-mini-tts",
			},
		}

		provider, err := NewOpenAITTSProvider(cfg)
		if err != nil {
			t.Fatalf("Failed to create provider: %v", err)
		}

		ctx := context.Background()
		req := TTSRequest{
			Text:    "Hello, world!",
			VoiceID: "coral",
		}

		_, err = provider.Synthesize(ctx, req)
		if err == nil {
			t.Error("Expected error for network failure")
		}
	})
}

func TestOpenAITTSProvider_Close(t *testing.T) {
	cfg := types.TTSProviderConfig{
		Name:     "test-openai-tts",
		Enabled:  true,
		Endpoint: "https://api.openai.com/v1",
		APIKey:   "test-key",
		Options: map[string]string{
			"model": "gpt-4o-mini-tts",
		},
	}

	provider, err := NewOpenAITTSProvider(cfg)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}

	err = provider.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestRegistryWithOpenAITTSProvider(t *testing.T) {
	// Create a mock HTTP server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "audio/mpeg")
		w.Write([]byte("MOCK_MP3_DATA"))
	}))
	defer server.Close()

	registry := NewRegistry()

	cfg := types.ProvidersConfig{
		TTS: []types.TTSProviderConfig{
			{
				Name:     "openai-tts",
				Enabled:  true,
				Endpoint: server.URL,
				APIKey:   "test-key",
				Options: map[string]string{
					"model": "gpt-4o-mini-tts",
				},
			},
			{
				Name:    "stub-tts",
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
	ttsList := registry.ListTTS()
	if len(ttsList) != 2 {
		t.Fatalf("Expected 2 TTS providers, got %d", len(ttsList))
	}

	// Test the OpenAI TTS provider
	openaiProvider, err := registry.GetTTS("openai-tts")
	if err != nil {
		t.Fatalf("Failed to get OpenAI TTS provider: %v", err)
	}

	ctx := context.Background()
	req := TTSRequest{
		Text:    "Test text",
		VoiceID: "coral",
	}

	resp, err := openaiProvider.Synthesize(ctx, req)
	if err != nil {
		t.Fatalf("Synthesize failed: %v", err)
	}

	if len(resp.AudioData) == 0 {
		t.Error("Expected audio data")
	}

	// Test the stub provider
	stubProvider, err := registry.GetTTS("stub-tts")
	if err != nil {
		t.Fatalf("Failed to get stub provider: %v", err)
	}

	resp, err = stubProvider.Synthesize(ctx, req)
	if err != nil {
		t.Fatalf("Stub synthesize failed: %v", err)
	}

	if len(resp.AudioData) == 0 {
		t.Error("Expected audio data from stub")
	}
}
