package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func TestOpenAITTSProvider_ListVoices(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Verify endpoint
		if r.URL.Path != "/voices" {
			t.Errorf("Expected /voices path, got %s", r.URL.Path)
		}

		// Verify authorization header
		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-api-key" {
			t.Errorf("Expected Authorization header with Bearer token, got %s", auth)
		}

		// Return mock response
		response := voicesAPIResponse{
			Object: "list",
			Data: []voiceData{
				{
					ID:          "alloy",
					Name:        "Alloy",
					Languages:   []string{"en"},
					Gender:      "neutral",
					Description: "A balanced, clear voice",
				},
				{
					ID:          "echo",
					Name:        "Echo",
					Languages:   []string{"en"},
					Gender:      "male",
					Accent:      "american",
					Description: "A confident, professional voice",
				},
				{
					ID:          "fable",
					Name:        "Fable",
					Language:    "en", // Test single language field
					Gender:      "female",
					Accent:      "british",
					Description: "A warm, engaging voice",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with test server URL
	config := types.TTSProviderConfig{
		Name:     "test-openai",
		Endpoint: server.URL,
		APIKey:   "test-api-key",
		Options: map[string]string{
			"model": "tts-1",
		},
	}

	provider, err := NewOpenAITTSProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Call ListVoices
	ctx := context.Background()
	voices, err := provider.ListVoices(ctx)
	if err != nil {
		t.Fatalf("ListVoices failed: %v", err)
	}

	// Verify results
	if len(voices) != 3 {
		t.Errorf("Expected 3 voices, got %d", len(voices))
	}

	// Check first voice
	if voices[0].ID != "alloy" {
		t.Errorf("Expected first voice ID 'alloy', got '%s'", voices[0].ID)
	}
	if voices[0].Name != "Alloy" {
		t.Errorf("Expected first voice name 'Alloy', got '%s'", voices[0].Name)
	}
	if len(voices[0].Languages) != 1 || voices[0].Languages[0] != "en" {
		t.Errorf("Expected languages [en], got %v", voices[0].Languages)
	}

	// Check second voice
	if voices[1].ID != "echo" {
		t.Errorf("Expected second voice ID 'echo', got '%s'", voices[1].ID)
	}
	if voices[1].Accent != "american" {
		t.Errorf("Expected accent 'american', got '%s'", voices[1].Accent)
	}

	// Check third voice (tests Language field fallback)
	if voices[2].ID != "fable" {
		t.Errorf("Expected third voice ID 'fable', got '%s'", voices[2].ID)
	}
	if len(voices[2].Languages) != 1 || voices[2].Languages[0] != "en" {
		t.Errorf("Expected languages [en] from Language field fallback, got %v", voices[2].Languages)
	}
}

func TestOpenAITTSProvider_ListVoicesError(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(ttsAPIErrorResponse{
			Error: struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			}{
				Message: "Invalid API key",
				Type:    "invalid_request_error",
				Code:    "invalid_api_key",
			},
		})
	}))
	defer server.Close()

	// Create provider with test server URL
	config := types.TTSProviderConfig{
		Name:     "test-openai",
		Endpoint: server.URL,
		APIKey:   "invalid-key",
		Options: map[string]string{
			"model": "tts-1",
		},
	}

	provider, err := NewOpenAITTSProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Call ListVoices
	ctx := context.Background()
	_, err = provider.ListVoices(ctx)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Check error message
	expectedMsg := "API error (status 401): Invalid API key"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestStubTTSProvider_ListVoices(t *testing.T) {
	config := types.TTSProviderConfig{
		Name: "test-stub",
	}

	provider := NewStubTTSProvider(config)
	defer provider.Close()

	ctx := context.Background()
	voices, err := provider.ListVoices(ctx)
	if err != nil {
		t.Fatalf("ListVoices failed: %v", err)
	}

	// Stub should return at least one voice
	if len(voices) == 0 {
		t.Error("Expected at least one voice from stub provider")
	}

	// Check first voice has required fields
	if voices[0].ID == "" {
		t.Error("Expected voice ID to be set")
	}
	if voices[0].Name == "" {
		t.Error("Expected voice name to be set")
	}
	if len(voices[0].Languages) == 0 {
		t.Error("Expected at least one language")
	}
}

func TestOpenAITTSProvider_ListVoicesWithConfigModel(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		// Verify endpoint
		if r.URL.Path != "/voices" {
			t.Errorf("Expected /voices path, got %s", r.URL.Path)
		}

		// Verify model query parameter is sent from config
		model := r.URL.Query().Get("model")
		if model != "tts-1-hd" {
			t.Errorf("Expected model=tts-1-hd query parameter from config, got %s", model)
		}

		// Return mock response with voices for the specified model
		response := voicesAPIResponse{
			Object: "list",
			Data: []voiceData{
				{
					ID:          "alloy-hd",
					Name:        "Alloy HD",
					Languages:   []string{"en"},
					Gender:      "neutral",
					Description: "High definition version of Alloy",
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with model in config
	config := types.TTSProviderConfig{
		Name:     "test-openai",
		Endpoint: server.URL,
		APIKey:   "test-api-key",
		Options: map[string]string{
			"model": "tts-1-hd", // Model comes from config
		},
	}

	provider, err := NewOpenAITTSProvider(config)
	if err != nil {
		t.Fatalf("Failed to create provider: %v", err)
	}
	defer provider.Close()

	// Call ListVoices - model comes from provider's config
	ctx := context.Background()
	voices, err := provider.ListVoices(ctx)
	if err != nil {
		t.Fatalf("ListVoices failed: %v", err)
	}

	// Verify results
	if len(voices) != 1 {
		t.Errorf("Expected 1 voice, got %d", len(voices))
	}

	// Check voice
	if voices[0].ID != "alloy-hd" {
		t.Errorf("Expected voice ID 'alloy-hd', got '%s'", voices[0].ID)
	}
}
