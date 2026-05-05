package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/unalkalkan/TwelveReader/internal/provider"
	"github.com/unalkalkan/TwelveReader/internal/storage"
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

func (m *MockFailingTTSProvider) ListVoices(ctx context.Context) ([]provider.Voice, error) {
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

func TestVoicesHandler_PreviewVoice(t *testing.T) {
	registry := provider.NewRegistry()

	stubConfig := types.TTSProviderConfig{
		Name:    "stub-tts",
		Enabled: true,
		Options: map[string]string{},
	}

	if err := registry.RegisterTTS(provider.NewStubTTSProvider(stubConfig)); err != nil {
		t.Fatalf("Failed to register stub TTS provider: %v", err)
	}

	handler := NewVoicesHandler(registry)

	body := map[string]string{
		"provider": "stub-tts",
		"voice_id": "stub-voice-1",
		"text":     "hello world",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voices/preview", bytes.NewReader(jsonBody))
	w := httptest.NewRecorder()

	handler.PreviewVoice(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	audioBase64, ok := response["audio_base64"].(string)
	if !ok || audioBase64 == "" {
		t.Fatal("Expected non-empty audio_base64 in response")
	}

	mimeType, ok := response["mime_type"].(string)
	if !ok || mimeType == "" {
		t.Fatal("Expected mime_type in response")
	}
}

func TestVoicesHandler_PreviewVoiceValidation(t *testing.T) {
	registry := provider.NewRegistry()
	handler := NewVoicesHandler(registry)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/voices/preview", strings.NewReader(`{"voice_id":"v1","text":"hello"}`))
	w := httptest.NewRecorder()

	handler.PreviewVoice(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("Expected status 400, got %d", w.Code)
	}
}

// countingTTSProvider records synthesis calls while returning deterministic audio.
type countingTTSProvider struct {
	name            string
	synthesizeCalls int
	voices          []provider.Voice
}

func (c *countingTTSProvider) Name() string { return c.name }

func (c *countingTTSProvider) Synthesize(ctx context.Context, req provider.TTSRequest) (*provider.TTSResponse, error) {
	c.synthesizeCalls++
	return &provider.TTSResponse{
		AudioData: []byte("CACHED_AUDIO_" + req.VoiceID),
		Format:    "wav",
	}, nil
}

func (c *countingTTSProvider) ListVoices(ctx context.Context) ([]provider.Voice, error) {
	return c.voices, nil
}

func (c *countingTTSProvider) Close() error { return nil }

func TestVoicesHandler_PreviewVoiceCachesSampleAudio(t *testing.T) {
	registry := provider.NewRegistry()
	counting := &countingTTSProvider{
		name: "counting-tts",
		voices: []provider.Voice{{
			ID:          "voice-1",
			Name:        "Voice 1",
			Languages:   []string{"en"},
			Description: "Neutral narrator",
		}},
	}
	if err := registry.RegisterTTS(counting); err != nil {
		t.Fatalf("Failed to register counting TTS provider: %v", err)
	}

	handler := NewVoicesHandlerWithSampleStorage(registry, storage.NewMemorySampleStore())

	body := map[string]string{
		"provider":          "counting-tts",
		"voice_id":          "voice-1",
		"text":              "This request text should not affect the reusable voice sample.",
		"language":          "en",
		"voice_description": "Neutral narrator",
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("Failed to marshal request body: %v", err)
	}

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/voices/preview", bytes.NewReader(jsonBody))
		w := httptest.NewRecorder()
		handler.PreviewVoice(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("Request %d: expected status 200, got %d", i+1, w.Code)
		}
	}

	if counting.synthesizeCalls != 1 {
		t.Fatalf("Expected preview sample to be synthesized once and then served from cache, got %d calls", counting.synthesizeCalls)
	}
}

func TestVoicesHandler_PreGenerateVoiceSamplesStoresEachVoice(t *testing.T) {
	registry := provider.NewRegistry()
	counting := &countingTTSProvider{
		name: "counting-tts",
		voices: []provider.Voice{
			{ID: "voice-1", Name: "Voice 1", Languages: []string{"en"}, Description: "Neutral"},
			{ID: "voice-2", Name: "Voice 2", Languages: []string{"en"}, Description: "Warm"},
		},
	}
	if err := registry.RegisterTTS(counting); err != nil {
		t.Fatalf("Failed to register counting TTS provider: %v", err)
	}

	localAdapter, err := storage.NewLocalAdapter(t.TempDir())
	if err != nil {
		t.Fatalf("Failed to create local storage adapter: %v", err)
	}
	store := storage.NewAdapterSampleStore(localAdapter)
	handler := NewVoicesHandlerWithSampleStorage(registry, store)

	if err := handler.PreGenerateVoiceSamples(context.Background()); err != nil {
		t.Fatalf("PreGenerateVoiceSamples failed: %v", err)
	}

	if counting.synthesizeCalls != 2 {
		t.Fatalf("Expected one startup synthesis per voice, got %d", counting.synthesizeCalls)
	}

	// A fresh handler backed by the same filesystem storage should reuse persisted samples after restart.
	restartedHandler := NewVoicesHandlerWithSampleStorage(registry, storage.NewAdapterSampleStore(localAdapter))
	if err := restartedHandler.PreGenerateVoiceSamples(context.Background()); err != nil {
		t.Fatalf("Second PreGenerateVoiceSamples failed: %v", err)
	}
	if counting.synthesizeCalls != 2 {
		t.Fatalf("Expected persisted startup samples to prevent resynthesis, got %d calls", counting.synthesizeCalls)
	}
}
