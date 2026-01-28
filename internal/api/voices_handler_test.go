package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/pkg/types"
)

func TestVoicesHandler_ListVoices(t *testing.T) {
	// Create a provider registry with stub providers
	registry := provider.NewRegistry()

	stubConfig := types.TTSProviderConfig{
		Name:    "stub-tts",
		Enabled: true,
		Options: map[string]string{},
	}

	if err := registry.RegisterTTS(provider.NewStubTTSProvider(stubConfig)); err != nil {
		t.Fatalf("Failed to register stub TTS provider: %v", err)
	}

	// Create handler
	handler := NewVoicesHandler(registry)

	// Test GET /api/v1/voices
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voices", nil)
	w := httptest.NewRecorder()

	handler.ListVoices(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check voices array exists
	voicesData, ok := response["voices"].([]interface{})
	if !ok {
		t.Fatal("Expected 'voices' array in response")
	}

	// Stub provider should return at least 1 voice
	if len(voicesData) < 1 {
		t.Errorf("Expected at least 1 voice, got %d", len(voicesData))
	}

	// Check count field
	count, ok := response["count"].(float64)
	if !ok {
		t.Fatal("Expected 'count' field in response")
	}
	if int(count) != len(voicesData) {
		t.Errorf("Count mismatch: count=%d, voices length=%d", int(count), len(voicesData))
	}

	// Check first voice has required fields
	if len(voicesData) > 0 {
		voice := voicesData[0].(map[string]interface{})

		if _, ok := voice["id"]; !ok {
			t.Error("Voice missing 'id' field")
		}
		if _, ok := voice["name"]; !ok {
			t.Error("Voice missing 'name' field")
		}
		if _, ok := voice["languages"]; !ok {
			t.Error("Voice missing 'languages' field")
		}
		if _, ok := voice["provider"]; !ok {
			t.Error("Voice missing 'provider' field")
		}

		provider := voice["provider"].(string)
		if provider != "stub-tts" {
			t.Errorf("Expected provider 'stub-tts', got '%s'", provider)
		}
	}
}

func TestVoicesHandler_ListVoicesWithProvider(t *testing.T) {
	// Create a provider registry with stub providers
	registry := provider.NewRegistry()

	stubConfig := types.TTSProviderConfig{
		Name:    "stub-tts",
		Enabled: true,
		Options: map[string]string{},
	}

	if err := registry.RegisterTTS(provider.NewStubTTSProvider(stubConfig)); err != nil {
		t.Fatalf("Failed to register stub TTS provider: %v", err)
	}

	// Create handler
	handler := NewVoicesHandler(registry)

	// Test GET /api/v1/voices?provider=stub-tts
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voices?provider=stub-tts", nil)
	w := httptest.NewRecorder()

	handler.ListVoices(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check voices array exists
	voicesData, ok := response["voices"].([]interface{})
	if !ok {
		t.Fatal("Expected 'voices' array in response")
	}

	// Should have voices from stub provider
	if len(voicesData) < 1 {
		t.Errorf("Expected at least 1 voice, got %d", len(voicesData))
	}
}

func TestVoicesHandler_ListVoicesWithModel(t *testing.T) {
	// Create a provider registry with stub providers
	registry := provider.NewRegistry()

	stubConfig := types.TTSProviderConfig{
		Name:    "stub-tts",
		Enabled: true,
		Options: map[string]string{},
	}

	if err := registry.RegisterTTS(provider.NewStubTTSProvider(stubConfig)); err != nil {
		t.Fatalf("Failed to register stub TTS provider: %v", err)
	}

	// Create handler
	handler := NewVoicesHandler(registry)

	// Test GET /api/v1/voices?model=tts-1
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voices?model=tts-1", nil)
	w := httptest.NewRecorder()

	handler.ListVoices(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Check voices array exists
	voicesData, ok := response["voices"].([]interface{})
	if !ok {
		t.Fatal("Expected 'voices' array in response")
	}

	// Should have voices from stub provider (stub doesn't filter by model)
	if len(voicesData) < 1 {
		t.Errorf("Expected at least 1 voice, got %d", len(voicesData))
	}
}

func TestVoicesHandler_ListVoicesProviderNotFound(t *testing.T) {
	// Create empty provider registry
	registry := provider.NewRegistry()

	// Create handler
	handler := NewVoicesHandler(registry)

	// Test GET /api/v1/voices?provider=nonexistent
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voices?provider=nonexistent", nil)
	w := httptest.NewRecorder()

	handler.ListVoices(w, req)

	// Should return 404
	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", w.Code)
	}
}

func TestVoicesHandler_ListVoicesNoProviders(t *testing.T) {
	// Create empty provider registry
	registry := provider.NewRegistry()

	// Create handler
	handler := NewVoicesHandler(registry)

	// Test GET /api/v1/voices
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voices", nil)
	w := httptest.NewRecorder()

	handler.ListVoices(w, req)

	// Should return 503 (Service Unavailable)
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", w.Code)
	}
}

func TestVoicesHandler_MethodNotAllowed(t *testing.T) {
	registry := provider.NewRegistry()
	handler := NewVoicesHandler(registry)

	// Test POST /api/v1/voices (should only allow GET)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/voices", nil)
	w := httptest.NewRecorder()

	handler.ListVoices(w, req)

	// Should return 405
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405, got %d", w.Code)
	}
}

// MockFailingTTSProvider is a TTS provider that fails on ListVoices
type MockFailingTTSProvider struct {
	name string
}

func (m *MockFailingTTSProvider) Name() string {
	return m.name
}

func (m *MockFailingTTSProvider) Synthesize(ctx context.Context, req provider.TTSRequest) (*provider.TTSResponse, error) {
	return nil, nil
}

func (m *MockFailingTTSProvider) ListVoices(ctx context.Context, model string) ([]provider.Voice, error) {
	return nil, http.ErrServerClosed
}

func (m *MockFailingTTSProvider) Close() error {
	return nil
}

func TestVoicesHandler_ListVoicesPartialFailure(t *testing.T) {
	// Create a provider registry with one working and one failing provider
	registry := provider.NewRegistry()

	stubConfig := types.TTSProviderConfig{
		Name:    "working-tts",
		Enabled: true,
		Options: map[string]string{},
	}

	if err := registry.RegisterTTS(provider.NewStubTTSProvider(stubConfig)); err != nil {
		t.Fatalf("Failed to register stub TTS provider: %v", err)
	}

	if err := registry.RegisterTTS(&MockFailingTTSProvider{name: "failing-tts"}); err != nil {
		t.Fatalf("Failed to register failing TTS provider: %v", err)
	}

	// Create handler
	handler := NewVoicesHandler(registry)

	// Test GET /api/v1/voices (should succeed with voices from working provider)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/voices", nil)
	w := httptest.NewRecorder()

	handler.ListVoices(w, req)

	// Should still succeed with voices from working provider
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Should have voices from working provider
	voicesData, ok := response["voices"].([]interface{})
	if !ok {
		t.Fatal("Expected 'voices' array in response")
	}

	if len(voicesData) < 1 {
		t.Errorf("Expected at least 1 voice from working provider, got %d", len(voicesData))
	}
}
